package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

// handleGitHubLogin starts the GitHub OAuth flow (Authorization Code + state).
func (h *Hub) handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	if h.githubID == "" || h.githubSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "github oauth not configured")
		return
	}
	device := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("device")))
	opener := r.URL.Query().Get("opener")
	if opener != "" && !allowedOpener(opener) {
		writeError(w, http.StatusBadRequest, "invalid opener")
		return
	}
	lang := r.URL.Query().Get("lang")
	if lang != "zh" && lang != "en" {
		lang = "zh"
	}
	h.mu.Lock()
	if device != "" {
		d, ok := h.deviceLogins[device]
		if !ok || d.Ready || time.Now().After(d.ExpiresAt) {
			h.mu.Unlock()
			if lang == "en" {
				writeHTML(w, http.StatusBadRequest, errorHTML(lang, "Code expired", "This authorization code is no longer valid. Run riffpad login again for a fresh code."))
			} else {
				writeHTML(w, http.StatusBadRequest, errorHTML(lang, "授权码已失效", "该授权码已失效，请重新运行 riffpad login 获取新授权码。"))
			}
			return
		}
	}
	// GitHub echoes only the state parameter on the callback, so the language
	// is embedded in it: "<state>.<lang>".
	state := protocol.NewID() + "." + lang
	h.oauthStates[state] = oauthState{expires: time.Now().Add(10 * time.Minute), device: device, opener: opener, lang: lang}
	h.mu.Unlock()
	redirect, _ := url.Parse("https://github.com/login/oauth/authorize")
	q := redirect.Query()
	q.Set("client_id", h.githubID)
	q.Set("redirect_uri", "https://api.riffpad.ai/api/auth/github/callback")
	q.Set("scope", "read:user")
	q.Set("state", state)
	redirect.RawQuery = q.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

// handleGitHubCallback exchanges the code for a token, resolves the GitHub
// user, and hands the Riffpad token back to the opener via postMessage.
func (h *Hub) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if h.githubID == "" || h.githubSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "github oauth not configured")
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		lang := oauthLangFromState(state)
		if lang == "" {
			lang = oauthLang(r)
		}
		if lang == "en" {
			writeHTML(w, http.StatusBadRequest, errorHTML(lang, "Sign-in failed", "The link is incomplete — please start again."))
		} else {
			writeHTML(w, http.StatusBadRequest, errorHTML(lang, "登录失败", "链接不完整，请重新发起登录。"))
		}
		return
	}
	h.mu.Lock()
	st, ok := h.oauthStates[state]
	if ok {
		delete(h.oauthStates, state)
	}
	h.mu.Unlock()
	if !ok || time.Now().After(st.expires) {
		lang := oauthLangFromState(state)
		if lang == "" {
			lang = oauthLang(r)
		}
		if lang == "en" {
			writeHTML(w, http.StatusUnauthorized, errorHTML(lang, "Link expired", "The authorization flow expired. Run riffpad login again for a fresh link."))
		} else {
			writeHTML(w, http.StatusUnauthorized, errorHTML(lang, "链接已失效", "授权流程已过期，请重新运行 riffpad login 获取新链接。"))
		}
		return
	}
	// Exchange code for an access token.
	form := url.Values{}
	form.Set("client_id", h.githubID)
	form.Set("client_secret", h.githubSecret)
	form.Set("code", code)
	form.Set("redirect_uri", "https://api.riffpad.ai/api/auth/github/callback")
	tokenReq, err := http.NewRequest(http.MethodPost, h.githubTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "github token exchange failed")
		return
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("Accept", "application/json")
	tokenResp, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		writeOAuthError(w, st.lang, "GitHub sign-in failed", "GitHub did not complete the sign-in. Please try again.", "GitHub 登录失败", "GitHub 未能完成登录，请稍后重试。")
		return
	}
	defer tokenResp.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tok); err != nil || tok.AccessToken == "" {
		h.log.Printf("github token exchange error: %s", tok.Error)
		if tok.Error == "access_denied" {
			writeOAuthError(w, st.lang, "Authorization canceled", "You declined the GitHub authorization. Run riffpad login again and choose Authorize.", "授权已取消", "你拒绝了 GitHub 授权。请重新运行 riffpad login 并选择 Authorize。")
		} else {
			writeOAuthError(w, st.lang, "GitHub sign-in failed", "GitHub did not complete the sign-in. Please try again.", "GitHub 登录失败", "GitHub 未能完成登录，请稍后重试。")
		}
		return
	}
	// Fetch the GitHub user.
	userReq, _ := http.NewRequest(http.MethodGet, h.githubUserURL, nil)
	userReq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		writeOAuthError(w, st.lang, "GitHub sign-in failed", "Could not verify your GitHub account. Please try again.", "GitHub 登录失败", "无法校验你的 GitHub 账号，请稍后重试。")
		return
	}
	defer userResp.Body.Close()
	var ghUser struct {
		ID    int    `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&ghUser); err != nil || ghUser.ID == 0 {
		writeOAuthError(w, st.lang, "GitHub sign-in failed", "GitHub returned an invalid account. Please try again.", "GitHub 登录失败", "GitHub 返回了无效账号，请稍后重试。")
		return
	}
	u, err := h.store.FindOrCreateGitHubUser(fmt.Sprintf("%d", ghUser.ID), ghUser.Login, ghUser.Email)
	if err != nil {
		writeOAuthError(w, st.lang, "Sign-in failed", "Your account could not be created. Please try again.", "登录失败", "无法创建账号，请稍后重试。")
		return
	}
	token, err := h.store.CreateToken(u.ID, 30*24*time.Hour)
	if err != nil {
		writeOAuthError(w, st.lang, "Sign-in failed", "Your session could not be created. Please try again.", "登录失败", "无法创建登录会话，请稍后重试。")
		return
	}
	h.log.Printf("github oauth user=%s gh=%s", u.Username, ghUser.Login)
	if st.device != "" {
		h.mu.Lock()
		d, ok := h.deviceLogins[st.device]
		stale := false
		if !ok || d.Ready || time.Now().After(d.ExpiresAt) {
			ok = false
		} else {
			d.Ready = true
			d.Token = token
			d.Username = u.Username
			stale = time.Since(d.LastPoll) > 8*time.Second
		}
		h.mu.Unlock()
		if !ok {
			if st.lang == "en" {
				writeHTML(w, http.StatusBadRequest, errorHTML(st.lang, "Authorization link expired", "Run riffpad login again for a fresh link."))
			} else {
				writeHTML(w, http.StatusBadRequest, errorHTML(st.lang, "授权链接已过期", "请重新运行 riffpad login 获取新链接。"))
			}
			return
		}
		h.log.Printf("device login authorized user=%s", u.Username)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, receiptHTML(st.lang, "device", stale))
		return
	}
	// Hand the token back to the opener window. The default is the production
	// app; a loopback opener (validated at login time) supports local dev.
	target := st.opener
	if target == "" {
		target = "https://app.riffpad.ai"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, receiptHTML(st.lang, "app", false)+`<script>
const data = {type: "riffpad-oauth", token: `+jsonQuote(token)+`, user: `+jsonQuote(u.Username)+`};
if (window.opener) { window.opener.postMessage(data, `+jsonQuote(target)+`); window.close(); }
</script>`)
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// allowedOpener restricts postMessage targets to the production app and
// loopback origins used by local development servers.
func allowedOpener(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	if u.Scheme == "https" && (u.Host == "app.riffpad.ai" || u.Host == "api.riffpad.ai") {
		return true
	}
	if u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
		return true
	}
	return false
}

const oauthPageCSS = `<style>
body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#0b0b0c;color:#e8e6e3;font:14px/1.6 ui-monospace,SFMono-Regular,Menlo,Consolas,"PingFang SC","Microsoft YaHei",sans-serif}
.card{background:#121214;border:1px solid rgba(255,255,255,.12);padding:28px 32px;max-width:380px;width:100%;box-sizing:border-box}
.row{display:flex;align-items:center;gap:10px;margin-bottom:8px}
.check{width:12px;height:12px;background:#7ee787;flex:none}
.warn .check{background:#d29922}
.bad .check{background:#f85149}
h1{font-size:16px;margin:0}
p{margin:0;color:#a8a8ad}
</style>`

func oauthPage(lang, extraClass, title, desc string) string {
	return `<!doctype html><html lang="` + lang + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Riffpad</title>` + oauthPageCSS + `</head><body><div class="card"><div class="row ` + extraClass + `"><span class="check"></span><h1>` + title + `</h1></div><p>` + desc + `</p></div></body></html>`
}

// receiptHTML renders the styled post-login receipt page for GitHub OAuth
// (app popup or CLI device flow).
func receiptHTML(lang, mode string, stale bool) string {
	extra, title, desc := "", "登录成功", "请回到 Riffpad 页面。"
	if lang == "en" {
		title, desc = "Signed in", "You can go back to the Riffpad page."
	}
	if mode == "device" {
		if lang == "en" {
			title, desc = "CLI login authorized", "Return to your terminal — sign-in completes automatically."
		} else {
			title, desc = "CLI 登录授权成功", "可以回到终端了，登录会自动完成。"
		}
		if stale {
			extra = "warn"
			if lang == "en" {
				title, desc = "Authorized, but no waiting terminal", "The terminal may have exited. If sign-in did not complete, run riffpad login again."
			} else {
				title, desc = "授权完成，但未检测到等待中的终端", "终端可能已退出；如果未登录成功，请重新运行 riffpad login。"
			}
		}
	}
	return oauthPage(lang, extra, title, desc)
}

func errorHTML(lang, title, desc string) string {
	return oauthPage(lang, "bad", title, desc)
}

func oauthLang(r *http.Request) string {
	lang := r.URL.Query().Get("lang")
	if lang != "zh" && lang != "en" {
		lang = "zh"
	}
	return lang
}

// oauthLangFromState recovers the language embedded in the OAuth state
// ("<state>.<lang>") for callbacks where the state lookup failed and the
// original query parameter is gone.
func oauthLangFromState(state string) string {
	i := strings.LastIndex(state, ".")
	if i <= 0 || i+1 >= len(state) {
		return ""
	}
	lang := state[i+1:]
	if lang != "zh" && lang != "en" {
		return ""
	}
	return lang
}

// writeOAuthError renders a styled error page for browser-facing OAuth
// failures instead of leaking raw JSON into the user's browser.
func writeOAuthError(w http.ResponseWriter, lang, enTitle, enDesc, zhTitle, zhDesc string) {
	if lang == "en" {
		writeHTML(w, http.StatusBadGateway, errorHTML(lang, enTitle, enDesc))
		return
	}
	writeHTML(w, http.StatusBadGateway, errorHTML(lang, zhTitle, zhDesc))
}

// Package hub implements the relay: user accounts, host/device registration,
// session routing and encrypted-envelope forwarding between hosts and viewers.
package hub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
	"github.com/riffpad/riffpad/packages/webui"
)

const version = "0.2.0"

// WebSocket heartbeat parameters, shared by host and viewer connections.
// Pings go out every wsPingPeriod; if no pong (or any other frame) arrives
// within wsPongWait the connection is treated as half-open and torn down.
// Atomics (not constants) so tests can shrink them without data races.
var (
	wsWriteWait  atomic.Int64
	wsPingPeriod atomic.Int64
	wsPongWait   atomic.Int64
)

func init() {
	wsWriteWait.Store(int64(10 * time.Second))
	wsPingPeriod.Store(int64(30 * time.Second))
	wsPongWait.Store(int64(75 * time.Second))
}

func wsWriteDeadline() time.Time    { return time.Now().Add(time.Duration(wsWriteWait.Load())) }
func wsPingInterval() time.Duration { return time.Duration(wsPingPeriod.Load()) }
func wsPongTimeout() time.Duration  { return time.Duration(wsPongWait.Load()) }

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type Pairing struct {
	Code      string
	HostID    string
	Curve     string
	PublicKey string
	Expires   time.Time
}

// oauthState tracks a pending GitHub authorization. device is set when the
// flow was started from the CLI device-login page.
type oauthState struct {
	expires time.Time
	device  string
	opener  string
	lang    string
}

// deviceLogin is a pending CLI login. The user opens verificationURL, signs
// in with GitHub, and the CLI polls until Ready.
type deviceLogin struct {
	UserCode  string
	ExpiresAt time.Time
	LastPoll  time.Time
	Ready     bool
	Token     string
	Username  string
}

type hostConn struct {
	id   string
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
	once sync.Once
	// writeMu serializes direct writes with writeLoop so a final control
	// frame (superseded) can be delivered synchronously before the
	// connection is closed.
	writeMu sync.Mutex
}

type viewerConn struct {
	id        string
	deviceID  string
	sessionID string
	host      *hostConn
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	once      sync.Once
}

func (h *hostConn) closeDone() {
	h.once.Do(func() { close(h.done) })
}

// writeFrame writes one frame directly on the connection, serialized with
// writeLoop. Used for the final superseded notice, which must go out before
// the connection is closed (queueing it on send would race with done).
func (h *hostConn) writeFrame(fr hostFrame) {
	data, err := json.Marshal(fr)
	if err != nil {
		return
	}
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	_ = h.conn.SetWriteDeadline(wsWriteDeadline())
	_ = h.conn.WriteMessage(websocket.TextMessage, data)
}

func (v *viewerConn) closeDone() {
	v.once.Do(func() { close(v.done) })
}

type Hub struct {
	log     *log.Logger
	dataDir string
	store   *Store

	mu             sync.Mutex
	hosts          map[string]*hostConn
	sessions       map[string]SessionMeta
	sessionHosts   map[string]string
	viewers        map[string]*viewerConn
	pairings       map[string]Pairing
	rateLimits     map[string]ipCounter
	oauthStates    map[string]oauthState
	deviceLogins   map[string]*deviceLogin
	githubID       string
	githubSecret   string
	githubTokenURL string
	githubUserURL  string
	appURL         string
	apiHosts       []string
}

type ipCounter struct {
	count       int
	windowStart time.Time
}

func New(logger *log.Logger, dataDir, databaseURL string) (*Hub, error) {
	store, err := OpenStore(dataDir, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Hub{
		log:            logger,
		dataDir:        dataDir,
		store:          store,
		hosts:          map[string]*hostConn{},
		sessions:       map[string]SessionMeta{},
		sessionHosts:   map[string]string{},
		viewers:        map[string]*viewerConn{},
		pairings:       map[string]Pairing{},
		rateLimits:     map[string]ipCounter{},
		oauthStates:    map[string]oauthState{},
		deviceLogins:   map[string]*deviceLogin{},
		githubID:       os.Getenv("GITHUB_CLIENT_ID"),
		githubSecret:   os.Getenv("GITHUB_CLIENT_SECRET"),
		githubTokenURL: "https://github.com/login/oauth/access_token",
		githubUserURL:  "https://api.github.com/user",
		appURL:         envOr("RIFFPAD_APP_URL", "https://app.riffpad.ai"),
		apiHosts:       splitCSV(os.Getenv("RIFFPAD_API_HOSTS")),
	}, nil
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// apiOnly reports whether this vhost is an API-only host (e.g. api.riffpad.ai):
// API and WebSocket routes keep working, but the web UI is not served here.
func (h *Hub) apiOnly(r *http.Request) bool {
	host := strings.ToLower(strings.Split(r.Host, ":")[0])
	for _, hh := range h.apiHosts {
		if strings.ToLower(hh) == host {
			return true
		}
	}
	return false
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/assets/", h.handleAsset)
	mux.HandleFunc("/api/status", h.handleStatus)
	mux.HandleFunc("/api/auth/register", h.handleRegister)
	mux.HandleFunc("/api/auth/login", h.handleLogin)
	mux.HandleFunc("/api/auth/logout", h.handleLogout)
	mux.HandleFunc("/api/auth/me", h.handleMe)
	mux.HandleFunc("/api/auth/github/login", h.handleGitHubLogin)
	mux.HandleFunc("/api/auth/github/callback", h.handleGitHubCallback)
	mux.HandleFunc("/api/auth/oauth/device", h.handleDeviceLogin)
	mux.HandleFunc("/api/auth/oauth/device/poll", h.handleDeviceLoginPoll)
	mux.HandleFunc("/api/auth/oauth/device/status", h.handleDeviceLoginStatus)
	mux.HandleFunc("/api/hosts", h.handleHosts)
	mux.HandleFunc("/api/hosts/register", h.handleRegisterHost)
	mux.HandleFunc("/api/pairings", h.handleCreatePairing)
	mux.HandleFunc("/api/pair", h.handlePair)
	mux.HandleFunc("/api/devices", h.handleDevices)
	mux.HandleFunc("/api/devices/", h.handleDeviceDelete)
	mux.HandleFunc("/api/hosts/", h.handleHostKillswitch)
	mux.HandleFunc("/api/sessions", h.handleSessions)
	mux.HandleFunc("/ws/host", h.handleHostWS)
	mux.HandleFunc("/ws", h.handleViewerWS)
	return mux
}

func (h *Hub) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/device" {
		http.NotFound(w, r)
		return
	}
	if h.apiOnly(r) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	raw, err := webui.IndexHTML()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "webui not built"})
		return
	}
	html := strings.Replace(string(raw), "</head>", "<script>window.RIFFPAD_RELAY=1;</script></head>", 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

func (h *Hub) handleAsset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	data, err := webui.Asset(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", webui.ContentType(name))
	_, _ = w.Write(data)
}

func (h *Hub) handleStatus(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	hosts, sessions := len(h.hosts), len(h.sessions)
	h.mu.Unlock()
	var users, devices int64
	h.store.db.Model(&User{}).Count(&users)
	h.store.db.Model(&Device{}).Count(&devices)
	writeJSON(w, http.StatusOK, map[string]any{
		"name": "riffpad-relay", "version": version,
		"hosts": hosts, "sessions": sessions, "users": users, "devices": devices,
	})
}

// ---------- auth ----------

func (h *Hub) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !h.allowRate("register", clientIP(r), 10, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Username) < 3 || len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "username >= 3 chars, password >= 6 chars")
		return
	}
	u, err := h.store.CreateUser(strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "username already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "register failed")
		return
	}
	token, err := h.store.CreateToken(u.ID, 30*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token failed")
		return
	}
	h.log.Printf("user registered username=%s", u.Username)
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

func (h *Hub) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !h.allowRate("login", clientIP(r), 10, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	u, err := h.store.VerifyLogin(strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := h.store.CreateToken(u.ID, 30*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

func (h *Hub) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	token := bearerToken(r)
	if token != "" {
		_ = h.store.DeleteToken(token)
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (h *Hub) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

// handleDeviceLogin starts a CLI device-login session. The CLI polls the
// /poll endpoint until the user authorizes it in the web app via GitHub.
func (h *Hub) handleDeviceLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !h.allowRate("oauth-device", clientIP(r), 10, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	code := newDeviceCode()
	h.mu.Lock()
	h.deviceLogins[code] = &deviceLogin{
		UserCode:  code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	h.mu.Unlock()
	verificationURL := strings.TrimSuffix(h.appURL, "/") + "/device?code=" + url.QueryEscape(code)
	writeJSON(w, http.StatusOK, map[string]any{
		"userCode":        code,
		"verificationURL": verificationURL,
		"expiresIn":       600,
		"interval":        3,
	})
}

// handleDeviceLoginPoll is the CLI polling endpoint. It returns
// {"pending":true} until the user authorizes, then hands the token once.
func (h *Hub) handleDeviceLoginPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !h.allowRate("oauth-device-poll", clientIP(r), 30, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	h.mu.Lock()
	d, ok := h.deviceLogins[code]
	if ok {
		d.LastPoll = time.Now()
	}
	if ok && time.Now().After(d.ExpiresAt) {
		delete(h.deviceLogins, code)
		ok = false
	}
	if ok && d.Ready {
		delete(h.deviceLogins, code)
	}
	h.mu.Unlock()
	if !ok {
		// Uniform error: never reveal whether a code ever existed.
		writeError(w, http.StatusUnauthorized, "invalid or expired code")
		return
	}
	if !d.Ready {
		writeJSON(w, http.StatusOK, map[string]any{"pending": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pending":  false,
		"token":    d.Token,
		"username": d.Username,
	})
}

// handleDeviceLoginStatus lets the /device page verify a code up front so a
// stale or already-used link fails immediately instead of after the user
// clicks "Continue with GitHub".
func (h *Hub) handleDeviceLoginStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if !h.allowRate("oauth-device-status", clientIP(r), 30, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	code := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("code")))
	h.mu.Lock()
	d, ok := h.deviceLogins[code]
	if ok && time.Now().After(d.ExpiresAt) {
		delete(h.deviceLogins, code)
		ok = false
	}
	valid := ok && !d.Ready
	expiresIn := 0
	if valid {
		expiresIn = int(time.Until(d.ExpiresAt).Seconds())
	}
	h.mu.Unlock()
	if !valid {
		writeJSON(w, http.StatusOK, map[string]any{"valid": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"valid": true, "expiresIn": expiresIn})
}

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

// ---------- hosts / pairing / sessions ----------

func (h *Hub) handleRegisterHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	rec, err := h.store.CreateHost(u.ID, req.Name, newSecret())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "register host failed")
		return
	}
	h.log.Printf("host registered id=%s name=%s owner=%s", rec.ID, rec.Name, u.Username)
	writeJSON(w, http.StatusOK, map[string]any{"hostId": rec.ID, "hostSecret": rec.Secret})
}

// handleHosts lists the hosts owned by the authenticated user. The daemon CLI
// uses it after login to detect stale host credentials from a previous
// account.
func (h *Hub) handleHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	hosts, err := h.store.HostsForUser(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hosts": hosts})
}

func (h *Hub) handleCreatePairing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.allowRate("pairings", clientIP(r), 30, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	var req struct {
		HostID    string `json:"hostId"`
		Curve     string `json:"curve"`
		PublicKey string `json:"publicKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.HostID == "" {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	host, err := h.store.GetHost(req.HostID)
	if err != nil || host.OwnerID != u.ID {
		writeError(w, http.StatusNotFound, "host not found")
		return
	}
	h.mu.Lock()
	_, online := h.hosts[req.HostID]
	h.mu.Unlock()
	if !online {
		writeError(w, http.StatusNotFound, "host offline")
		return
	}
	code := newCode()
	h.mu.Lock()
	h.pairings[code] = Pairing{
		Code: code, HostID: req.HostID, Curve: req.Curve,
		PublicKey: req.PublicKey, Expires: time.Now().Add(10 * time.Minute),
	}
	h.mu.Unlock()
	// The pairing link must land on the web UI origin (appURL), not on the
	// request's own host: behind a TLS-terminating reverse proxy r.TLS is nil
	// and r.Host is the API domain (e.g. api.riffpad.ai), which serves no web
	// UI. Fall back to the request host only if appURL is unconfigured.
	base := strings.TrimSuffix(h.appURL, "/")
	if base == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}
	url := base + "/?pair=" + code
	writeJSON(w, http.StatusOK, map[string]any{
		"code": code, "url": url,
		"expiresAt": time.Now().Add(10 * time.Minute).Format(time.RFC3339),
	})
}

func (h *Hub) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.allowRate("pair", clientIP(r), 10, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	var req struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		Curve     string `json:"curve"`
		PublicKey string `json:"publicKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	h.mu.Lock()
	p, ok := h.pairings[code]
	h.mu.Unlock()
	if !ok || time.Now().After(p.Expires) {
		writeError(w, http.StatusUnauthorized, "invalid or expired pairing code")
		return
	}
	host, err := h.store.GetHost(p.HostID)
	if err != nil || host.OwnerID != u.ID {
		// Keep the response identical to a missing/expired code so that
		// callers cannot tell whether a live code exists for another user.
		h.log.Printf("pair rejected: code exists but host %s is not owned by user %s", p.HostID, u.Username)
		writeError(w, http.StatusUnauthorized, "invalid or expired pairing code")
		return
	}
	name := req.Name
	if name == "" {
		name = "device"
	}
	dev, err := h.store.CreateDevice(u.ID, p.HostID, name, req.Curve, req.PublicKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create device failed")
		return
	}
	h.mu.Lock()
	delete(h.pairings, code)
	h.mu.Unlock()
	h.log.Printf("paired device=%s host=%s user=%s", dev.ID, p.HostID, u.Username)
	writeJSON(w, http.StatusOK, map[string]any{
		"deviceId":        dev.ID,
		"serverPublicKey": p.PublicKey,
		"hostId":          p.HostID,
	})
}

func (h *Hub) handleSessions(w http.ResponseWriter, r *http.Request) {
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	hostIDs, err := h.store.HostIDsForUser(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	h.mu.Lock()
	live := map[string]bool{}
	for id := range h.hosts {
		live[id] = true
	}
	// Serve the live snapshot announced by each host, not the accumulated
	// database history: sessions that exited (or hosts that restarted) must
	// not appear in the client list. hostIDs keeps owner isolation intact.
	owners := map[string]bool{}
	for _, id := range hostIDs {
		owners[id] = true
	}
	liveSessions := make([]SessionMeta, 0, len(h.sessions))
	for _, s := range h.sessions {
		if owners[s.HostID] && live[s.HostID] {
			liveSessions = append(liveSessions, s)
		}
	}
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"sessions": liveSessions})
}

// handleDevices lists devices paired to the authenticated user's hosts.
func (h *Hub) handleDevices(w http.ResponseWriter, r *http.Request) {
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	list, err := h.store.DevicesForUser(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": list})
}

// handleDeviceDelete revokes one paired device and disconnects it immediately.
func (h *Hub) handleDeviceDelete(w http.ResponseWriter, r *http.Request) {
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if err := h.store.DeleteDevice(id, u.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	n := h.disconnectViewers(func(v *viewerConn) bool { return v.deviceID == id })
	h.log.Printf("revoked device=%s user=%s disconnected=%d", id, u.Username, n)
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true, "disconnected": n})
}

// handleHostKillswitch revokes every device paired to a host and disconnects
// all of its viewers (one-action kill switch).
func (h *Hub) handleHostKillswitch(w http.ResponseWriter, r *http.Request) {
	u, ok := h.authUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	hostID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/hosts/"), "/killswitch")
	host, err := h.store.GetHost(hostID)
	if err != nil || host.OwnerID != u.ID {
		writeError(w, http.StatusNotFound, "host not found")
		return
	}
	if err := h.store.DeleteDevicesForHost(hostID); err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	n := h.disconnectViewers(func(v *viewerConn) bool { return v.host.id == hostID })
	h.log.Printf("killswitch host=%s user=%s devices-revoked disconnected=%d", hostID, u.Username, n)
	writeJSON(w, http.StatusOK, map[string]any{"killed": true, "disconnected": n})
}

// disconnectViewers closes matching viewer connections and removes them.
func (h *Hub) disconnectViewers(match func(*viewerConn) bool) int {
	h.mu.Lock()
	var targets []*viewerConn
	for _, v := range h.viewers {
		if match(v) {
			targets = append(targets, v)
		}
	}
	for _, v := range targets {
		delete(h.viewers, v.id)
		h.hostSend(v.host, hostFrame{Kind: "leave", ViewerID: v.id})
		v.closeDone()
		_ = v.conn.Close()
	}
	h.mu.Unlock()
	return len(targets)
}

// ---------- host connection ----------

type hostFrame struct {
	Kind string `json:"kind"`

	Sessions []SessionMeta `json:"sessions,omitempty"`

	ViewerID  string `json:"viewerId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	DeviceID  string `json:"deviceId,omitempty"`
	Curve     string `json:"curve,omitempty"`
	Pub       string `json:"pub,omitempty"`
	Eph       string `json:"eph,omitempty"`
	Data      string `json:"data,omitempty"`
}

func (h *Hub) handleHostWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	hostID := q.Get("hostId")
	token := q.Get("token")
	rec, err := h.store.GetHost(hostID)
	if err != nil || token != rec.Secret {
		writeError(w, http.StatusUnauthorized, "bad host credentials")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	host := &hostConn{id: hostID, conn: conn, send: make(chan []byte, 256), done: make(chan struct{})}
	h.mu.Lock()
	if old, ok := h.hosts[hostID]; ok {
		// Tell the replaced connection why it is being dropped: two daemons
		// sharing the same host credentials would otherwise kick each other
		// in an endless reconnect loop (#169).
		old.writeFrame(hostFrame{Kind: protocol.RelayFrameSuperseded})
		old.closeDone()
		_ = old.conn.Close()
	}
	h.hosts[hostID] = host
	h.mu.Unlock()
	h.log.Printf("host connected id=%s", hostID)
	go host.writeLoop()
	go h.hostReadLoop(host)
}

func (h *Hub) hostReadLoop(host *hostConn) {
	defer func() {
		h.removeHost(host)
		host.closeDone()
		_ = host.conn.Close()
	}()
	host.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	host.conn.SetPongHandler(func(string) error {
		return host.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	})
	for {
		_, data, err := host.conn.ReadMessage()
		if err != nil {
			return
		}
		var fr hostFrame
		if json.Unmarshal(data, &fr) != nil {
			continue
		}
		switch fr.Kind {
		case "sessions":
			h.mu.Lock()
			if h.hosts[host.id] == host {
				// Replace only this host's entries: other hosts of the same
				// account (desktop + laptop) keep their sessions (#169). The
				// current-connection check also rejects a stale announce from
				// a superseded connection still draining its read loop.
				for sid, hid := range h.sessionHosts {
					if hid == host.id {
						delete(h.sessionHosts, sid)
						delete(h.sessions, sid)
					}
				}
				for _, s := range fr.Sessions {
					s.HostID = host.id
					h.sessions[s.ID] = s
					h.sessionHosts[s.ID] = host.id
				}
			}
			h.mu.Unlock()
			_ = h.store.UpsertSessions(host.id, fr.Sessions)
		case "viewer":
			h.mu.Lock()
			v, ok := h.viewers[fr.ViewerID]
			h.mu.Unlock()
			if !ok {
				continue
			}
			payload, err := base64.RawStdEncoding.DecodeString(fr.Data)
			if err != nil {
				continue
			}
			h.sendToViewer(v, payload)
		case protocol.RelayFrameKick:
			// The daemon dropped this viewer (e.g. send-buffer overflow on a
			// critical event, #173): close the browser connection so the
			// client reconnects and replays instead of hanging silently.
			h.mu.Lock()
			v, ok := h.viewers[fr.ViewerID]
			h.mu.Unlock()
			if ok {
				h.dropViewer(v, "kicked by host")
			}
		}
	}
}

func (h *Hub) removeHost(host *hostConn) {
	h.mu.Lock()
	// Only the connection currently registered under this host id may tear
	// down its sessions: after a reconnect the old connection's deferred
	// cleanup runs after the new one has already re-announced, and must not
	// wipe those fresh entries (#169).
	current := h.hosts[host.id] == host
	if current {
		delete(h.hosts, host.id)
		for sid, hid := range h.sessionHosts {
			if hid == host.id {
				delete(h.sessionHosts, sid)
				delete(h.sessions, sid)
			}
		}
	}
	for id, v := range h.viewers {
		if v.host == host {
			delete(h.viewers, id)
			v.closeDone()
			_ = v.conn.Close()
		}
	}
	h.mu.Unlock()
	if current {
		_ = h.store.MarkHostSessionsOffline(host.id)
	}
	h.log.Printf("host disconnected id=%s", host.id)
}

// ---------- viewer connection ----------

func (h *Hub) handleViewerWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sid := q.Get("session")
	deviceID := q.Get("device")
	eph := q.Get("eph")
	token := q.Get("token")
	if sid == "" || deviceID == "" || eph == "" || token == "" {
		writeError(w, http.StatusBadRequest, "session, device, eph and token are required")
		return
	}
	u, err := h.store.UserByToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	dev, err := h.store.GetDevice(deviceID)
	if err != nil || dev.OwnerID != u.ID {
		writeError(w, http.StatusUnauthorized, "device not paired for this user")
		return
	}
	host := func() *hostConn {
		h.mu.Lock()
		defer h.mu.Unlock()
		hostID, ok := h.sessionHosts[sid]
		if !ok {
			return nil
		}
		hostRec, err := h.store.GetHost(hostID)
		if err != nil || hostRec.OwnerID != u.ID {
			return nil
		}
		return h.hosts[hostID]
	}()
	if host == nil {
		writeError(w, http.StatusNotFound, "session offline")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	v := &viewerConn{
		id: protocol.NewID(), deviceID: deviceID, sessionID: sid, host: host,
		conn: conn, send: make(chan []byte, 256), done: make(chan struct{}),
	}
	h.mu.Lock()
	h.viewers[v.id] = v
	h.mu.Unlock()
	h.hostSend(host, hostFrame{
		Kind: "join", ViewerID: v.id, SessionID: sid, DeviceID: deviceID,
		Curve: dev.Curve, Pub: dev.PublicKey, Eph: eph,
	})
	go v.writeLoop()
	go h.viewerReadLoop(v)
}

func (h *Hub) viewerReadLoop(v *viewerConn) {
	defer func() {
		h.mu.Lock()
		if h.viewers[v.id] == v {
			delete(h.viewers, v.id)
		}
		h.mu.Unlock()
		h.hostSend(v.host, hostFrame{Kind: "leave", ViewerID: v.id})
		v.closeDone()
		_ = v.conn.Close()
	}()
	v.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	v.conn.SetPongHandler(func(string) error {
		return v.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	})
	for {
		_, data, err := v.conn.ReadMessage()
		if err != nil {
			return
		}
		h.hostSend(v.host, hostFrame{Kind: "viewer", ViewerID: v.id, Data: base64.RawStdEncoding.EncodeToString(data)})
	}
}

// sendToViewer queues one payload for a viewer. The relay cannot inspect the
// encrypted contents, so an overflow may be dropping a critical event: close
// the connection instead of silently dropping, forcing the client to
// reconnect and replay history (#173).
func (h *Hub) sendToViewer(v *viewerConn, payload []byte) {
	select {
	case v.send <- payload:
	default:
		h.dropViewer(v, "send buffer full")
	}
}

// dropViewer removes a viewer and closes its connection. The viewer's read
// loop notices the close and sends the host a "leave" notice.
func (h *Hub) dropViewer(v *viewerConn, reason string) {
	h.mu.Lock()
	if h.viewers[v.id] != v {
		h.mu.Unlock()
		return
	}
	delete(h.viewers, v.id)
	h.mu.Unlock()
	h.log.Printf("viewer dropped (%s) viewer=%s session=%s host=%s", reason, v.id, v.sessionID, v.host.id)
	v.closeDone()
	if v.conn != nil {
		_ = v.conn.Close()
	}
}

func (h *Hub) hostSend(host *hostConn, fr hostFrame) {
	data, err := json.Marshal(fr)
	if err != nil {
		return
	}
	select {
	case host.send <- data:
	default:
		// Same reasoning as sendToViewer: with no visibility into the payload,
		// any drop may be critical. Closing forces the daemon to reconnect and
		// re-announce, and its viewers to reconnect and replay (#173).
		h.log.Printf("host send buffer full, closing connection host=%s kind=%s", host.id, fr.Kind)
		host.closeDone()
		if host.conn != nil {
			_ = host.conn.Close()
		}
	}
}

func (h *hostConn) writeLoop() {
	ticker := time.NewTicker(wsPingInterval())
	defer ticker.Stop()
	for {
		select {
		case data := <-h.send:
			_ = h.conn.SetWriteDeadline(wsWriteDeadline())
			h.writeMu.Lock()
			err := h.conn.WriteMessage(websocket.TextMessage, data)
			h.writeMu.Unlock()
			if err != nil {
				return
			}
		case <-ticker.C:
			if err := h.conn.WriteControl(websocket.PingMessage, nil, wsWriteDeadline()); err != nil {
				return
			}
		case <-h.done:
			return
		}
	}
}

func (v *viewerConn) writeLoop() {
	ticker := time.NewTicker(wsPingInterval())
	defer ticker.Stop()
	for {
		select {
		case data := <-v.send:
			_ = v.conn.SetWriteDeadline(wsWriteDeadline())
			if err := v.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			if err := v.conn.WriteControl(websocket.PingMessage, nil, wsWriteDeadline()); err != nil {
				return
			}
		case <-v.done:
			return
		}
	}
}

// ---------- helpers ----------

func (h *Hub) authUser(r *http.Request) (*User, bool) {
	token := bearerToken(r)
	if token == "" {
		return nil, false
	}
	u, err := h.store.UserByToken(token)
	if err != nil {
		return nil, false
	}
	return u, true
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (h *Hub) allowRate(scope, ip string, limit int, window time.Duration) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := scope + "|" + ip
	c, ok := h.rateLimits[key]
	now := time.Now()
	if !ok || now.Sub(c.windowStart) > window {
		c = ipCounter{windowStart: now}
	}
	c.count++
	if c.count > limit {
		return false
	}
	h.rateLimits[key] = c
	return true
}

func clientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if i := strings.Index(ip, ","); i >= 0 {
		ip = ip[:i]
	}
	if ip == "" {
		ip = strings.SplitN(r.RemoteAddr, ":", 2)[0]
	}
	return strings.TrimSpace(ip)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeHTML(w http.ResponseWriter, status int, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, html)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

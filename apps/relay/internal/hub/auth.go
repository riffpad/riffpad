package hub

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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

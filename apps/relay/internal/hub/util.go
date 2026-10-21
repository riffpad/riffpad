package hub

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
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

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// Rate limits are keyed by whatever the caller can actually be held to: a
// username, an email, a device code, a user ID, or the shared "global" bucket.
// Never by client IP: the relay runs behind a reverse proxy, so the client
// controls any IP-shaped header (X-Forwarded-For was trusted here until #349,
// which made every limit bypassable), and RemoteAddr is the proxy itself.
// Per-source floods are the proxy's job (nginx limit_req); these buckets bound
// how much a single identity — or the whole instance — can burn.
const rateKeyGlobal = "global"

// Per-minute limits. The global buckets are deliberately high: the point is to
// cap a runaway instance, not to let one attacker lock everybody else out.
var (
	limitRegisterGlobal = 120
	limitLoginGlobal    = 300
	limitDeviceGlobal   = 120
	limitDevicePoll     = 30 // per device code
	limitDeviceStatus   = 30 // per device code
	limitPairings       = 30 // per user
	limitPairGlobal     = 300
	limitWaitlistEmail  = 5 // per email
	limitWaitlistGlobal = 120
)

// Login backoff: repeated failures for one username slow that username down
// instead of locking it. A hard lock would let anyone take a known account
// offline just by guessing wrong passwords.
var (
	loginFreeAttempts = 3
	loginBackoffBase  = time.Second
	loginBackoffMax   = time.Minute
	loginMaxGuards    = 10000
)

type rateCounter struct {
	count       int
	windowStart time.Time
}

// allowRate is a fixed-window counter over scope|key.
func (h *Hub) allowRate(scope, key string, limit int, window time.Duration) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	k := scope + "|" + key
	c, ok := h.rateLimits[k]
	now := time.Now()
	if !ok || now.Sub(c.windowStart) > window {
		c = rateCounter{windowStart: now}
	}
	c.count++
	if c.count > limit {
		return false
	}
	h.rateLimits[k] = c
	return true
}

// loginGuard is the per-username failure state used by loginBackoffDelay.
type loginGuard struct {
	fails int
	next  time.Time
}

// loginBackoffDelay returns how long the username must wait after fails
// consecutive failures. The first few typos are free.
func loginBackoffDelay(fails int) time.Duration {
	if fails <= loginFreeAttempts {
		return 0
	}
	shift := fails - loginFreeAttempts - 1
	if shift > 10 {
		shift = 10
	}
	d := loginBackoffBase << shift
	if d > loginBackoffMax {
		d = loginBackoffMax
	}
	return d
}

// loginReady reports whether username may try again now, and clears expired
// state on the way through.
func (h *Hub) loginReady(username string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	g, ok := h.loginGuards[username]
	if !ok {
		return true
	}
	if time.Now().Before(g.next) {
		return false
	}
	if g.fails == 0 {
		delete(h.loginGuards, username)
	}
	return true
}

func (h *Hub) loginFailed(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	g := h.loginGuards[username]
	if g == nil {
		g = &loginGuard{}
	}
	g.fails++
	g.next = time.Now().Add(loginBackoffDelay(g.fails))
	h.loginGuards[username] = g
	if len(h.loginGuards) > loginMaxGuards {
		h.pruneLoginGuardsLocked()
	}
}

func (h *Hub) loginSucceeded(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.loginGuards, username)
}

// pruneLoginGuardsLocked drops guards whose backoff already elapsed, so the
// map cannot grow without bound from throwaway usernames.
func (h *Hub) pruneLoginGuardsLocked() {
	now := time.Now()
	for name, g := range h.loginGuards {
		if !now.Before(g.next) {
			delete(h.loginGuards, name)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeErrorCode is writeError plus a machine-readable code field so clients
// can localize the message. Older clients only read "error", so adding the
// field is backward compatible.
func writeErrorCode(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": msg, "code": code})
}

func writeHTML(w http.ResponseWriter, status int, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, html)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

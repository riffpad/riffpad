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
// username, an email, a device code, a user ID, or a shared bucket. Never by
// client IP: the relay runs behind a reverse proxy, so the client controls any
// IP-shaped header (X-Forwarded-For was trusted here until #349, which made
// every limit bypassable), and RemoteAddr is the proxy itself. Per-source
// floods are the proxy's job (nginx limit_req); these buckets bound how much a
// single identity — or the whole instance — can burn.
const rateKeyGlobal = "global"

// Per-minute limits. Per-identity buckets are the real defence; the global
// buckets are circuit breakers that only a distributed flood can trip, so they
// are set an order of magnitude above what one account can spend.
var (
	limitRegisterGlobal = 600
	limitLoginGlobal    = 300
	limitDeviceGlobal   = 120
	limitDevicePoll     = 30   // per device code
	limitDeviceStatus   = 30   // per device code
	limitPairings       = 30   // per user
	limitPairUser       = 10   // per user: stops one account breaking pairing for everybody
	limitPairGlobal     = 3000 // distributed flood breaker only
	limitWaitlistEmail  = 5    // per email
	limitWaitlistGlobal = 120
)

// Login backoff: repeated failures for one username slow that username down
// instead of locking it. A hard lock would let anyone take a known account
// offline just by guessing wrong passwords. Only one attempt per username may
// be in flight at a time, otherwise a concurrent burst would all pass the
// backoff check before any of them recorded a failure.
var (
	loginFreeAttempts = 3
	loginBackoffBase  = time.Second
	loginBackoffMax   = time.Minute
)

// Both bucket maps are keyed by attacker-chosen values (usernames, emails,
// device codes), so they need a cap: without one a random-key flood grows the
// map until the process dies. Sweeps are amortised (at most once per interval
// on the insert path) instead of a full scan per insert.
var (
	rateLimitsMax            = 20000
	rateLimitsSweepInterval  = time.Minute
	loginGuardsMax           = 10000
	loginGuardsSweepInterval = time.Minute
)

// sweepFraction is how far below the cap a sweep goes when the map is full of
// live entries: evicting a batch keeps the amortised cost low.
const sweepFraction = 4

type rateCounter struct {
	count       int
	windowStart time.Time
	expires     time.Time
}

// allowRate is a fixed-window counter over scope|key.
func (h *Hub) allowRate(scope, key string, limit int, window time.Duration) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	if len(h.rateLimits) >= rateLimitsMax || now.Sub(h.lastRateSweep) > rateLimitsSweepInterval {
		h.sweepRateLimitsLocked(now)
		h.lastRateSweep = now
	}
	k := scope + "|" + key
	c, ok := h.rateLimits[k]
	if !ok || now.Sub(c.windowStart) > window {
		c = rateCounter{windowStart: now, expires: now.Add(window)}
	}
	c.count++
	if c.count > limit {
		return false
	}
	h.rateLimits[k] = c
	return true
}

// sweepRateLimitsLocked drops expired buckets and, if that is not enough,
// evicts the ones expiring soonest until the map is comfortably under its cap.
// Caller holds h.mu.
func (h *Hub) sweepRateLimitsLocked(now time.Time) {
	for k, c := range h.rateLimits {
		if !now.Before(c.expires) {
			delete(h.rateLimits, k)
		}
	}
	if len(h.rateLimits) < rateLimitsMax {
		return
	}
	// Still full of live buckets (an active flood): evict a batch arbitrarily.
	// During a flood that costs a few seconds of limiting, which is a far
	// better trade than unbounded memory.
	target := rateLimitsMax - rateLimitsMax/sweepFraction
	for k := range h.rateLimits {
		if len(h.rateLimits) <= target {
			return
		}
		delete(h.rateLimits, k)
	}
}

// loginOutcome tells endLoginAttempt how the attempt finished.
type loginOutcome int

const (
	loginAttemptRetry  loginOutcome = iota // our own failure (DB error): release the slot, no penalty
	loginAttemptOK                         // credentials accepted: forget the failures
	loginAttemptDenied                     // wrong password / unknown user: back off
)

// loginGuard is the per-username failure state used by loginBackoffDelay.
type loginGuard struct {
	fails    int
	inflight int
	next     time.Time
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

// beginLoginAttempt reserves the single in-flight slot for username. Charging
// the slot here (rather than counting failures afterwards) is what stops a
// concurrent burst: the second request is refused while the first is still
// checking the password.
func (h *Hub) beginLoginAttempt(username string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	g := h.loginGuards[username]
	if g == nil {
		g = h.newLoginGuardLocked(username, now)
	}
	if now.Before(g.next) || g.inflight > 0 {
		return false
	}
	g.inflight++
	return true
}

// endLoginAttempt releases the slot reserved by beginLoginAttempt and applies
// the outcome. Must be called exactly once per successful begin.
func (h *Hub) endLoginAttempt(username string, outcome loginOutcome) {
	h.mu.Lock()
	defer h.mu.Unlock()
	g := h.loginGuards[username]
	if g == nil {
		return
	}
	if g.inflight > 0 {
		g.inflight--
	}
	switch outcome {
	case loginAttemptOK:
		delete(h.loginGuards, username)
	case loginAttemptDenied:
		g.fails++
		g.next = time.Now().Add(loginBackoffDelay(g.fails))
	}
}

// newLoginGuardLocked allocates a guard, sweeping the map first when it is
// full or the sweep interval has passed. Caller holds h.mu.
func (h *Hub) newLoginGuardLocked(username string, now time.Time) *loginGuard {
	if len(h.loginGuards) >= loginGuardsMax || now.Sub(h.lastLoginSweep) > loginGuardsSweepInterval {
		h.sweepLoginGuardsLocked(now)
		h.lastLoginSweep = now
	}
	g := &loginGuard{}
	h.loginGuards[username] = g
	return g
}

// sweepLoginGuardsLocked forgets usernames whose backoff has elapsed and that
// have no attempt in flight, then evicts arbitrary idle entries if the map is
// still full. Caller holds h.mu.
func (h *Hub) sweepLoginGuardsLocked(now time.Time) {
	for name, g := range h.loginGuards {
		if g.inflight == 0 && !now.Before(g.next) {
			delete(h.loginGuards, name)
		}
	}
	target := loginGuardsMax - loginGuardsMax/sweepFraction
	for name, g := range h.loginGuards {
		if len(h.loginGuards) <= target {
			return
		}
		if g.inflight == 0 {
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

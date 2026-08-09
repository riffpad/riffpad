package hub

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"
)

// Waitlist unsubscribe endpoints. The announcement tool signs every email
// with UNSUBSCRIBE_SECRET (HMAC-SHA256 of the normalized address); the
// landing page presents that link to the relay, which verifies the signature
// and stores the opt-out in email_optouts. The tool then fetches the list
// with WAITLIST_ADMIN_KEY so future sends skip opted-out addresses.

func normalizeWaitlistEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func validWaitlistEmail(s string) bool {
	if s == "" || strings.ContainsAny(s, "\r\n ") {
		return false
	}
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}

func waitlistSign(email string) (string, error) {
	if h, ok := os.LookupEnv("UNSUBSCRIBE_SECRET"); ok {
		mac := hmac.New(sha256.New, []byte(h))
		_, _ = mac.Write([]byte(normalizeWaitlistEmail(email)))
		return hex.EncodeToString(mac.Sum(nil)), nil
	}
	return "", os.ErrNotExist
}

func waitlistValidSignature(email, token string) bool {
	want, err := waitlistSign(email)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(want)) == 1
}

// setCORSHeaders allows browser calls from the configured web origins only.
func (h *Hub) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	for _, allowed := range h.webOrigins {
		if strings.EqualFold(allowed, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			break
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Vary", "Origin")
}

// handleWaitlistUnsubscribe records an email opt-out after verifying the
// HMAC token embedded in the announcement link. Idempotent.
func (h *Hub) handleWaitlistUnsubscribe(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if h.unsubSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "waitlist unsubscribe not configured")
		return
	}
	if !h.allowRate("waitlist-unsub", clientIP(r), 10, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	var req struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	email := normalizeWaitlistEmail(req.Email)
	if !validWaitlistEmail(email) || req.Token == "" {
		writeError(w, http.StatusBadRequest, "invalid email or token")
		return
	}
	if !waitlistValidSignature(email, req.Token) {
		writeError(w, http.StatusBadRequest, "invalid token")
		return
	}
	if err := h.store.AddEmailOptout(email); err != nil {
		h.log.Printf("waitlist opt-out failed email=%s: %v", email, err)
		writeError(w, http.StatusInternalServerError, "failed to save opt-out")
		return
	}
	h.log.Printf("waitlist opt-out email=%s", email)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleWaitlistOptouts returns the opted-out addresses for the announcement
// tool. Protected by WAITLIST_ADMIN_KEY; not meant for browsers.
func (h *Hub) handleWaitlistOptouts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if h.adminKey == "" || subtle.ConstantTimeCompare(
		[]byte(r.Header.Get("X-Admin-Key")), []byte(h.adminKey)) != 1 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	emails, err := h.store.EmailOptouts()
	if err != nil {
		h.log.Printf("waitlist opt-outs read failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to read opt-outs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"emails": emails})
}

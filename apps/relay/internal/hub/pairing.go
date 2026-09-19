package hub

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newCodeN(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeAlphabet))))
		if err != nil {
			panic(err)
		}
		b.WriteByte(codeAlphabet[n.Int64()])
	}
	return b.String()
}

func newCode() string {
	return newCodeN(6)
}

func newDeviceCode() string {
	return newCodeN(8)
}

func newSecret() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
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
	expiresAt := time.Now().Add(10 * time.Minute)
	if err := h.store.CreatePairing(&PairingRecord{
		Code: code, HostID: req.HostID, Curve: req.Curve,
		PublicKey: req.PublicKey, ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "create pairing failed")
		return
	}
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
		"expiresAt": expiresAt.Format(time.RFC3339),
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
	p, err := h.store.GetPairing(code)
	if err != nil {
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
	// The code is consumed before the device is created so that concurrent
	// pair attempts (double click, two phones scanning one QR) cannot each
	// mint a device from the same code: ConsumePairing is a DB-level
	// compare-and-swap and only one caller wins. The ownership check above
	// runs first so a foreign user can neither burn nor probe the code.
	if p.ConsumedAt != nil {
		writeErrorCode(w, http.StatusConflict, "pairing_code_used", "pairing code already used")
		return
	}
	if err := h.store.ConsumePairing(code); err != nil {
		if errors.Is(err, ErrPairingUsed) {
			writeErrorCode(w, http.StatusConflict, "pairing_code_used", "pairing code already used")
			return
		}
		writeError(w, http.StatusInternalServerError, "pair failed")
		return
	}
	name := req.Name
	if name == "" {
		name = "device"
	}
	dev, err := h.store.CreateDevice(u.ID, p.HostID, name, req.Curve, req.PublicKey)
	if err != nil {
		// The code stays consumed: CreateDevice only fails on a database
		// error, in which case an un-consume write would very likely fail
		// too. The user regenerates a code.
		writeError(w, http.StatusInternalServerError, "create device failed")
		return
	}
	h.log.Printf("paired device=%s host=%s user=%s", dev.ID, p.HostID, u.Username)
	writeJSON(w, http.StatusOK, map[string]any{
		"deviceId":        dev.ID,
		"serverPublicKey": p.PublicKey,
		"hostId":          p.HostID,
	})
}

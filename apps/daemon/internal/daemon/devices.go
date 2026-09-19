package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/packages/protocol"
)

// Device is a paired client (phone or web UI).
type Device struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Curve     protocol.Curve `json:"curve"`
	PublicKey string         `json:"publicKey"`
	CreatedAt time.Time      `json:"createdAt"`
}

func (d Device) PublicKeyBytes() ([]byte, error) {
	return protocol.DecodeKey(d.PublicKey)
}

type pendingPair struct {
	Code    string
	Expires time.Time
}

func (s *Server) handleCreatePairing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	// ?local=1 forces a local-only code (for the embedded UI at 8787) even
	// when the daemon is connected to a relay. Without it, a relay-connected
	// daemon only mints cloud codes, which the local UI cannot claim because
	// handlePair only looks up the daemon's in-memory pending codes.
	if r.URL.Query().Get("local") != "1" && s.rc != nil {
		s.createRemotePairing(w)
		return
	}
	code := newPairingCode()
	s.mu.Lock()
	now := time.Now()
	// Expired codes are only reaped when used (#174); sweep them here so
	// repeated `riffpad pair` calls cannot grow s.pending without bound.
	for c, p := range s.pending {
		if now.After(p.Expires) {
			delete(s.pending, c)
		}
	}
	s.pending[code] = pendingPair{Code: code, Expires: now.Add(10 * time.Minute)}
	s.mu.Unlock()
	// Local mode: the daemon only listens on 127.0.0.1, so this URL is only
	// reachable from a browser on this machine. The `local` flag tells the
	// CLI to print the URL instead of a QR code (a phone scanning it would
	// try to open the phone's own localhost).
	url := fmt.Sprintf("http://127.0.0.1:%d/?pair=%s&token=%s", s.cfg.Port, code, s.token)
	writeJSON(w, http.StatusOK, map[string]any{
		"code":      code,
		"url":       url,
		"local":     true,
		"expiresAt": time.Now().Add(10 * time.Minute).Format(time.RFC3339),
	})
}

func (s *Server) createRemotePairing(w http.ResponseWriter) {
	if s.cfg.RelayToken == "" {
		writeError(w, http.StatusUnauthorized, "未登录：请先运行 riffpad login")
		return
	}
	httpURL := s.cfg.RelayURL
	httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
	httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")
	body, _ := json.Marshal(map[string]string{
		"hostId":    s.rc.hostID,
		"curve":     "p256",
		"publicKey": s.keys.P256Public,
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimSuffix(httpURL, "/")+"/api/pairings", strings.NewReader(string(body)))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "pairing request failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.RelayToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.RelayToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "relay unreachable: "+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		// Relay login tokens expire (30-day TTL) while the host WS keeps
		// working (it authenticates with the host secret), so an expired
		// login often surfaces here first. Give the user an actionable
		// message; errorCode lets the CLI localize it (#172).
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":     "登录已过期，请重新运行 riffpad login",
			"errorCode": "relay_auth_expired",
		})
		return
	}
	if resp.StatusCode != http.StatusOK {
		var relayErr struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&relayErr)
		msg := relayErr.Error
		if msg == "" {
			msg = fmt.Sprintf("relay pairing failed (status %d)", resp.StatusCode)
		}
		writeError(w, http.StatusBadGateway, msg)
		return
	}
	var out struct {
		Code      string `json:"code"`
		URL       string `json:"url"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		writeError(w, http.StatusBadGateway, "invalid relay response")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"code": out.Code, "url": out.URL, "expiresAt": out.ExpiresAt})
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		Code      string         `json:"code"`
		Name      string         `json:"name"`
		Curve     protocol.Curve `json:"curve"`
		PublicKey string         `json:"publicKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	s.mu.Lock()
	p, ok := s.pending[strings.ToUpper(strings.TrimSpace(req.Code))]
	if !ok {
		s.mu.Unlock()
		writeError(w, http.StatusUnauthorized, "invalid or expired pairing code")
		return
	}
	delete(s.pending, p.Code)
	s.mu.Unlock()
	if time.Now().After(p.Expires) {
		writeError(w, http.StatusUnauthorized, "pairing code expired")
		return
	}
	if req.Curve != protocol.CurveX25519 && req.Curve != protocol.CurveP256 {
		writeError(w, http.StatusBadRequest, "unsupported curve")
		return
	}
	pub, err := protocol.DecodeKey(req.PublicKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid public key")
		return
	}
	dev := Device{
		ID:        protocol.NewID(),
		Name:      req.Name,
		Curve:     req.Curve,
		PublicKey: protocol.EncodeKey(pub),
		CreatedAt: time.Now(),
	}
	if dev.Name == "" {
		dev.Name = "device-" + dev.ID[:6]
	}
	s.mu.Lock()
	s.devices[dev.ID] = dev
	s.mu.Unlock()
	_ = s.saveDevices()
	identity, err := s.keys.Identity(req.Curve)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server identity unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deviceId":        dev.ID,
		"serverPublicKey": protocol.EncodeKey(identity.PublicKey),
		// Local API token so a UI that paired by code (no ?token= link) can make
		// subsequent authenticated calls.
		"localToken": s.token,
	})
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	s.mu.Lock()
	list := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		list = append(list, d)
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"devices": list})
}

func (s *Server) handleDevice(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	s.mu.Lock()
	_, ok := s.devices[id]
	if ok {
		delete(s.devices, id)
	}
	s.mu.Unlock()
	_ = s.saveDevices()
	writeJSON(w, http.StatusOK, map[string]any{"revoked": ok})
}

func (s *Server) loadDevices() {
	path := filepath.Join(s.dataDir, "devices.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var list []Device
	if err := json.Unmarshal(data, &list); err != nil {
		// Corrupted devices file: back it up and start with an empty list
		// instead of silently ignoring it (#172).
		if backup, berr := config.BackupCorrupted(path, "devices"); berr != nil {
			s.log.Printf("devices.json corrupted (%v); backup failed: %v", err, berr)
		} else {
			s.log.Printf("devices.json corrupted (%v); backed up to %s, starting with no paired devices", err, backup)
		}
		return
	}
	for _, d := range list {
		s.devices[d.ID] = d
	}
}

func (s *Server) saveDevices() error {
	s.mu.Lock()
	list := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		list = append(list, d)
	}
	s.mu.Unlock()
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteFileAtomic(filepath.Join(s.dataDir, "devices.json"), data, 0o600)
}

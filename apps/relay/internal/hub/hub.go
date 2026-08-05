// Package hub implements the stateless WebSocket relay: hosts (daemons)
// announce sessions, viewers connect to a session, and encrypted envelopes
// are forwarded between the two without inspection or persistence.
package hub

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
	"github.com/riffpad/riffpad/packages/webui"
)

const version = "0.1.0-m1"

type SessionMeta struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	CLI    string `json:"cli"`
	Cwd    string `json:"cwd"`
	Status string `json:"status"`
}

type Pairing struct {
	Code      string
	HostID    string
	Curve     protocol.Curve
	PublicKey string
	Expires   time.Time
}

type Device struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Curve     protocol.Curve `json:"curve"`
	PublicKey string         `json:"publicKey"`
	HostID    string         `json:"hostId"`
	CreatedAt time.Time      `json:"createdAt"`
}

type HostRecord struct {
	ID        string    `json:"id"`
	Secret    string    `json:"secret"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type hostConn struct {
	id   string
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
}

type viewerConn struct {
	id        string
	sessionID string
	host      *hostConn
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
}

type Hub struct {
	log     *log.Logger
	regKey  string
	dataDir string

	mu           sync.Mutex
	hosts        map[string]*hostConn
	hostRecords  map[string]HostRecord
	sessions     map[string]SessionMeta
	sessionHosts map[string]string
	viewers      map[string]*viewerConn
	pairings     map[string]Pairing
	devices      map[string]Device
	rateLimits   map[string]ipCounter
}

type ipCounter struct {
	count       int
	windowStart time.Time
}

func New(logger *log.Logger, regKey, dataDir string) *Hub {
	h := &Hub{
		log:          logger,
		regKey:       regKey,
		dataDir:      dataDir,
		hosts:        map[string]*hostConn{},
		hostRecords:  map[string]HostRecord{},
		sessions:     map[string]SessionMeta{},
		sessionHosts: map[string]string{},
		viewers:      map[string]*viewerConn{},
		pairings:     map[string]Pairing{},
		devices:      map[string]Device{},
		rateLimits:   map[string]ipCounter{},
	}
	h.loadStore()
	return h
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/app.js", h.handleAsset)
	mux.HandleFunc("/style.css", h.handleAsset)
	mux.HandleFunc("/api/status", h.handleStatus)
	mux.HandleFunc("/api/hosts/register", h.handleRegisterHost)
	mux.HandleFunc("/api/pairings", h.handleCreatePairing)
	mux.HandleFunc("/api/pair", h.handlePair)
	mux.HandleFunc("/api/sessions", h.handleSessions)
	mux.HandleFunc("/ws/host", h.handleHostWS)
	mux.HandleFunc("/ws", h.handleViewerWS)
	return mux
}

func (h *Hub) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	html := strings.Replace(string(webui.IndexHTML), "</head>", "<script>window.RIFFPAD_RELAY=1;</script></head>", 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

func (h *Hub) handleAsset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	switch r.URL.Path {
	case "/app.js":
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write(webui.AppJS)
	case "/style.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(webui.StyleCSS)
	default:
		http.NotFound(w, r)
	}
}

func (h *Hub) handleStatus(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     "riffpad-relay",
		"version":  version,
		"hosts":    len(h.hosts),
		"sessions": len(h.sessions),
		"devices":  len(h.devices),
	})
}

func (h *Hub) handleRegisterHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		Name            string `json:"name"`
		RegistrationKey string `json:"registrationKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if h.regKey != "" && req.RegistrationKey != h.regKey {
		writeError(w, http.StatusUnauthorized, "invalid registration key")
		return
	}
	rec := HostRecord{
		ID:        "h-" + protocol.NewID()[:12],
		Secret:    newSecret(),
		Name:      req.Name,
		CreatedAt: time.Now(),
	}
	h.mu.Lock()
	h.hostRecords[rec.ID] = rec
	h.mu.Unlock()
	h.saveHosts()
	h.log.Printf("host registered id=%s name=%s", rec.ID, req.Name)
	writeJSON(w, http.StatusOK, map[string]any{
		"hostId":     rec.ID,
		"hostSecret": rec.Secret,
	})
}

func (h *Hub) handleCreatePairing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !h.allowRate("pairings", clientIP(r), 30, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	var req struct {
		HostID    string         `json:"hostId"`
		Curve     protocol.Curve `json:"curve"`
		PublicKey string         `json:"publicKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.HostID == "" {
		writeError(w, http.StatusBadRequest, "invalid body")
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
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	url := scheme + "://" + r.Host + "/?pair=" + code
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
	if !h.allowRate("pair", clientIP(r), 10, time.Minute) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
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
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	h.mu.Lock()
	p, ok := h.pairings[code]
	h.mu.Unlock()
	if !ok || time.Now().After(p.Expires) {
		writeError(w, http.StatusUnauthorized, "invalid or expired pairing code")
		return
	}
	dev := Device{
		ID: protocol.NewID(), Name: req.Name, Curve: req.Curve,
		PublicKey: req.PublicKey, HostID: p.HostID, CreatedAt: time.Now(),
	}
	if dev.Name == "" {
		dev.Name = "device-" + dev.ID[:6]
	}
	h.mu.Lock()
	h.devices[dev.ID] = dev
	delete(h.pairings, code)
	h.mu.Unlock()
	h.saveDevices()
	h.log.Printf("paired device=%s host=%s", dev.ID, p.HostID)
	writeJSON(w, http.StatusOK, map[string]any{
		"deviceId":        dev.ID,
		"serverPublicKey": p.PublicKey,
		"hostId":          p.HostID,
	})
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

func (h *Hub) handleSessions(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	list := make([]SessionMeta, 0, len(h.sessions))
	for _, s := range h.sessions {
		list = append(list, s)
	}
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

// hostFrame is the JSON envelope used between relay and hosts.
type hostFrame struct {
	Kind string `json:"kind"` // hello | sessions | join | leave | viewer
	// sessions
	Sessions []SessionMeta `json:"sessions,omitempty"`
	// join / leave
	ViewerID  string         `json:"viewerId,omitempty"`
	SessionID string         `json:"sessionId,omitempty"`
	DeviceID  string         `json:"deviceId,omitempty"`
	Curve     protocol.Curve `json:"curve,omitempty"`
	Pub       string         `json:"pub,omitempty"`
	Eph       string         `json:"eph,omitempty"`
	// viewer
	Data string `json:"data,omitempty"`
}

func (h *Hub) handleHostWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	hostID := q.Get("hostId")
	token := q.Get("token")
	h.mu.Lock()
	rec, ok := h.hostRecords[hostID]
	h.mu.Unlock()
	if !ok || token != rec.Secret {
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
		close(old.done)
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
		close(host.done)
		_ = host.conn.Close()
	}()
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
			h.sessions = map[string]SessionMeta{}
			h.sessionHosts = map[string]string{}
			for _, s := range fr.Sessions {
				h.sessions[s.ID] = s
				h.sessionHosts[s.ID] = host.id
			}
			h.mu.Unlock()
		case "viewer":
			h.mu.Lock()
			v, ok := h.viewers[fr.ViewerID]
			h.mu.Unlock()
			if !ok {
				continue
			}
			data, err := base64.RawStdEncoding.DecodeString(fr.Data)
			if err != nil {
				continue
			}
			select {
			case v.send <- data:
			default:
			}
		}
	}
}

func (h *Hub) removeHost(host *hostConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hosts[host.id] == host {
		delete(h.hosts, host.id)
	}
	for sid, hid := range h.sessionHosts {
		if hid == host.id {
			delete(h.sessionHosts, sid)
			delete(h.sessions, sid)
		}
	}
	for id, v := range h.viewers {
		if v.host == host {
			delete(h.viewers, id)
			close(v.done)
			_ = v.conn.Close()
		}
	}
	h.log.Printf("host disconnected id=%s", host.id)
}

func (h *Hub) handleViewerWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sid := q.Get("session")
	deviceID := q.Get("device")
	eph := q.Get("eph")
	if sid == "" || deviceID == "" || eph == "" {
		writeError(w, http.StatusBadRequest, "session, device and eph are required")
		return
	}
	h.mu.Lock()
	dev, devOK := h.devices[deviceID]
	h.mu.Unlock()
	if !devOK {
		writeError(w, http.StatusUnauthorized, "device not paired")
		return
	}
	host := func() *hostConn {
		h.mu.Lock()
		defer h.mu.Unlock()
		hostID, ok := h.sessionHosts[sid]
		if !ok {
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
		id: protocol.NewID(), sessionID: sid, host: host,
		conn: conn, send: make(chan []byte, 256), done: make(chan struct{}),
	}
	h.mu.Lock()
	h.viewers[v.id] = v
	h.mu.Unlock()
	join := hostFrame{
		Kind: "join", ViewerID: v.id, SessionID: sid, DeviceID: deviceID,
		Curve: dev.Curve, Pub: dev.PublicKey, Eph: eph,
	}
	hostSend(host, join)
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
		hostSend(v.host, hostFrame{Kind: "leave", ViewerID: v.id})
		close(v.done)
		_ = v.conn.Close()
	}()
	for {
		_, data, err := v.conn.ReadMessage()
		if err != nil {
			return
		}
		hostSend(v.host, hostFrame{Kind: "viewer", ViewerID: v.id, Data: base64.RawStdEncoding.EncodeToString(data)})
	}
}

func hostSend(host *hostConn, fr hostFrame) {
	data, err := json.Marshal(fr)
	if err != nil {
		return
	}
	select {
	case host.send <- data:
	default:
	}
}

func (h *hostConn) writeLoop() {
	for {
		select {
		case data := <-h.send:
			if err := h.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-h.done:
			return
		}
	}
}

func (v *viewerConn) writeLoop() {
	for {
		select {
		case data := <-v.send:
			if err := v.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-v.done:
			return
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

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

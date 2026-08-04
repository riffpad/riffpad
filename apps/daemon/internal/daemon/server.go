package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/claude"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/webui"
	"github.com/riffpad/riffpad/packages/protocol"
)

const version = "0.1.0-m0"

// Device is a paired client (phone or web UI).
type Device struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Curve     protocol.Curve  `json:"curve"`
	PublicKey string          `json:"publicKey"`
	CreatedAt time.Time       `json:"createdAt"`
}

func (d Device) PublicKeyBytes() ([]byte, error) {
	return protocol.DecodeKey(d.PublicKey)
}

type pendingPair struct {
	Code    string
	Expires time.Time
}

type session struct {
	id      string
	meta    protocol.SessionStartPayload
	adapter adapter.Session
	events  <-chan protocol.Event
	status  string
	mu      sync.Mutex
	history []protocol.Event
	clients map[*client]struct{}
}

// Server is the local daemon HTTP/WS server.
type Server struct {
	cfg     *config.Config
	keys    *config.Keys
	dataDir string
	log     *log.Logger
	factory adapter.Factory
	httpSrv *http.Server

	mu           sync.Mutex
	devices      map[string]Device
	pending      map[string]pendingPair
	sessions     map[string]*session
	pendingHooks map[string]chan string
}

// New creates a daemon server.
func New(cfg *config.Config, keys *config.Keys, dataDir string, logger *log.Logger, factory adapter.Factory) *Server {
	s := &Server{
		cfg:          cfg,
		keys:         keys,
		dataDir:      dataDir,
		log:          logger,
		factory:      factory,
		devices:      map[string]Device{},
		pending:      map[string]pendingPair{},
		sessions:     map[string]*session{},
		pendingHooks: map[string]chan string{},
	}
	s.loadDevices()
	return s
}

// DefaultFactory maps CLI names to adapters.
func DefaultFactory() adapter.Factory {
	return func(_ context.Context, req adapter.CreateRequest) (adapter.Session, error) {
		switch req.CLI {
		case "", "claude":
			return claude.New(req), nil
		default:
			return nil, fmt.Errorf("unsupported cli %q", req.CLI)
		}
	}
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/app.js", s.handleAsset)
	mux.HandleFunc("/style.css", s.handleAsset)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/pairings", s.handleCreatePairing)
	mux.HandleFunc("/api/pair", s.handlePair)
	mux.HandleFunc("/api/devices", s.handleDevices)
	mux.HandleFunc("/api/devices/", s.handleDevice)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSession)
	mux.HandleFunc("/api/shutdown", s.handleShutdown)
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/hooks/claude/notification", s.handleHookNotification)
	mux.HandleFunc("/hooks/claude/permission", s.handleHookPermission)
	return mux
}

// Start begins serving on cfg.Port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	s.httpSrv = &http.Server{Addr: addr, Handler: s.Handler()}
	s.log.Printf("riffpad daemon %s listening on http://%s", version, addr)
	if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeRaw(w, http.StatusOK, "text/html; charset=utf-8", webui.IndexHTML)
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/app.js":
		writeRaw(w, http.StatusOK, "application/javascript", webui.AppJS)
	case "/style.css":
		writeRaw(w, http.StatusOK, "text/css; charset=utf-8", webui.StyleCSS)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	n := len(s.sessions)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":  version,
		"port":     s.cfg.Port,
		"sessions": n,
		"uptime":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleCreatePairing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	code := newPairingCode()
	s.mu.Lock()
	s.pending[code] = pendingPair{Code: code, Expires: time.Now().Add(10 * time.Minute)}
	s.mu.Unlock()
	url := fmt.Sprintf("http://127.0.0.1:%d/?pair=%s", s.cfg.Port, code)
	writeJSON(w, http.StatusOK, map[string]any{
		"code":      code,
		"url":       url,
		"expiresAt": time.Now().Add(10 * time.Minute).Format(time.RFC3339),
	})
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
		"deviceId":         dev.ID,
		"serverPublicKey":  protocol.EncodeKey(identity.PublicKey),
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

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		list := make([]map[string]any, 0, len(s.sessions))
		for _, sess := range s.sessions {
			list = append(list, map[string]any{
				"id":     sess.id,
				"name":   sess.meta.Name,
				"cli":    sess.meta.CLI,
				"cwd":    sess.meta.Cwd,
				"status": sess.status,
			})
		}
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
	case http.MethodPost:
		s.createSession(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		CLI    string `json:"cli"`
		Cwd    string `json:"cwd"`
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.CLI == "" {
		req.CLI = "claude"
	}
	if req.Cwd == "" {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		req.Cwd = cwd
	}
	id := protocol.NewID()
	createReq := adapter.CreateRequest{
		ID:       id,
		Name:     req.Name,
		CLI:      req.CLI,
		Cwd:      req.Cwd,
		Prompt:   req.Prompt,
		DataDir:  s.dataDir,
		HookBase: fmt.Sprintf("http://127.0.0.1:%d", s.cfg.Port),
	}
	factory := s.factory
	if factory == nil {
		factory = DefaultFactory()
	}
	sessAdapter, err := factory(context.Background(), createReq)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sess := &session{
		id:      id,
		meta:    sessAdapter.Meta(),
		adapter: sessAdapter,
		events:  sessAdapter.Events(),
		status:  protocol.StatusRunning,
		clients: map[*client]struct{}{},
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	startEv, err := protocol.NewEvent(id, protocol.EventSessionStart, sess.meta)
	if err == nil {
		s.pumpEvent(sess, startEv)
	}
	go func() {
		if err := sessAdapter.Start(context.Background()); err != nil {
			s.log.Printf("session %s start error: %v", id, err)
			ev, _ := protocol.NewEvent(id, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "start_error: " + err.Error()})
			s.pumpEvent(sess, ev)
			return
		}
	}()
	go s.pump(sess)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":     id,
		"name":   req.Name,
		"cli":    req.CLI,
		"cwd":    req.Cwd,
		"status": sess.status,
		"url":    fmt.Sprintf("http://127.0.0.1:%d/?session=%s", s.cfg.Port, id),
	})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if !strings.HasSuffix(id, "/stop") {
		http.NotFound(w, r)
		return
	}
	id = strings.TrimSuffix(id, "/stop")
	s.mu.Lock()
	sess, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	_ = sess.adapter.Stop()
	writeJSON(w, http.StatusOK, map[string]any{"stopped": true})
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shuttingDown": true})
	go func() {
		time.Sleep(100 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	}()
}

func (s *Server) pump(sess *session) {
	for ev := range sess.events {
		s.pumpEvent(sess, ev)
	}
}

func (s *Server) pumpEvent(sess *session, ev protocol.Event) {
	if ev.Type == protocol.EventSessionEnd || ev.Type == protocol.EventAgentStatus {
		var p protocol.AgentStatusPayload
		_ = ev.DecodePayload(&p)
		if ev.Type == protocol.EventSessionEnd {
			sess.status = protocol.StatusDone
		} else if p.Status != "" {
			sess.status = p.Status
		}
	}
	sess.addEvent(ev)
	sess.broadcast(ev)
}

func (s *Server) getSession(id string) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[id]
}

func (s *Server) loadDevices() {
	data, err := os.ReadFile(filepath.Join(s.dataDir, "devices.json"))
	if err != nil {
		return
	}
	var list []Device
	if json.Unmarshal(data, &list) != nil {
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
	return os.WriteFile(filepath.Join(s.dataDir, "devices.json"), data, 0o600)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeRaw(w http.ResponseWriter, status int, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

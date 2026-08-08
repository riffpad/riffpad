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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/claude"
	"github.com/riffpad/riffpad/apps/daemon/internal/codex"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/kimi"
	"github.com/riffpad/riffpad/packages/protocol"
	"github.com/riffpad/riffpad/packages/webui"
)

const version = "0.1.0-m0"

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

type session struct {
	id       string
	meta     protocol.SessionStartPayload
	adapter  adapter.Session
	events   <-chan protocol.Event
	status   string
	ended    bool
	lease    bool      // local TUI attached: session closes when heartbeat lapses
	lastHB   time.Time // last lease heartbeat from the local CLI
	lastSeen time.Time // last event activity (for dashboard "recent" display)
	created  time.Time
	connect  map[string]string // adapter connect info for restart recovery (e.g. codex socket/threadId)
	mu       sync.Mutex
	seq      uint64 // last assigned event sequence number (#173)
	history  []protocol.Event
	clients  map[*client]struct{}
}

// Server is the local daemon HTTP/WS server.
type Server struct {
	cfg         *config.Config
	keys        *config.Keys
	dataDir     string
	log         *log.Logger
	factory     adapter.Factory
	httpSrv     *http.Server
	startedAt   time.Time
	sweepDone   chan struct{}
	rc          *relayClient
	relayCancel context.CancelFunc
	token       string // local API auth token (see localAuth)

	mu           sync.Mutex
	devices      map[string]Device
	pending      map[string]pendingPair
	sessions     map[string]*session
	pendingHooks map[string]chan string
	messageBuf   map[string]string
}

// New creates a daemon server.
func New(cfg *config.Config, keys *config.Keys, dataDir string, logger *log.Logger, factory adapter.Factory) *Server {
	if cfg.LocalToken == "" {
		// config.Load normally guarantees a persisted token; generate an
		// in-memory one for directly-constructed configs (e.g. tests).
		cfg.LocalToken = config.NewLocalToken()
	}
	s := &Server{
		cfg:          cfg,
		keys:         keys,
		dataDir:      dataDir,
		log:          logger,
		factory:      factory,
		startedAt:    time.Now(),
		sweepDone:    make(chan struct{}),
		token:        cfg.LocalToken,
		devices:      map[string]Device{},
		pending:      map[string]pendingPair{},
		sessions:     map[string]*session{},
		pendingHooks: map[string]chan string{},
		messageBuf:   map[string]string{},
	}
	s.loadDevices()
	s.cleanupCodexProcesses()
	s.restoreSessions()
	if cfg.RelayURL != "" {
		hostID := cfg.HostID
		if hostID == "" {
			if hn, err := os.Hostname(); err == nil {
				hostID = hn
			} else {
				hostID = "riffpad-host"
			}
		}
		s.rc = newRelayClient(cfg.RelayURL, hostID, cfg.HostSecret, logger, s.handleRelayJoin)
	}
	return s
}

// cleanupCodexProcesses kills app-server processes left over from an unclean
// daemon shutdown and removes their stale socket/pid files.
func (s *Server) cleanupCodexProcesses() {
	dir := filepath.Join(s.dataDir, "codex")
	// Sockets belonging to persisted sessions may be reused by
	// restoreSessions after a daemon restart; do not kill those app-servers
	// (their attached TUI must survive the restart).
	keep := map[string]bool{}
	if persisted, err := loadPersistedSessions(s.dataDir); err == nil {
		for _, ps := range persisted {
			if sock := ps.Connect["socket"]; sock != "" {
				keep[sock] = true
			}
		}
	}
	// Kill leftover app-server processes by pid file (written by current
	// adapters) and, on Linux, by scanning /proc cmdlines for any app-server
	// listening under our codex dir (covers processes spawned before pid
	// files existed, e.g. unclean upgrades).
	if runtime.GOOS == "linux" {
		s.killCodexByProcScan(dir, keep)
	}
	pidFiles, err := filepath.Glob(filepath.Join(dir, "*.pid"))
	if err != nil {
		return
	}
	for _, pf := range pidFiles {
		sock := strings.TrimSuffix(pf, ".pid")
		if keep[sock] {
			continue
		}
		data, err := os.ReadFile(pf)
		if err != nil {
			continue
		}
		var pid int
		n, _ := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
		if n != 1 || pid <= 0 {
			_ = os.Remove(pf)
			continue
		}
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Kill()
		}
		_ = os.Remove(pf)
		_ = os.Remove(strings.TrimSuffix(pf, ".pid"))
		s.log.Printf("cleaned up stale codex app-server pid=%d", pid)
	}
	// Remove any remaining stale sockets (their processes, if any, were
	// killed above or by pid file).
	if socks, err := filepath.Glob(filepath.Join(dir, "*.sock")); err == nil {
		for _, sf := range socks {
			if keep[sf] {
				continue
			}
			_ = os.Remove(sf)
		}
	}
}

func (s *Server) killCodexByProcScan(dir string, keep map[string]bool) {
	procs, err := filepath.Glob("/proc/[0-9]*/cmdline")
	if err != nil {
		return
	}
	for _, cmdlinePath := range procs {
		data, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}
		cmdline := strings.ReplaceAll(string(data), "\x00", " ")
		if !strings.Contains(cmdline, "app-server") || !strings.Contains(cmdline, dir) {
			continue
		}
		skip := false
		for sock := range keep {
			if strings.Contains(cmdline, sock) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		pidStr := strings.Trim(filepath.Base(filepath.Dir(cmdlinePath)), "/")
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			continue
		}
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Kill()
		}
		s.log.Printf("cleaned up stale codex app-server pid=%d (proc scan)", pid)
	}
}

// DefaultFactory maps CLI names to adapters.
func DefaultFactory() adapter.Factory {
	return func(_ context.Context, req adapter.CreateRequest) (adapter.Session, error) {
		switch req.CLI {
		case "", "claude":
			return claude.New(req), nil
		case "kimi":
			return kimi.New(req), nil
		case "codex":
			return codex.New(req), nil
		default:
			return nil, fmt.Errorf("unsupported cli %q", req.CLI)
		}
	}
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/assets/", s.handleAsset)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/pairings", s.handleCreatePairing)
	mux.HandleFunc("/api/pair", s.handlePair)
	mux.HandleFunc("/api/devices", s.handleDevices)
	mux.HandleFunc("/api/devices/", s.handleDevice)
	mux.HandleFunc("/api/killswitch", s.handleKillswitch)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSession)
	mux.HandleFunc("/api/shutdown", s.handleShutdown)
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/hooks/claude/notification", s.handleHookNotification)
	mux.HandleFunc("/hooks/claude/permission", s.handleHookPermission)
	mux.HandleFunc("/hooks/claude/session-start", s.handleHookSessionStart)
	mux.HandleFunc("/hooks/claude/session-end", s.handleHookSessionEnd)
	mux.HandleFunc("/hooks/claude/pre-tool-use", s.handleHookPreToolUse)
	mux.HandleFunc("/hooks/claude/post-tool-use", s.handleHookPostToolUse)
	mux.HandleFunc("/hooks/claude/user-prompt-submit", s.handleHookUserPromptSubmit)
	mux.HandleFunc("/hooks/claude/message-display", s.handleHookMessageDisplay)
	return s.localAuth(mux)
}

// Start begins serving on cfg.Port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	s.httpSrv = &http.Server{Addr: addr, Handler: s.Handler()}
	s.log.Printf("riffpad daemon %s listening on http://%s", version, addr)
	go s.sweepLoop()
	if s.rc != nil {
		ctx, cancel := context.WithCancel(context.Background())
		s.relayCancel = cancel
		persistCreds := func(hostID, secret string) {
			s.cfg.HostID = hostID
			s.cfg.HostSecret = secret
			if err := config.Save(s.dataDir, s.cfg); err != nil {
				s.log.Printf("save host credentials: %v", err)
			}
		}
		persistToken := func(token string) {
			s.cfg.RelayToken = token
			if err := config.Save(s.dataDir, s.cfg); err != nil {
				s.log.Printf("save relay token: %v", err)
			}
		}
		if s.cfg.RelayToken == "" && s.cfg.RelayUser != "" && s.cfg.RelayPassword != "" {
			if err := s.rc.login(ctx, s.cfg.RelayUser, s.cfg.RelayPassword, persistToken); err != nil {
				s.log.Printf("relay login failed: %v", err)
				cancel()
			}
		} else {
			s.rc.setToken(s.cfg.RelayToken)
		}
		if s.rc != nil && s.relayCancel != nil {
			if err := s.rc.ensureRegistered(ctx, persistCreds); err != nil {
				s.log.Printf("relay registration failed: %v", err)
				cancel()
			} else {
				go s.rc.run(ctx)
				s.announceSessions()
			}
		}
	}
	if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.relayCancel != nil {
		s.relayCancel()
	}
	select {
	case <-s.sweepDone:
	default:
		close(s.sweepDone)
	}
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
	html, err := webui.IndexHTML()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "webui not built")
		return
	}
	writeRaw(w, http.StatusOK, "text/html; charset=utf-8", html)
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
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
	writeRaw(w, http.StatusOK, webui.ContentType(name), data)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	n := len(s.sessions)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":   version,
		"port":      s.cfg.Port,
		"sessions":  n,
		"startedAt": s.startedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleCreatePairing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if s.rc != nil {
		s.createRemotePairing(w)
		return
	}
	code := newPairingCode()
	s.mu.Lock()
	s.pending[code] = pendingPair{Code: code, Expires: time.Now().Add(10 * time.Minute)}
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

// handleKillswitch stops every agent session, clears all paired devices, and
// asks the relay to revoke cloud devices / disconnect viewers.
func (s *Server) handleKillswitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	s.mu.Lock()
	sessions := make([]*session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.devices = map[string]Device{}
	s.mu.Unlock()
	_ = s.saveDevices()
	for _, sess := range sessions {
		sess.mu.Lock()
		for c := range sess.clients {
			_ = c.transport.Close()
		}
		sess.mu.Unlock()
		s.stopSession(sess.id)
	}
	if s.rc != nil {
		_ = s.relayKillswitch()
	}
	s.log.Printf("killswitch: stopped %d sessions, revoked all devices", len(sessions))
	writeJSON(w, http.StatusOK, map[string]any{"killed": true, "sessions": len(sessions)})
}

func (s *Server) relayKillswitch() error {
	httpURL := s.cfg.RelayURL
	httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
	httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")
	body := strings.NewReader("{}")
	req, err := http.NewRequest(http.MethodPost,
		strings.TrimSuffix(httpURL, "/")+"/api/hosts/"+s.rc.hostID+"/killswitch", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.RelayToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.RelayToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		list := make([]map[string]any, 0, len(s.sessions))
		for _, sess := range s.sessions {
			if sess.ended {
				continue
			}
			list = append(list, map[string]any{
				"id":       sess.id,
				"name":     sess.meta.Name,
				"cli":      sess.meta.CLI,
				"cwd":      sess.meta.Cwd,
				"status":   sess.status,
				"lastSeen": sess.lastSeen,
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
		ID:        id,
		Name:      req.Name,
		CLI:       req.CLI,
		Cwd:       req.Cwd,
		Prompt:    req.Prompt,
		DataDir:   s.dataDir,
		HookBase:  fmt.Sprintf("http://127.0.0.1:%d", s.cfg.Port),
		HookToken: s.token,
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
		id:       id,
		meta:     sessAdapter.Meta(),
		adapter:  sessAdapter,
		events:   sessAdapter.Events(),
		status:   protocol.StatusRunning,
		ended:    false,
		created:  time.Now(),
		lastSeen: time.Now(),
		clients:  map[*client]struct{}{},
	}
	if req.Prompt == "" {
		sess.status = protocol.StatusWaitingInput
	}
	s.persistSession(sess)
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	s.announceSessions()
	startEv, err := protocol.NewEvent(id, protocol.EventSessionStart, sess.meta)
	if err == nil {
		s.pumpEvent(sess, startEv)
	}
	// Respond with the initial status before starting the adapter, so the
	// create response is deterministic (e.g. waiting_input for an empty
	// prompt) instead of racing with the first agent status event.
	writeJSON(w, http.StatusOK, map[string]any{
		"id":     id,
		"name":   req.Name,
		"cli":    req.CLI,
		"cwd":    req.Cwd,
		"status": sess.status,
		"url":    fmt.Sprintf("http://127.0.0.1:%d/?session=%s&token=%s", s.cfg.Port, id, s.token),
	})
	go func() {
		if err := sessAdapter.Start(context.Background()); err != nil {
			s.log.Printf("session %s start error: %v", id, err)
			ev, _ := protocol.NewEvent(id, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "start_error: " + err.Error()})
			s.pumpEvent(sess, ev)
			return
		}
	}()
	go s.pump(sess)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(id, "/connect") {
		s.handleSessionConnect(w, r, strings.TrimSuffix(id, "/connect"))
		return
	}
	if strings.HasSuffix(id, "/heartbeat") {
		s.handleSessionHeartbeat(w, strings.TrimSuffix(id, "/heartbeat"))
		return
	}
	if !strings.HasSuffix(id, "/stop") {
		http.NotFound(w, r)
		return
	}
	id = strings.TrimSuffix(id, "/stop")
	s.mu.Lock()
	_, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	s.stopSession(id)
	writeJSON(w, http.StatusOK, map[string]any{"stopped": true})
}

// handleSessionHeartbeat renews the local-TUI lease for a session. The first
// heartbeat enables the lease; once enabled, the session is closed if no
// heartbeat arrives within the lease window (see sweepOnce).
func (s *Server) handleSessionHeartbeat(w http.ResponseWriter, id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	if ok {
		sess.lease = true
		sess.lastHB = time.Now()
	}
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// stopSession removes a session from the live list and stops its adapter.
func (s *Server) stopSession(id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mu.Unlock()
	if !ok {
		return
	}
	sess.ended = true
	s.persistSession(sess)
	s.announceSessions()
	_ = sess.adapter.Stop()
	s.log.Printf("session %s stopped", id)
}

// handleSessionConnect returns the local connect info (app-server socket +
// thread id) for adapters that support attaching a local TUI, waiting until
// the session is ready.
func (s *Server) handleSessionConnect(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	ci, ok := sess.adapter.(interface {
		ConnectInfo() (socket string, threadID string, err error)
	})
	if !ok {
		writeError(w, http.StatusBadRequest, "session does not support TUI attach")
		return
	}
	socket, threadID, err := ci.ConnectInfo()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"socket": socket, "threadId": threadID})
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
	sess.lastSeen = time.Now()
	if ev.Type == protocol.EventSessionEnd || ev.Type == protocol.EventAgentStatus {
		var p protocol.AgentStatusPayload
		_ = ev.DecodePayload(&p)
		if ev.Type == protocol.EventSessionEnd {
			sess.status = protocol.StatusDone
			sess.ended = true
			s.announceSessions()
		} else if p.Status != "" {
			sess.status = p.Status
		}
	}
	ev = sess.addEvent(ev)
	sess.broadcast(ev)
	s.persistEvent(sess, ev)
	if ev.Type == protocol.EventSessionEnd {
		s.persistSession(sess)
	}
}

func (s *Server) sweepLoop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.sweepDone:
			return
		case <-t.C:
			s.sweepOnce()
		}
	}
}

func (s *Server) sweepOnce() {
	s.mu.Lock()
	sessions := make([]*session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.mu.Unlock()
	for _, sess := range sessions {
		s.mu.Lock()
		leaseExpired := sess.lease && time.Since(sess.lastHB) > 20*time.Second
		alreadyEnded := sess.ended
		s.mu.Unlock()
		if leaseExpired && !alreadyEnded {
			s.log.Printf("session %s lease expired (no local TUI heartbeat); closing", sess.id)
			s.stopSession(sess.id)
			continue
		}
		if sess.status != protocol.StatusRunning {
			continue
		}
		if !sess.adapter.Alive() {
			ev, _ := protocol.NewEvent(sess.id, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "process_exit"})
			s.pumpEvent(sess, ev)
			s.log.Printf("session %s marked ended (process gone)", sess.id)
			s.announceSessions()
		}
	}
}

func (s *Server) handleRelayJoin(ji RelayJoin) {
	ephPub, err := protocol.DecodeKey(ji.Eph)
	if err != nil {
		s.log.Printf("relay join invalid eph session=%s", ji.SessionID)
		return
	}
	devPub, err := protocol.DecodeKey(ji.Pub)
	if err != nil {
		s.log.Printf("relay join invalid pub session=%s", ji.SessionID)
		return
	}
	tr := s.rc.viewerTransport(ji.ViewerID)
	if err := s.attachViewer(tr, ji.DeviceID, ji.SessionID, ephPub, ji.Curve, devPub); err != nil {
		s.log.Printf("relay join rejected session=%s device=%s: %v", ji.SessionID, ji.DeviceID, err)
	}
}

func (s *Server) announceSessions() {
	if s.rc == nil {
		return
	}
	s.mu.Lock()
	list := make([]RelaySession, 0, len(s.sessions))
	for _, sess := range s.sessions {
		if sess.ended {
			continue
		}
		list = append(list, RelaySession{
			ID: sess.id, Name: sess.meta.Name, CLI: sess.meta.CLI,
			Cwd: sess.meta.Cwd, Status: sess.status,
		})
	}
	s.mu.Unlock()
	s.rc.announce(list)
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

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

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/claude"
	"github.com/riffpad/riffpad/apps/daemon/internal/codex"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/demo"
	"github.com/riffpad/riffpad/apps/daemon/internal/kimi"
	"github.com/riffpad/riffpad/apps/daemon/internal/version"
	"github.com/riffpad/riffpad/packages/protocol"
	"github.com/riffpad/riffpad/packages/webui"
)

// sess.mu guards every mutable field on session (status, ended, lastSeen,
// lease, lastHB, leaseMissed, connect, adapter, seq, history, clients). The
// snapshot/update helpers below are the only place those fields are touched
// outside the lock, so no caller has to remember the rule (#312).

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
	ptys         map[string]*websocket.Conn
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
		ptys:         map[string]*websocket.Conn{},
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
		case "demo":
			return demo.New(req), nil
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
	mux.HandleFunc("/hooks/claude/post-tool-use-failure", s.handleHookPostToolUseFailure)
	mux.HandleFunc("/hooks/claude/user-prompt-submit", s.handleHookUserPromptSubmit)
	mux.HandleFunc("/hooks/claude/message-display", s.handleHookMessageDisplay)
	mux.HandleFunc("/hooks/claude/stop", s.handleHookStop)
	mux.HandleFunc("/hooks/kimi/", s.handleKimiHook)
	return s.localAuth(mux)
}

// Start begins serving on cfg.Port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	s.httpSrv = &http.Server{Addr: addr, Handler: s.Handler()}
	s.log.Printf("riffpad daemon %s listening on http://%s", version.Version, addr)
	go s.sweepLoop()
	if s.rc != nil {
		ctx, cancel := context.WithCancel(context.Background())
		s.relayCancel = cancel
		persistCreds := func(hostID, secret string) {
			s.cfg.HostID = hostID
			s.cfg.HostSecret = secret
			// Merge into the on-disk config under a lock: `riffpad login` may
			// be writing at the same time, and a blind Save would clobber its
			// fresh token (last-write-wins, #172).
			if err := config.Update(s.dataDir, func(c *config.Config) {
				c.HostID = hostID
				c.HostSecret = secret
			}); err != nil {
				s.log.Printf("save host credentials: %v", err)
			}
		}
		persistToken := func(token string) {
			s.cfg.RelayToken = token
			if err := config.Update(s.dataDir, func(c *config.Config) {
				c.RelayToken = token
			}); err != nil {
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
	// Reclaim agent processes the daemon itself spawned (#170): their Start
	// ctx is context.Background(), so without this they outlive the daemon as
	// orphans and pending approvals hang forever. Attach sessions run in the
	// user's own tmux/terminal — never kill those.
	s.mu.Lock()
	sessions := make([]*session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.mu.Unlock()
	for _, sess := range sessions {
		managed, ended := sess.managed, sess.isEnded()
		if managed && !ended {
			s.log.Printf("shutdown: stopping spawned session %s", sess.id)
			_ = sess.getAdapter().Stop()
		}
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
		"version":   version.Version,
		"port":      s.cfg.Port,
		"sessions":  n,
		"startedAt": s.startedAt.Format(time.RFC3339),
	})
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
		sess.mu.Lock()
		leaseExpired := sess.lease && time.Since(sess.lastHB) > 20*time.Second
		alreadyEnded := sess.ended
		// Grace period (#170): after a system sleep the sweep ticker and the
		// TUI heartbeat come due at the same time and the sweep can win the
		// race, killing a session whose TUI is still open. Require the lease
		// to stay expired across two consecutive sweeps before closing.
		grace := leaseExpired && !alreadyEnded && !sess.leaseMissed
		if grace {
			sess.leaseMissed = true
		}
		status := sess.status
		sess.mu.Unlock()
		if grace {
			s.log.Printf("session %s lease expired; granting one sweep period before closing", sess.id)
			continue
		}
		if leaseExpired && !alreadyEnded {
			s.log.Printf("session %s lease expired (no local TUI heartbeat); closing", sess.id)
			s.stopSession(sess.id)
			continue
		}
		if status != protocol.StatusRunning {
			continue
		}
		if _, isAttach := sess.getAdapter().(*attachAdapter); isAttach {
			// attachAdapter.Alive is always true (the process lives in the
			// user's tmux), so a `kill -9` on claude — where the SessionEnd
			// hook never fires — would otherwise leave the session "running"
			// forever. Hook activity is the only liveness signal we have; if
			// none arrived within attachIdleTimeout, mark the session ended.
			// False positives (user left claude idle) are harmless: the next
			// hook revives the session in attachSession (#170).
			sess.mu.Lock()
			idleFor := time.Since(sess.lastSeen)
			sess.mu.Unlock()
			if idleFor > attachIdleTimeout {
				ev, _ := protocol.NewEvent(sess.id, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "no_activity"})
				s.pumpEvent(sess, ev)
				s.log.Printf("session %s marked ended (no hook activity for %s)", sess.id, idleFor.Round(time.Second))
				s.announceSessions()
			}
			continue
		}
		if !sess.getAdapter().Alive() {
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
		sess.mu.Lock()
		if sess.ended {
			sess.mu.Unlock()
			continue
		}
		lastSeen := sess.lastSeen
		status := sess.status
		sess.mu.Unlock()
		list = append(list, RelaySession{
			ID: sess.id, Name: sess.meta.Name, CLI: sess.meta.CLI,
			Cwd: sess.meta.Cwd, Status: status, LastSeenAt: lastSeen,
		})
	}
	s.mu.Unlock()
	s.rc.announce(list)
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

package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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

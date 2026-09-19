// Package codex implements the Codex adapter over the official
// `codex app-server` JSON-RPC protocol. The daemon spawns a remote-control
// app-server on a unix socket per session, creates a thread, and exposes
// connect info so the local CLI can attach `codex resume --remote` and keep
// the TUI in the user's terminal (no-silent hosting). Command/file/permission
// approvals arrive as server-initiated JSON-RPC requests and are answered by
// the daemon.
package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/version"
	"github.com/riffpad/riffpad/packages/protocol"
)

type pendingApproval struct {
	kind      string // command | file | permissions
	action    string
	summary   string
	requested []string // permissions request: requested permission names
}

// approvalTimeout bounds how long an approval request waits for a viewer
// decision before defaulting to decline, mirroring the attach hook path.
// A var so tests can shrink it.
var approvalTimeout = 10 * time.Minute

// Codex is a Codex session driven through the app-server protocol.
type Codex struct {
	id      string
	name    string
	cwd     string
	prompt  string
	binary  string
	dataDir string
	events  chan protocol.Event
	stopCh  chan struct{}
	doneCh  chan struct{}
	ready   chan struct{}

	mu            sync.Mutex
	ctx           context.Context
	cmd           *exec.Cmd
	conn          *websocket.Conn
	socketPath    string
	sendFn        func(data []byte) error
	launched      bool
	exited        bool
	threadID      string
	restoreThread string // non-empty when reconnecting to an existing thread after daemon restart
	adoptedPID    int    // pid of a surviving app-server reused across daemon restarts
	turnActive    bool
	promptSent    bool
	initError     error
	nextRequestID int
	pendingRes    map[int]chan json.RawMessage
	pending       map[string]pendingApproval
	messages      map[string]*strings.Builder
	knownThreads  map[string]bool
}

// New creates a Codex app-server session adapter.
func New(req adapter.CreateRequest) *Codex {
	binary := req.Binary
	if binary == "" {
		binary = "codex"
	}
	return &Codex{
		id:            req.ID,
		name:          req.Name,
		cwd:           req.Cwd,
		prompt:        req.Prompt,
		binary:        binary,
		dataDir:       req.DataDir,
		events:        make(chan protocol.Event, 256),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		ready:         make(chan struct{}),
		nextRequestID: 1,
		pendingRes:    make(map[int]chan json.RawMessage),
		pending:       make(map[string]pendingApproval),
		messages:      make(map[string]*strings.Builder),
		knownThreads:  make(map[string]bool),
	}
}

func (c *Codex) ID() string                    { return c.id }
func (c *Codex) Events() <-chan protocol.Event { return c.events }
func (c *Codex) Meta() protocol.SessionStartPayload {
	return protocol.SessionStartPayload{Name: c.name, CLI: "codex", Cwd: c.cwd}
}

// Start records the context, spawns the app-server, and (when a prompt is
// present) waits for the thread to be ready before sending it.
func (c *Codex) Start(ctx context.Context) error {
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
	if err := c.ensureStarted(); err != nil {
		return err
	}
	if c.prompt != "" {
		if err := c.waitReady(); err != nil {
			return err
		}
		return c.SendPrompt(c.prompt)
	}
	return nil
}

func (c *Codex) ensureStarted() error {
	c.mu.Lock()
	if c.launched {
		c.mu.Unlock()
		return nil
	}
	if c.ctx == nil {
		c.mu.Unlock()
		return fmt.Errorf("session not started")
	}
	c.launched = true
	ctx := c.ctx
	c.mu.Unlock()
	return c.spawn(ctx)
}

func (c *Codex) spawn(ctx context.Context) error {
	dir := filepath.Join(c.dataDir, "codex")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create codex dir: %w", err)
	}
	socketPath := filepath.Join(dir, c.id+".sock")
	cmd := exec.CommandContext(ctx, c.binary, "app-server", "--remote-control", "--listen", "unix://"+socketPath)
	if c.cwd != "" {
		cmd.Dir = c.cwd
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s app-server: %w", c.binary, err)
	}
	// Record the app-server pid so a later daemon start can clean up
	// processes left behind by an unclean shutdown (SIGKILL etc).
	_ = os.WriteFile(socketPath+".pid", []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o600)
	c.mu.Lock()
	c.cmd = cmd
	c.socketPath = socketPath
	c.mu.Unlock()
	go c.copyStderr(stderr)
	go func() {
		<-c.stopCh
		if c.cmd != nil && c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
	}()
	if err := c.connect(ctx, socketPath); err != nil {
		return err
	}
	_ = c.request("initialize", map[string]any{
		"clientInfo": map[string]any{"name": "riffpad", "version": version.Version},
	})
	return nil
}

// connect waits for the app-server unix socket and establishes the JSON-RPC
// WebSocket connection.
func (c *Codex) connect(ctx context.Context, socketPath string) error {
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("codex app-server socket not ready")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.Dial("ws://codex/", http.Header{})
	if err != nil {
		return fmt.Errorf("dial codex app-server: %w", err)
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	go c.readLoop(conn)
	return nil
}

// ConnectInfo returns the local app-server socket and thread id so the CLI can
// attach the Codex TUI (`codex resume --remote unix://… <threadId>`).
func (c *Codex) ConnectInfo() (socket string, threadID string, err error) {
	if err := c.waitReady(); err != nil {
		return "", "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.socketPath, c.threadID, nil
}

// Restore reattaches this adapter to an existing thread (e.g. after a daemon
// restart). The app-server is spawned again and the thread is resumed instead
// of creating a new one.
func (c *Codex) Restore(threadID string) error {
	c.mu.Lock()
	if c.ctx == nil {
		c.ctx = context.Background()
	}
	c.threadID = threadID
	c.restoreThread = threadID
	c.knownThreads[threadID] = true
	c.prompt = "" // conversation history already exists
	c.mu.Unlock()

	// Prefer reusing a surviving app-server: a daemon restart must not drop
	// the TUI connection attached to it.
	socketPath := filepath.Join(c.dataDir, "codex", c.id+".sock")
	if _, err := os.Stat(socketPath); err == nil {
		if pid, ok := readPIDFile(socketPath + ".pid"); ok {
			c.mu.Lock()
			c.adoptedPID = pid
			c.socketPath = socketPath
			c.launched = true
			c.mu.Unlock()
			go func() {
				<-c.stopCh
				if c.adoptedPID > 0 {
					killProcess(c.adoptedPID)
				}
			}()
			if err := c.connect(c.ctx, socketPath); err == nil {
				return c.request("initialize", map[string]any{
					"clientInfo": map[string]any{"name": "riffpad", "version": "0.1.0"},
				})
			}
			// Surviving socket but dead process: fall through to a fresh spawn.
			c.mu.Lock()
			c.launched = false
			c.adoptedPID = 0
			c.mu.Unlock()
		}
	}
	_ = os.Remove(socketPath)
	_ = os.Remove(socketPath + ".pid")
	return c.ensureStarted()
}

func readPIDFile(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var pid int
	n, _ := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	return pid, n == 1 && pid > 0
}

// CurrentConnect returns the live socket path and thread id without waiting.
func (c *Codex) CurrentConnect() (socket string, threadID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.socketPath, c.threadID
}

// Stop terminates the app-server.
func (c *Codex) Stop() error {
	c.mu.Lock()
	launched := c.launched
	c.mu.Unlock()
	select {
	case <-c.stopCh:
		return nil
	default:
		close(c.stopCh)
	}
	if !launched {
		return nil
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	if c.adoptedPID > 0 {
		killProcess(c.adoptedPID)
	}
	c.mu.Lock()
	if c.socketPath != "" {
		_ = os.Remove(c.socketPath + ".pid")
		_ = os.Remove(c.socketPath)
	}
	c.mu.Unlock()
	<-c.doneCh
	return nil
}

// killProcess terminates a process on any platform (SIGKILL on Unix,
// TerminateProcess on Windows).
func killProcess(pid int) {
	if pid <= 0 {
		return
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}

// Alive reports whether the app-server process is running.
func (c *Codex) Alive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.launched && !c.exited
}

// SendPrompt starts a new turn, or steers the in-flight turn.
func (c *Codex) SendPrompt(text string) error {
	if err := c.ensureStarted(); err != nil {
		return err
	}
	if err := c.waitReady(); err != nil {
		return err
	}
	c.mu.Lock()
	needResume := c.promptSent
	c.promptSent = true
	c.mu.Unlock()
	if needResume {
		// Re-attach to the thread before injecting: app-server only streams
		// turn/item events to connections subscribed to the thread. After the
		// local TUI resumes the same thread, our subscription must be
		// refreshed or events silently stop reaching the daemon (and phone).
		// The first prompt skips this: thread/start already subscribed and a
		// fresh thread has no rollout yet (resume would fail).
		if _, err := c.requestSync("thread/resume", map[string]any{"threadId": c.threadID}, 5*time.Second); err != nil {
			log.Printf("codex[%s] thread/resume: %v", c.id, err)
		}
	}
	input := []any{map[string]any{"type": "text", "text": text}}
	c.mu.Lock()
	active := c.turnActive
	c.mu.Unlock()
	if active {
		return c.request("turn/steer", map[string]any{
			"threadId": c.threadID,
			"input":    input,
		})
	}
	return c.request("turn/start", map[string]any{
		"threadId": c.threadID,
		"input":    input,
	})
}

// SendApproval resolves a pending app-server approval request.
func (c *Codex) SendApproval(requestID, decision string) error {
	c.mu.Lock()
	p, ok := c.pending[requestID]
	if ok {
		delete(c.pending, requestID)
	}
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending approval %s", requestID)
	}
	var result any
	switch p.kind {
	case "command", "file":
		d := "decline"
		if decision == "approve" {
			d = "accept"
		}
		result = map[string]any{"decision": d}
	case "permissions":
		granted := []any{}
		if decision == "approve" {
			for _, name := range p.requested {
				granted = append(granted, name)
			}
		}
		result = map[string]any{"permissions": granted, "scope": "turn"}
	default:
		return fmt.Errorf("unsupported approval kind %q", p.kind)
	}
	return c.writeJSON(map[string]any{"id": requestID, "result": result})
}

func (c *Codex) waitReady() error {
	select {
	case <-c.ready:
		c.mu.Lock()
		err := c.initError
		tid := c.threadID
		c.mu.Unlock()
		if err != nil {
			return err
		}
		if tid == "" {
			return fmt.Errorf("codex thread not initialized")
		}
		return nil
	case <-c.stopCh:
		return fmt.Errorf("session stopped")
	case <-time.After(30 * time.Second):
		return fmt.Errorf("codex app-server initialization timed out")
	}
}

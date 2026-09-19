// Package kimi implements the Kimi Code adapter over the Agent Client
// Protocol (ACP). The daemon spawns `kimi acp` (a stdio JSON-RPC server) and
// acts as an ACP client: initialize -> session/new -> session/prompt, with
// streamed session/update notifications and session/request_permission
// approvals. No tmux is required.
package kimi

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/version"
	"github.com/riffpad/riffpad/packages/protocol"
)

type pendingApproval struct {
	sessionID string
	action    string
	summary   string
	options   map[string]string // optionId -> name
	kinds     map[string]string // optionId -> kind (allow_once / reject_once / ...)
}

type pendingTool struct {
	name    string
	summary string
	args    map[string]any
}

// approvalTimeout bounds how long a permission prompt waits for a viewer
// decision before defaulting to reject, mirroring the attach hook path.
// A var so tests can shrink it.
var approvalTimeout = 10 * time.Minute

// Kimi is a Kimi Code session driven through the ACP stdio server.
type Kimi struct {
	id      string
	name    string
	cwd     string
	prompt  string
	binary  string
	dataDir string
	events  chan protocol.Event
	stopCh  chan struct{}
	doneCh  chan struct{}
	readyCh chan struct{}

	mu            sync.Mutex
	interactive   bool
	ctx           context.Context
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	pty           *os.File
	ptySubs       map[*ptyTerm]struct{}
	hookBase      string
	hookToken     string
	configPath    string
	launched      bool
	exited        bool
	sessionID     string
	initError     error
	nextRequestID int
	pending       map[string]pendingApproval
	pendingTools  map[string]pendingTool
	msgBuf        strings.Builder
	msgActive     bool
	turnActive    bool // true between a prompt and the turn result (client "running" indicator)
}

// New creates a Kimi Code ACP session adapter.
func New(req adapter.CreateRequest) *Kimi {
	binary := req.Binary
	if binary == "" {
		binary = "kimi"
	}
	return &Kimi{
		id:            req.ID,
		name:          req.Name,
		cwd:           req.Cwd,
		prompt:        req.Prompt,
		binary:        binary,
		dataDir:       req.DataDir,
		events:        make(chan protocol.Event, 256),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		readyCh:       make(chan struct{}),
		nextRequestID: 1,
		interactive:   true,
		ptySubs:       make(map[*ptyTerm]struct{}),
		hookBase:      req.HookBase,
		hookToken:     req.HookToken,
		pending:       make(map[string]pendingApproval),
		pendingTools:  make(map[string]pendingTool),
	}
}

// ptyTerm is one console attached to the interactive TUI. The type is
// platform-neutral; the PTY-backed methods live in tui_unix.go.
type ptyTerm struct {
	k  *Kimi
	ch chan []byte
}

func (k *Kimi) ID() string                    { return k.id }
func (k *Kimi) Events() <-chan protocol.Event { return k.events }
func (k *Kimi) Meta() protocol.SessionStartPayload {
	return protocol.SessionStartPayload{Name: k.name, CLI: "kimi", Cwd: k.cwd}
}

// Start records the context and spawns `kimi acp`. The initial prompt is sent
// once the ACP session is ready.
func (k *Kimi) Start(ctx context.Context) error {
	k.mu.Lock()
	k.ctx = ctx
	k.mu.Unlock()
	if err := k.ensureStarted(); err != nil {
		return err
	}
	if k.interactive {
		if k.prompt != "" {
			return k.SendPrompt(k.prompt)
		}
		return nil
	}
	if k.prompt != "" {
		if err := k.waitReady(); err != nil {
			return err
		}
		return k.SendPrompt(k.prompt)
	}
	return nil
}

func (k *Kimi) ensureStarted() error {
	k.mu.Lock()
	if k.launched {
		k.mu.Unlock()
		return nil
	}
	if k.ctx == nil {
		k.mu.Unlock()
		return fmt.Errorf("session not started")
	}
	k.launched = true
	ctx := k.ctx
	k.mu.Unlock()
	return k.spawn(ctx)
}

func (k *Kimi) spawn(ctx context.Context) error {
	if k.interactive {
		if err := k.spawnInteractive(ctx); err == nil {
			return nil
		} else {
			log.Printf("kimi[%s] interactive spawn failed (%v); falling back to ACP", k.id, err)
		}
	}
	cmd := exec.CommandContext(ctx, k.binary, "acp")
	if k.cwd != "" {
		cmd.Dir = k.cwd
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s acp: %w", k.binary, err)
	}
	k.mu.Lock()
	k.cmd = cmd
	k.stdin = stdin
	k.mu.Unlock()
	go k.readLoop(stdout)
	go k.copyStderr(stderr)
	go func() {
		<-k.stopCh
		if k.cmd != nil && k.cmd.Process != nil {
			_ = k.cmd.Process.Kill()
		}
	}()
	// ACP initialization: protocol negotiation, then session creation.
	_ = k.request("initialize", map[string]any{
		"protocolVersion": 1,
		"clientCapabilities": map[string]any{
			"fs":       map[string]any{"readTextFile": false, "writeTextFile": false},
			"terminal": false,
		},
		"clientInfo": map[string]any{"name": "riffpad", "version": version.Version},
	})
	return nil
}

// Stop terminates the ACP server.
func (k *Kimi) Stop() error {
	k.mu.Lock()
	launched := k.launched
	k.mu.Unlock()
	select {
	case <-k.stopCh:
		return nil
	default:
		close(k.stopCh)
	}
	if !launched {
		return nil
	}
	if k.cmd != nil && k.cmd.Process != nil {
		_ = k.cmd.Process.Kill()
	}
	if k.pty != nil {
		_ = k.pty.Close()
	}
	<-k.doneCh
	return nil
}

// Alive reports whether the ACP server process is running.
func (k *Kimi) Alive() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.launched && !k.exited
}

// SendPrompt sends a text prompt to the current ACP session.
func (k *Kimi) SendPrompt(text string) error {
	if err := k.ensureStarted(); err != nil {
		return err
	}
	if k.interactive && k.pty != nil {
		if _, err := fmt.Fprintf(k.pty, "%s\r", text); err != nil {
			return err
		}
		return nil
	}
	if err := k.waitReady(); err != nil {
		return err
	}
	// Flip to running immediately so the client shows the activity indicator
	// from the moment the prompt is sent, before any session/update arrives
	// (#255). The turn result flips it back to waiting_input.
	k.mu.Lock()
	first := !k.turnActive
	k.turnActive = true
	k.mu.Unlock()
	if first {
		_ = k.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusRunning})
	}
	return k.request("session/prompt", map[string]any{
		"sessionId": k.sessionID,
		"prompt":    []any{map[string]any{"type": "text", "text": text}},
	})
}

// SendApproval resolves a pending session/request_permission.
func (k *Kimi) SendApproval(requestID, decision string) error {
	k.mu.Lock()
	p, ok := k.pending[requestID]
	if ok {
		delete(k.pending, requestID)
	}
	k.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending approval %s", requestID)
	}
	kind := "allow"
	if decision == "reject" {
		kind = "reject"
	}
	optionID := ""
	for id, name := range p.options {
		// Prefer the protocol-level option kind; fall back to the display
		// name for servers that don't send kinds.
		ok := strings.HasPrefix(p.kinds[id], kind) ||
			(p.kinds[id] == "" && decision == "reject" && strings.Contains(name, "Reject")) ||
			(p.kinds[id] == "" && decision == "approve" && !strings.Contains(name, "Reject"))
		if ok {
			optionID = id
			break
		}
	}
	if optionID == "" && decision == "approve" {
		// Fall back to the first option id. Never do this for reject: a
		// timeout default-deny must not silently turn into an allow.
		for id := range p.options {
			optionID = id
			break
		}
	}
	if optionID == "" {
		return fmt.Errorf("approval %s has no selectable option", requestID)
	}
	return k.writeJSON(map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"result": map[string]any{
			"outcome": map[string]any{"outcome": "selected", "optionId": optionID},
		},
	})
}

func (k *Kimi) waitReady() error {
	select {
	case <-k.readyCh:
		k.mu.Lock()
		err := k.initError
		sid := k.sessionID
		k.mu.Unlock()
		if err != nil {
			return err
		}
		if sid == "" {
			return fmt.Errorf("kimi session not initialized")
		}
		return nil
	case <-k.stopCh:
		return fmt.Errorf("session stopped")
	case <-time.After(30 * time.Second):
		return fmt.Errorf("kimi ACP initialization timed out")
	}
}

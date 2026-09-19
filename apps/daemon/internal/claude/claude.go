// Package claude implements the Claude Code adapter (L1 stream-json + L2
// Notification hooks). Field shapes follow the official stream-json protocol;
// the adapter is isolated so format drift only touches this file.
package claude

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

type pendingTool struct {
	name  string
	input map[string]any
}

// ptyTerm is one console attached to the interactive TUI. The type is
// platform-neutral; the PTY-backed methods live in tui_unix.go.
type ptyTerm struct {
	c  *Claude
	ch chan []byte
}

// approvalTimeout bounds how long a permission prompt waits for a viewer
// decision before defaulting to deny, mirroring the attach hook path.
// A var so tests can shrink it.
var approvalTimeout = 10 * time.Minute

// Claude is a wrapped Claude Code session speaking stream-json on stdio.
type Claude struct {
	id           string
	name         string
	cwd          string
	prompt       string
	binary       string
	dataDir      string
	hookBase     string
	hookToken    string
	settingsPath string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	pty    *os.File
	events chan protocol.Event
	stopCh chan struct{}
	doneCh chan struct{}

	mu               sync.Mutex
	interactive      bool
	ctx              context.Context
	launched         bool
	exited           bool
	agentActive      bool // true between the first assistant output and the turn result
	pendingTools     map[string]pendingTool
	pendingApprovals map[string]chan string
	ptySubs          map[*ptyTerm]struct{}
}

// New creates a Claude Code session adapter.
func New(req adapter.CreateRequest) *Claude {
	binary := req.Binary
	if binary == "" {
		binary = "claude"
	}
	return &Claude{
		id:               req.ID,
		name:             req.Name,
		cwd:              req.Cwd,
		prompt:           req.Prompt,
		binary:           binary,
		dataDir:          req.DataDir,
		hookBase:         req.HookBase,
		hookToken:        req.HookToken,
		events:           make(chan protocol.Event, 256),
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
		pendingTools:     make(map[string]pendingTool),
		pendingApprovals: make(map[string]chan string),
		interactive:      true,
		ptySubs:          make(map[*ptyTerm]struct{}),
	}
}

func (c *Claude) ID() string                    { return c.id }
func (c *Claude) Events() <-chan protocol.Event { return c.events }
func (c *Claude) Meta() protocol.SessionStartPayload {
	return protocol.SessionStartPayload{Name: c.name, CLI: "claude", Cwd: c.cwd}
}

// Start records the context and, when an initial prompt is present, spawns
// claude. Without an initial prompt, the process is started lazily on the
// first SendPrompt (claude -p exits immediately if started with no input).
func (c *Claude) Start(ctx context.Context) error {
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
	if c.interactive {
		// Interactive TUI must spawn immediately even without an initial
		// prompt (unlike headless `claude -p`, which exits on empty input).
		return c.ensureStarted()
	}
	if c.prompt == "" {
		return nil
	}
	if err := c.ensureStarted(); err != nil {
		return err
	}
	return c.SendPrompt(c.prompt)
}

// ensureStarted spawns claude exactly once.
func (c *Claude) ensureStarted() error {
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

func (c *Claude) spawn(ctx context.Context) error {
	if c.interactive {
		if err := c.spawnInteractive(ctx); err == nil {
			return nil
		} else {
			log.Printf("claude[%s] interactive spawn failed (%v); falling back to headless", c.id, err)
		}
	}
	if err := c.writeSettings(); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	args := []string{
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--settings", c.settingsPath,
		"--permission-mode", "default",
		"--include-partial-messages",
	}
	cmd := exec.CommandContext(ctx, c.binary, args...)
	if c.cwd != "" {
		cmd.Dir = c.cwd
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
		return fmt.Errorf("start %s: %w", c.binary, err)
	}
	c.cmd = cmd
	c.stdin = stdin
	go c.readLoop(stdout)
	go c.copyStderr(stderr)
	// Host-mode control protocol: register hooks so the daemon can answer
	// UserPromptSubmit callbacks. Without this, prompts still work, but hook
	// notifications would be missing and future approval events would not
	// reach the adapter.
	go func() {
		if err := c.initControl(); err != nil {
			log.Printf("claude[%s] control initialize: %v", c.id, err)
		}
	}()
	go func() {
		<-c.stopCh
		if c.cmd != nil && c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
	}()
	return nil
}

// Stop terminates the wrapped process.
func (c *Claude) Stop() error {
	c.mu.Lock()
	launched := c.launched
	c.mu.Unlock()
	if !launched {
		select {
		case <-c.stopCh:
		default:
			close(c.stopCh)
		}
		return nil
	}
	select {
	case <-c.stopCh:
		return nil
	default:
		close(c.stopCh)
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	if c.pty != nil {
		_ = c.pty.Close()
	}
	<-c.doneCh
	return nil
}

// SendApproval resolves a pending control_request with allow/deny.
func (c *Claude) SendApproval(requestID, decision string) error {
	c.mu.Lock()
	ch, ok := c.pendingApprovals[requestID]
	if ok {
		delete(c.pendingApprovals, requestID)
	}
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending approval %s", requestID)
	}
	mapped := "deny"
	if decision == "approve" {
		mapped = "allow"
	}
	ch <- mapped
	return nil
}

// Alive reports whether the wrapped process exists and has not exited.
// Sessions that were never launched (lazy start) report false.
func (c *Claude) Alive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.launched {
		return false
	}
	return !c.exited
}

// SendPrompt writes a user message into the stream-json stdin.
func (c *Claude) SendPrompt(text string) error {
	if err := c.ensureStarted(); err != nil {
		return err
	}
	if c.interactive && c.pty != nil {
		if _, err := fmt.Fprintf(c.pty, "%s\r", text); err != nil {
			return err
		}
		return nil
	}
	if c.promptEcho() {
		_ = c.emit(protocol.EventUserMessage, protocol.AgentMessagePayload{Text: text})
	}
	// Flip to running immediately so the client shows the activity indicator
	// from the moment the prompt is sent, before the first assistant chunk.
	c.mu.Lock()
	first := !c.agentActive
	c.agentActive = true
	c.mu.Unlock()
	if first {
		_ = c.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusRunning})
	}
	msg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": []any{map[string]any{"type": "text", "text": text}},
		},
	}
	return c.writeLine(msg)
}

// promptEcho reports whether SendPrompt should emit a user_message event for
// the text it writes. The interactive TUI streams the prompt back through the
// UserPromptSubmit hook (handleHookUserPromptSubmit), so an extra echo would
// render every client-sent message twice; headless stream-json mode never sees
// the text again and needs the local echo.
func (c *Claude) promptEcho() bool {
	return !(c.interactive && c.pty != nil)
}

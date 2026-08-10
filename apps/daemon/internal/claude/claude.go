// Package claude implements the Claude Code adapter (L1 stream-json + L2
// Notification hooks). Field shapes follow the official stream-json protocol;
// the adapter is isolated so format drift only touches this file.
package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// initControl sends the stream-json control-protocol initialize frame.
// Claude 2.1.x requires camelCase matchers/hookCallbackIds arrays.
func (c *Claude) initControl() error {
	msg := map[string]any{
		"type": "control_request",
		"request": map[string]any{
			"subtype":    "initialize",
			"request_id": "riffpad_init_1",
			"hooks": map[string]any{
				"UserPromptSubmit": []any{
					map[string]any{
						"matchers":        []any{""},
						"hookCallbackIds": []any{"hook_user_prompt"},
					},
				},
			},
			"sdk_mcp_servers": []any{},
		},
	}
	return c.writeLine(msg)
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

func (c *Claude) writeLine(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stdin == nil {
		return fmt.Errorf("stdin not ready")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(c.stdin, string(data)); err != nil {
		return err
	}
	return nil
}

func (c *Claude) writeSettings() error {
	dir := filepath.Join(c.dataDir, "sessions", c.id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	c.settingsPath = filepath.Join(dir, "settings.json")
	settings := map[string]any{"hooks": map[string]any{}}
	hooks := map[string]any{}
	if c.hookBase != "" {
		for event, spec := range c.hookSpecs() {
			hookURL := c.hookBase + "/hooks/claude/" + spec.path + "?session=" + c.id
			if c.hookToken != "" {
				hookURL += "&token=" + url.QueryEscape(c.hookToken)
			}
			hooks[event] = []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{
							"type":    "http",
							"url":     hookURL,
							"timeout": spec.timeout,
						},
					},
				},
			}
		}
	}
	if len(hooks) > 0 {
		settings["hooks"] = hooks
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.settingsPath, data, 0o600)
}

type hookSpec struct {
	path    string // daemon route suffix (kebab-case)
	timeout int    // seconds
}

// hookSpecs returns the Claude hooks to register, keyed by the event name
// Claude expects in settings.json. The interactive TUI needs the full set
// (structured events + permissions flow through hooks); headless
// stream-json mode only needs Notification.
func (c *Claude) hookSpecs() map[string]hookSpec {
	if !c.interactive {
		return map[string]hookSpec{"Notification": {path: "notification", timeout: 10}}
	}
	return map[string]hookSpec{
		"SessionStart":      {path: "session-start", timeout: 10},
		"SessionEnd":        {path: "session-end", timeout: 10},
		"UserPromptSubmit":  {path: "user-prompt-submit", timeout: 30},
		"MessageDisplay":    {path: "message-display", timeout: 10},
		"PreToolUse":        {path: "pre-tool-use", timeout: 10},
		"PostToolUse":       {path: "post-tool-use", timeout: 10},
		"PermissionRequest": {path: "permission", timeout: 600},
		"Notification":      {path: "notification", timeout: 10},
	}
}

func (c *Claude) readLoop(r io.Reader) {
	defer close(c.doneCh)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		c.handleLine(line)
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Wait()
	}
	c.mu.Lock()
	c.exited = true
	c.mu.Unlock()
	_ = c.emit(protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "process_exit"})
}

func (c *Claude) copyStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf("claude[%s] stderr: %s", c.id, scanner.Text())
	}
}

func (c *Claude) handleLine(line []byte) {
	var raw struct {
		Type      string          `json:"type"`
		Subtype   string          `json:"subtype"`
		RequestID string          `json:"request_id"`
		Message   json.RawMessage `json:"message"`
		Request   json.RawMessage `json:"request"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return
	}
	switch raw.Type {
	case "system":
		c.handleSystem(raw.Subtype, line)
	case "control_response":
		// initialize ack; nothing to do.
	case "assistant":
		c.handleAssistant(raw.Message)
	case "user":
		c.handleUser(raw.Message)
	case "control_request", "sdk_control_request":
		body := raw.Message
		if len(body) == 0 {
			body = raw.Request
		}
		c.handleControlRequest(raw.RequestID, body)
	case "result":
		// A turn finished, but the session is still alive and waiting for
		// input; "done" is reserved for real process exit (see readLoop).
		status := protocol.StatusWaitingInput
		if raw.Subtype == "error" || raw.Subtype == "error_max_turns" {
			status = protocol.StatusError
		}
		_ = c.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: status})
	}
}

func (c *Claude) handleSystem(subtype string, line []byte) {
	switch subtype {
	case "api_retry":
		var m struct {
			Attempt      int    `json:"attempt"`
			MaxRetries   int    `json:"max_retries"`
			Error        string `json:"error"`
			RetryDelayMS int64  `json:"retry_delay_ms"`
		}
		if err := json.Unmarshal(line, &m); err != nil {
			return
		}
		msg := fmt.Sprintf("API 限流（%s），重试 %d/%d…", m.Error, m.Attempt, m.MaxRetries)
		_ = c.emit(protocol.EventNotify, protocol.NotifyPayload{Level: "waiting", Message: msg})
	case "error":
		_ = c.emit(protocol.EventNotify, protocol.NotifyPayload{Level: "error", Message: "Claude Code 报错，见 daemon 日志"})
	}
}

func (c *Claude) handleAssistant(msg json.RawMessage) {
	var m struct {
		Content []struct {
			Type  string         `json:"type"`
			Text  string         `json:"text"`
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"content"`
	}
	if err := json.Unmarshal(msg, &m); err != nil {
		return
	}
	for _, block := range m.Content {
		switch block.Type {
		case "text":
			if strings.TrimSpace(block.Text) != "" {
				_ = c.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: block.Text})
			}
		case "tool_use":
			if block.Name == "Bash" {
				// Bash renders as a single "$ cmd" row (started here, completed
				// in handleUser); skip the tool_call row to avoid a duplicate.
				if cmd, _ := block.Input["command"].(string); cmd != "" {
					_ = c.emit(protocol.EventCommand, protocol.CommandPayload{Command: cmd})
				}
			} else {
				_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{
					Tool: block.Name, Status: "started", Summary: summarize(block.Name, block.Input), Args: block.Input,
				})
			}
			c.mu.Lock()
			c.pendingTools[block.ID] = pendingTool{name: block.Name, input: block.Input}
			c.mu.Unlock()
		}
	}
}

func (c *Claude) handleUser(msg json.RawMessage) {
	var m struct {
		Content []struct {
			Type      string          `json:"type"`
			ToolUseID string          `json:"tool_use_id"`
			IsError   bool            `json:"is_error"`
			Content   json.RawMessage `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal(msg, &m); err != nil {
		return
	}
	for _, block := range m.Content {
		if block.Type != "tool_result" {
			continue
		}
		c.mu.Lock()
		pt, ok := c.pendingTools[block.ToolUseID]
		if ok {
			delete(c.pendingTools, block.ToolUseID)
		}
		c.mu.Unlock()
		if !ok {
			continue
		}
		output := contentText(block.Content)
		switch pt.name {
		case "Bash":
			cmd, _ := pt.input["command"].(string)
			exit := 0
			if block.IsError {
				exit = 1
			}
			_ = c.emit(protocol.EventCommand, protocol.CommandPayload{
				Command: cmd, ExitCode: &exit, Output: truncate(output, 2000),
			})
		case "Write", "Edit", "MultiEdit", "NotepadEdit":
			path, _ := pt.input["file_path"].(string)
			if path == "" {
				path, _ = pt.input["path"].(string)
			}
			_ = c.emit(protocol.EventFileChange, protocol.FileChangePayload{Path: path, Summary: "updated"})
		}
		// Completed tool_call for non-Bash tools. Bash is represented by the
		// command event above (started in handleAssistant), so emitting a
		// tool_call here would duplicate the row.
		if pt.name != "Bash" {
			_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{
				Tool: pt.name, Status: "completed",
				Summary: summarize(pt.name, pt.input), Args: pt.input,
			})
		}
	}
}

func (c *Claude) handleControlRequest(requestID string, msg json.RawMessage) {
	var cr struct {
		Type     string `json:"type"`
		Subtype  string `json:"subtype"`
		Callback string `json:"callback_id"`
		Input    struct {
			HookEventName string `json:"hook_event_name"`
		} `json:"input"`
		ToolUseID string `json:"tool_use_id"`
		ToolUse   struct {
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"tool_use"`
	}
	if err := json.Unmarshal(msg, &cr); err != nil {
		return
	}
	if cr.Subtype == "hook_callback" {
		c.replyHook(requestID, cr.Input.HookEventName)
		return
	}
	// Old SDK format used message.type == "request_permission"; the control
	// protocol may also emit subtype == "permission". Everything else (e.g.
	// mcp_message) is ignored.
	if cr.Subtype == "" && cr.Type != "request_permission" {
		return
	}
	ch := make(chan string, 1)
	c.mu.Lock()
	c.pendingApprovals[requestID] = ch
	c.mu.Unlock()
	_ = c.emit(protocol.EventApprovalReq, protocol.ApprovalRequestPayload{
		RequestID: requestID,
		Action:    cr.ToolUse.Name,
		Summary:   summarize(cr.ToolUse.Name, cr.ToolUse.Input),
		Options:   []string{"approve", "reject"},
	})
	timeout := approvalTimeout
	go func() {
		var decision string
		select {
		case decision = <-ch:
		case <-time.After(timeout):
			// No viewer answered in time: default to deny, mirroring the
			// attach hook path. Drop the pending entry so a late
			// SendApproval fails and the viewer gets an "expired" notify,
			// and settle the card on every viewer (#171).
			decision = "deny"
			c.mu.Lock()
			delete(c.pendingApprovals, requestID)
			c.mu.Unlock()
			_ = c.emit(protocol.EventApprovalResolved, protocol.ApprovalResolvedPayload{
				RequestID: requestID,
				Decision:  "reject",
			})
		case <-c.stopCh:
			decision = "deny"
		}
		resp := map[string]any{
			"type":       "control_response",
			"request_id": requestID,
			"response":   map[string]any{"type": decision, "allowAlways": false},
		}
		if err := c.writeLine(resp); err != nil {
			log.Printf("claude[%s] write control_response: %v", c.id, err)
		}
	}()
}

// replyHook answers a control-protocol hook callback. Prompt-related hooks are
// allowed immediately; the daemon already forwarded the prompt to the user.
func (c *Claude) replyHook(requestID, hookEventName string) {
	if hookEventName == "" {
		hookEventName = "UserPromptSubmit"
	}
	resp := map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": requestID,
			"response": map[string]any{
				"hookSpecificOutput": map[string]any{
					"hookEventName": hookEventName,
					"decision":      map[string]any{"behavior": "allow"},
				},
			},
		},
	}
	if err := c.writeLine(resp); err != nil {
		log.Printf("claude[%s] write hook response: %v", c.id, err)
	}
}

func (c *Claude) emit(typ string, payload any) error {
	ev, err := protocol.NewEvent(c.id, typ, payload)
	if err != nil {
		return err
	}
	select {
	case c.events <- ev:
		return nil
	case <-c.stopCh:
		return nil
	default:
		// Slow consumer: drop rather than block the CLI parser.
		return nil
	}
}

func summarize(name string, input map[string]any) string {
	switch name {
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			return cmd
		}
	case "Write", "Edit", "MultiEdit", "NotepadEdit":
		p, _ := input["file_path"].(string)
		if p == "" {
			p, _ = input["path"].(string)
		}
		return "file: " + p
	}
	return name
}

func contentText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var arr []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &arr) == nil {
		var b strings.Builder
		for _, c := range arr {
			b.WriteString(c.Text)
		}
		return b.String()
	}
	return string(raw)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…[truncated]"
}

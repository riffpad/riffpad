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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

type pendingTool struct {
	name  string
	input map[string]any
}

// Claude is a wrapped Claude Code session speaking stream-json on stdio.
type Claude struct {
	id           string
	name         string
	cwd          string
	prompt       string
	binary       string
	dataDir      string
	hookBase     string
	settingsPath string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	events chan protocol.Event
	stopCh chan struct{}
	doneCh chan struct{}

	mu               sync.Mutex
	ctx              context.Context
	launched         bool
	exited           bool
	pendingTools     map[string]pendingTool
	pendingApprovals map[string]chan string
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
		events:           make(chan protocol.Event, 256),
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
		pendingTools:     make(map[string]pendingTool),
		pendingApprovals: make(map[string]chan string),
	}
}

func (c *Claude) ID() string              { return c.id }
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
	if err := c.writeSettings(); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	args := []string{
		"-p",
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
	msg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": []any{map[string]any{"type": "text", "text": text}},
		},
	}
	return c.writeLine(msg)
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
		hooks["Notification"] = []any{
			map[string]any{
				"type":    "http",
				"url":     c.hookBase + "/hooks/claude/notification?session=" + c.id,
				"timeout": 10,
			},
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
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return
	}
	switch raw.Type {
	case "system":
		c.handleSystem(raw.Subtype, line)
	case "assistant":
		c.handleAssistant(raw.Message)
	case "user":
		c.handleUser(raw.Message)
	case "control_request":
		c.handleControlRequest(raw.RequestID, raw.Message)
	case "result":
		status := protocol.StatusDone
		if raw.Subtype == "error" || raw.Subtype == "error_max_turns" {
			status = protocol.StatusError
		}
		_ = c.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: status})
		_ = c.emit(protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: raw.Subtype})
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
			summary := summarize(block.Name, block.Input)
			_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{
				Tool: block.Name, Status: "started", Summary: summary, Args: block.Input,
			})
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
			_ = c.emit(protocol.EventCommand, protocol.CommandPayload{
				Command: cmd, Output: truncate(output, 2000),
			})
		case "Write", "Edit", "MultiEdit", "NotepadEdit":
			path, _ := pt.input["file_path"].(string)
			if path == "" {
				path, _ = pt.input["path"].(string)
			}
			_ = c.emit(protocol.EventFileChange, protocol.FileChangePayload{Path: path, Summary: "updated"})
		}
		_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{Tool: pt.name, Status: "completed"})
	}
}

func (c *Claude) handleControlRequest(requestID string, msg json.RawMessage) {
	var cr struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
		ToolUse   struct {
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"tool_use"`
	}
	if err := json.Unmarshal(msg, &cr); err != nil {
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
	go func() {
		var decision string
		select {
		case decision = <-ch:
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

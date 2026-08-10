// Package kimi implements the Kimi Code adapter over the Agent Client
// Protocol (ACP). The daemon spawns `kimi acp` (a stdio JSON-RPC server) and
// acts as an ACP client: initialize -> session/new -> session/prompt, with
// streamed session/update notifications and session/request_permission
// approvals. No tmux is required.
package kimi

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
	"strconv"
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

// kimiHookSpec describes one [[hooks]] entry in the per-session config.
type kimiHookSpec struct {
	event   string // Kimi hook event name
	route   string // daemon route suffix
	timeout int
}

func kimiHookSpecs() []kimiHookSpec {
	return []kimiHookSpec{
		{event: "SessionStart", route: "session-start", timeout: 10},
		{event: "SessionEnd", route: "session-end", timeout: 10},
		{event: "UserPromptSubmit", route: "user-prompt-submit", timeout: 30},
		{event: "PreToolUse", route: "pre-tool-use", timeout: 600},
		{event: "PostToolUse", route: "post-tool-use", timeout: 30},
		{event: "PostToolUseFailure", route: "post-tool-use-failure", timeout: 30},
		{event: "Stop", route: "stop", timeout: 30},
		{event: "Notification", route: "notification", timeout: 30},
	}
}

// writeSessionHome builds an isolated per-session kimi home:
//
//	<dataDir>/sessions/<id>/kimi-home/
//	  config.toml   = user's config + riffpad [[hooks]] (daemon-routed)
//	  credentials/  = symlink to the user's real home (auth preserved)
//	  sessions/…    = symlinked so resume and history still work
//
// kimi is launched with KIMI_CODE_HOME=<that dir>, so plain `kimi` runs never
// see riffpad hooks. The session id rides in each hook URL so the daemon
// routes events to the right session.
func (k *Kimi) writeSessionHome() error {
	dir := filepath.Join(k.dataDir, "sessions", k.id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	home := filepath.Join(dir, "kimi-home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		return err
	}
	k.configPath = filepath.Join(home, "config.toml")

	var body strings.Builder
	if realHome, userCfg, err := findUserKimiHome(); err == nil && userCfg != "" {
		body.WriteString(userCfg)
		if !strings.HasSuffix(body.String(), "\n") {
			body.WriteString("\n")
		}
		// Symlink everything else (credentials, sessions, blobs, store, …) so
		// login state and session history behave exactly like a normal kimi.
		entries, _ := os.ReadDir(realHome)
		for _, e := range entries {
			if e.Name() == "config.toml" || e.Name() == "config.json" {
				continue
			}
			src := filepath.Join(realHome, e.Name())
			dst := filepath.Join(home, e.Name())
			if _, err := os.Lstat(dst); os.IsNotExist(err) {
				_ = os.Symlink(src, dst)
			}
		}
	}
	body.WriteString("\n# riffpad hooks (per-session, generated)\n")
	for _, spec := range kimiHookSpecs() {
		hookURL := k.hookBase + "/hooks/kimi/" + spec.route + "?session=" + k.id
		if k.hookToken != "" {
			hookURL += "&token=" + url.QueryEscape(k.hookToken)
		}
		fmt.Fprintf(&body, "[[hooks]]\n")
		fmt.Fprintf(&body, "event = %q\n", spec.event)
		fmt.Fprintf(&body, "matcher = \"\"\n")
		fmt.Fprintf(&body, "command = %q\n", "curl -sS --max-time 590 -X POST '"+hookURL+"' -H 'Content-Type: application/json' --data-binary @-")
		fmt.Fprintf(&body, "timeout = %d\n\n", spec.timeout)
	}
	return os.WriteFile(k.configPath, []byte(body.String()), 0o600)
}

// findUserKimiHome returns the user's real kimi home and config.toml text.
func findUserKimiHome() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	for _, dir := range []string{
		filepath.Join(home, ".kimi-code"),
		filepath.Join(home, ".kimi"),
	} {
		raw, err := os.ReadFile(filepath.Join(dir, "config.toml"))
		if err == nil {
			return dir, string(raw), nil
		}
		if !os.IsNotExist(err) {
			return "", "", err
		}
	}
	return "", "", os.ErrNotExist
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

func (k *Kimi) request(method string, params map[string]any) error {
	k.mu.Lock()
	id := k.nextRequestID
	k.nextRequestID++
	k.mu.Unlock()
	return k.writeJSON(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
}

func (k *Kimi) writeJSON(v any) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.stdin == nil {
		return fmt.Errorf("stdin not ready")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(k.stdin, string(data)); err != nil {
		return err
	}
	return nil
}

func (k *Kimi) readLoop(r io.Reader) {
	defer close(k.doneCh)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		k.handleLine(line)
	}
	if k.cmd != nil && k.cmd.Process != nil {
		_ = k.cmd.Wait()
	}
	k.flushMessage()
	k.mu.Lock()
	k.exited = true
	k.mu.Unlock()
	_ = k.emit(protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "process_exit"})
}

func (k *Kimi) copyStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf("kimi[%s] stderr: %s", k.id, scanner.Text())
	}
}

// handleLine processes one JSON-RPC line from the ACP server.
func (k *Kimi) handleLine(line []byte) {
	var msg struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(line, &msg); err != nil {
		return
	}
	if len(msg.Error) > 0 && string(msg.Error) != "null" {
		var e struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(msg.Error, &e)
		k.mu.Lock()
		k.initError = fmt.Errorf("kimi ACP error: %s", e.Message)
		k.mu.Unlock()
		k.closeReady()
		_ = k.emit(protocol.EventNotify, protocol.NotifyPayload{Level: "error", Message: e.Message})
		return
	}
	if msg.Method != "" {
		k.handleNotification(msg.Method, msg.ID, msg.Params)
		return
	}
	k.handleResponse(msg.ID, msg.Result)
}

func (k *Kimi) closeReady() {
	select {
	case <-k.readyCh:
	default:
		close(k.readyCh)
	}
}

func (k *Kimi) handleResponse(id json.RawMessage, result json.RawMessage) {
	var rid int
	if err := json.Unmarshal(id, &rid); err != nil {
		return
	}
	var res struct {
		SessionID  string `json:"sessionId"`
		StopReason string `json:"stopReason"`
	}
	_ = json.Unmarshal(result, &res)
	switch rid {
	case 1:
		// initialize response; session/new follows immediately.
		_ = k.request("session/new", map[string]any{
			"cwd":        k.cwd,
			"mcpServers": []any{},
		})
	case 2:
		k.mu.Lock()
		k.sessionID = res.SessionID
		k.mu.Unlock()
		if res.SessionID == "" {
			k.mu.Lock()
			k.initError = fmt.Errorf("kimi session/new returned no sessionId")
			k.mu.Unlock()
		}
		k.closeReady()
	default:
		// A session/prompt response ends the current turn.
		k.flushMessage()
		k.mu.Lock()
		k.turnActive = false
		k.mu.Unlock()
		// The turn is over but the session is still alive and waiting for
		// input; "done" is reserved for real process exit.
		status := protocol.StatusWaitingInput
		if res.StopReason == "error" || res.StopReason == "max_turns" {
			status = protocol.StatusError
		}
		_ = k.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: status})
	}
}

func (k *Kimi) handleNotification(method string, id json.RawMessage, params json.RawMessage) {
	switch method {
	case "session/update":
		k.handleSessionUpdate(params)
	case "session/request_permission":
		k.handlePermissionRequest(id, params)
	default:
		log.Printf("kimi[%s] unhandled ACP notification %s", k.id, method)
	}
}

func (k *Kimi) handleSessionUpdate(params json.RawMessage) {
	var n struct {
		SessionID string `json:"sessionId"`
		Update    struct {
			SessionUpdate string          `json:"sessionUpdate"`
			Content       json.RawMessage `json:"content"`
			ToolCall      json.RawMessage `json:"toolCall"`
			Title         string          `json:"title"`
		} `json:"update"`
	}
	if err := json.Unmarshal(params, &n); err != nil {
		return
	}
	switch n.Update.SessionUpdate {
	case "user_message_chunk":
		var c struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(n.Update.Content, &c) == nil && c.Type == "text" && c.Text != "" {
			_ = k.emit(protocol.EventUserMessage, protocol.AgentMessagePayload{Text: c.Text})
		}
	case "agent_message_chunk":
		var c struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(n.Update.Content, &c) == nil && c.Type == "text" {
			k.mu.Lock()
			k.msgBuf.WriteString(c.Text)
			k.msgActive = true
			k.mu.Unlock()
		}
	case "tool_call":
		k.handleToolCall(n.Update.ToolCall, "started")
	case "tool_call_update":
		k.handleToolCallUpdate(n.Update.ToolCall)
	}
}

func (k *Kimi) handleToolCall(raw json.RawMessage, status string) {
	var tc struct {
		ToolCallID string          `json:"toolCallId"`
		Title      string          `json:"title"`
		Kind       string          `json:"kind"`
		RawInput   json.RawMessage `json:"rawInput"`
	}
	if err := json.Unmarshal(raw, &tc); err != nil {
		return
	}
	args := map[string]any{}
	_ = json.Unmarshal(tc.RawInput, &args)
	k.mu.Lock()
	if tc.ToolCallID != "" {
		k.pendingTools[tc.ToolCallID] = pendingTool{
			name:    firstNonEmpty(tc.Title, tc.Kind),
			summary: summarizeTool(tc.Title, args),
			args:    args,
		}
	}
	k.mu.Unlock()
	_ = k.emit(protocol.EventToolCall, protocol.ToolCallPayload{
		Tool:    firstNonEmpty(tc.Title, tc.Kind),
		Status:  status,
		Summary: summarizeTool(tc.Title, args),
		Args:    args,
	})
}

func (k *Kimi) handleToolCallUpdate(raw json.RawMessage) {
	var tc struct {
		ToolCallID string          `json:"toolCallId"`
		Status     string          `json:"status"`
		Title      string          `json:"title"`
		RawOutput  json.RawMessage `json:"rawOutput"`
	}
	if err := json.Unmarshal(raw, &tc); err != nil {
		return
	}
	if tc.Status == "" {
		return
	}
	status := "completed"
	if strings.Contains(tc.Status, "fail") || strings.Contains(tc.Status, "error") {
		status = "failed"
	}
	k.mu.Lock()
	pt, ok := k.pendingTools[tc.ToolCallID]
	if ok {
		delete(k.pendingTools, tc.ToolCallID)
	}
	k.mu.Unlock()
	payload := protocol.ToolCallPayload{Tool: firstNonEmpty(tc.Title, pt.name), Status: status}
	if ok {
		// Keep the same summary/args as the "started" event so the client's
		// in-place merge keys match (no duplicate spinner + completed rows).
		payload.Summary = pt.summary
		payload.Args = pt.args
	}
	_ = k.emit(protocol.EventToolCall, payload)
}

func (k *Kimi) handlePermissionRequest(id json.RawMessage, params json.RawMessage) {
	var req struct {
		SessionID string `json:"sessionId"`
		ToolCall  struct {
			ToolCallID string          `json:"toolCallId"`
			Title      string          `json:"title"`
			Kind       string          `json:"kind"`
			RawInput   json.RawMessage `json:"rawInput"`
		} `json:"toolCall"`
		Options []struct {
			OptionID string `json:"optionId"`
			Name     string `json:"name"`
			Kind     string `json:"kind"`
		} `json:"options"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return
	}
	requestID := ""
	if json.Unmarshal(id, &requestID) != nil {
		var n int
		if json.Unmarshal(id, &n) != nil {
			return
		}
		requestID = strconv.Itoa(n)
	}
	options := map[string]string{}
	kinds := map[string]string{}
	optionNames := []string{}
	for _, o := range req.Options {
		options[o.OptionID] = o.Name
		kinds[o.OptionID] = o.Kind
		optionNames = append(optionNames, o.Name)
	}
	args := map[string]any{}
	_ = json.Unmarshal(req.ToolCall.RawInput, &args)
	k.mu.Lock()
	k.pending[requestID] = pendingApproval{
		sessionID: req.SessionID,
		action:    firstNonEmpty(req.ToolCall.Title, req.ToolCall.Kind),
		summary:   summarizeTool(req.ToolCall.Title, args),
		options:   options,
		kinds:     kinds,
	}
	k.mu.Unlock()
	_ = k.emit(protocol.EventApprovalReq, protocol.ApprovalRequestPayload{
		RequestID: requestID,
		Action:    firstNonEmpty(req.ToolCall.Title, req.ToolCall.Kind),
		Summary:   summarizeTool(req.ToolCall.Title, args),
		Options:   optionNames,
		Args:      args,
	})
	// Default to reject when no viewer answers in time, mirroring the attach
	// hook path; SendApproval deletes the pending entry, so a resolution that
	// already happened wins the race and the timer turns into a no-op.
	timeout := approvalTimeout
	go func() {
		select {
		case <-time.After(timeout):
			if err := k.SendApproval(requestID, "reject"); err == nil {
				_ = k.emit(protocol.EventApprovalResolved, protocol.ApprovalResolvedPayload{
					RequestID: requestID,
					Decision:  "reject",
				})
			}
		case <-k.stopCh:
		}
	}()
}

func (k *Kimi) flushMessage() {
	k.mu.Lock()
	if !k.msgActive {
		k.mu.Unlock()
		return
	}
	text := k.msgBuf.String()
	k.msgBuf.Reset()
	k.msgActive = false
	k.mu.Unlock()
	if strings.TrimSpace(text) != "" {
		_ = k.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: text})
	}
}

func (k *Kimi) emit(typ string, payload any) error {
	ev, err := protocol.NewEvent(k.id, typ, payload)
	if err != nil {
		return err
	}
	select {
	case k.events <- ev:
		return nil
	case <-k.stopCh:
		return nil
	default:
		return nil
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func summarizeTool(title string, args map[string]any) string {
	if title != "" {
		return title
	}
	if cmd, ok := args["command"].(string); ok {
		return cmd
	}
	if p, ok := args["filePath"].(string); ok {
		return "file: " + p
	}
	if p, ok := args["file_path"].(string); ok {
		return "file: " + p
	}
	return ""
}

// Package codex implements the Codex adapter over the official
// `codex app-server` JSON-RPC protocol. The daemon spawns a remote-control
// app-server on a unix socket per session, creates a thread, and exposes
// connect info so the local CLI can attach `codex resume --remote` and keep
// the TUI in the user's terminal (no-silent hosting). Command/file/permission
// approvals arrive as server-initiated JSON-RPC requests and are answered by
// the daemon.
package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

type pendingApproval struct {
	kind      string // command | file | permissions
	action    string
	summary   string
	requested []string // permissions request: requested permission names
}

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
		"clientInfo": map[string]any{"name": "riffpad", "version": "0.1.0"},
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
	c.mu.Lock()
	if c.socketPath != "" {
		_ = os.Remove(c.socketPath + ".pid")
		_ = os.Remove(c.socketPath)
	}
	c.mu.Unlock()
	<-c.doneCh
	return nil
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

func (c *Codex) request(method string, params map[string]any) error {
	c.mu.Lock()
	id := c.nextRequestID
	c.nextRequestID++
	c.mu.Unlock()
	return c.writeJSON(map[string]any{"method": method, "id": id, "params": params})
}

// requestSync sends a JSON-RPC request and waits for its response.
func (c *Codex) requestSync(method string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	c.mu.Lock()
	id := c.nextRequestID
	c.nextRequestID++
	ch := make(chan json.RawMessage, 1)
	c.pendingRes[id] = ch
	c.mu.Unlock()
	if err := c.writeJSON(map[string]any{"method": method, "id": id, "params": params}); err != nil {
		c.mu.Lock()
		delete(c.pendingRes, id)
		c.mu.Unlock()
		return nil, err
	}
	select {
	case res := <-ch:
		var e struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(res, &e) == nil && e.Message != "" {
			return nil, fmt.Errorf("codex %s: %s", method, e.Message)
		}
		return res, nil
	case <-time.After(timeout):
		c.mu.Lock()
		delete(c.pendingRes, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("request %s timed out", method)
	case <-c.stopCh:
		return nil, fmt.Errorf("session stopped")
	}
}

func (c *Codex) writeJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if c.sendFn != nil {
		return c.sendFn(data)
	}
	if c.conn == nil {
		return fmt.Errorf("app-server not connected")
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Codex) readLoop(conn *websocket.Conn) {
	defer close(c.doneCh)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		c.handleLine(data)
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Wait()
	}
	c.flushMessages()
	c.mu.Lock()
	c.exited = true
	c.mu.Unlock()
	_ = c.emit(protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "process_exit"})
}

func (c *Codex) copyStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf("codex[%s] stderr: %s", c.id, scanner.Text())
	}
}

func (c *Codex) handleLine(line []byte) {
	var msg struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(line, &msg); err != nil {
		return
	}
	if len(msg.Error) > 0 && string(msg.Error) != "null" {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(msg.Error, &e)
		var rid int
		if json.Unmarshal(msg.ID, &rid) == nil {
			c.mu.Lock()
			ch, ok := c.pendingRes[rid]
			if ok {
				delete(c.pendingRes, rid)
			}
			c.mu.Unlock()
			if ok {
				ch <- json.RawMessage(msg.Error)
				return
			}
		}
		c.mu.Lock()
		c.initError = fmt.Errorf("codex app-server error: %s", e.Message)
		c.mu.Unlock()
		c.closeReady()
		_ = c.emit(protocol.EventNotify, protocol.NotifyPayload{Level: "error", Message: e.Message})
		return
	}
	if msg.Method != "" && len(msg.ID) > 0 && string(msg.ID) != "null" {
		c.handleServerRequest(msg.Method, msg.ID, msg.Params)
		return
	}
	if msg.Method != "" {
		c.handleNotification(msg.Method, msg.Params)
		return
	}
	c.handleResponse(msg.ID, msg.Result)
}

func (c *Codex) closeReady() {
	select {
	case <-c.ready:
	default:
		close(c.ready)
	}
}

func (c *Codex) handleResponse(id json.RawMessage, result json.RawMessage) {
	var rid int
	if json.Unmarshal(id, &rid) != nil {
		return
	}
	c.mu.Lock()
	ch, ok := c.pendingRes[rid]
	if ok {
		delete(c.pendingRes, rid)
	}
	c.mu.Unlock()
	if ok {
		ch <- result
		return
	}
	switch rid {
	case 1:
		// initialize response; acknowledge then create the thread.
		_ = c.writeJSON(map[string]any{"method": "initialized", "params": map[string]any{}})
		_ = c.request("thread/start", map[string]any{"cwd": c.cwd})
	case 2:
		var res struct {
			Thread struct {
				ID string `json:"id"`
			} `json:"thread"`
		}
		_ = json.Unmarshal(result, &res)
		c.mu.Lock()
		c.threadID = res.Thread.ID
		if c.threadID == "" {
			c.initError = fmt.Errorf("thread/start returned no thread id")
		}
		c.mu.Unlock()
		// A fresh thread has no persisted rollout yet, so `codex resume` (the
		// local TUI bootstrap) would fail with "no rollout found". Setting a
		// name persists the rollout at zero cost; only then mark ready so the
		// CLI can attach the TUI.
		go func() {
			c.mu.Lock()
			tid := c.threadID
			c.mu.Unlock()
			if tid != "" {
				name := c.name
				if name == "" {
					name = "riffpad"
				}
				if _, err := c.requestSync("thread/name/set", map[string]any{"threadId": tid, "name": name}, 5*time.Second); err != nil {
					log.Printf("codex[%s] thread/name/set: %v", c.id, err)
				}
				c.mu.Lock()
				c.knownThreads[tid] = true
				c.mu.Unlock()
			}
			c.closeReady()
			go c.followLoop()
		}()
	default:
		var res struct {
			Turn struct {
				Status string `json:"status"`
			} `json:"turn"`
		}
		if err := json.Unmarshal(result, &res); err == nil && res.Turn.Status != "" {
			c.mu.Lock()
			c.turnActive = true
			c.mu.Unlock()
			_ = c.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusRunning})
		}
	}
}

func (c *Codex) handleNotification(method string, params json.RawMessage) {
	switch method {
	case "item/started":
		c.handleItemStarted(params)
	case "item/completed":
		c.handleItemCompleted(params)
	case "item/agentMessage/delta":
		c.handleAgentDelta(params)
	case "turn/completed":
		c.handleTurnCompleted(params)
	case "thread/status/changed":
		// turn/completed is authoritative; ignore.
	default:
		// Ignore noisy notifications (usage, diffs, plans, review, etc).
	}
}

func (c *Codex) handleItemStarted(params json.RawMessage) {
	var n struct {
		Item struct {
			ID      string          `json:"id"`
			Type    string          `json:"type"`
			Command string          `json:"command"`
			Changes json.RawMessage `json:"changes"`
			Tool    string          `json:"tool"`
			Server  string          `json:"server"`
			Status  string          `json:"status"`
		} `json:"item"`
	}
	if err := json.Unmarshal(params, &n); err != nil {
		return
	}
	switch n.Item.Type {
	case "agentMessage":
		c.mu.Lock()
		c.messages[n.Item.ID] = &strings.Builder{}
		c.mu.Unlock()
	case "commandExecution":
		_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{
			Tool: "Command", Status: "started", Summary: n.Item.Command,
		})
	case "fileChange":
		paths := fileChangePaths(n.Item.Changes)
		_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{
			Tool: "FileChange", Status: "started", Summary: strings.Join(paths, ", "),
		})
	case "mcpToolCall", "dynamicToolCall":
		_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{
			Tool: firstNonEmpty(n.Item.Server, n.Item.Tool), Status: "started",
		})
	}
}

func (c *Codex) handleItemCompleted(params json.RawMessage) {
	var n struct {
		Item struct {
			ID       string          `json:"id"`
			Type     string          `json:"type"`
			Command  string          `json:"command"`
			Changes  json.RawMessage `json:"changes"`
			Content  json.RawMessage `json:"content"`
			Tool     string          `json:"tool"`
			Server   string          `json:"server"`
			Status   string          `json:"status"`
			ExitCode *int            `json:"exitCode"`
			Output   string          `json:"aggregatedOutput"`
		} `json:"item"`
	}
	if err := json.Unmarshal(params, &n); err != nil {
		return
	}
	switch n.Item.Type {
	case "agentMessage":
		c.mu.Lock()
		b, ok := c.messages[n.Item.ID]
		if ok {
			delete(c.messages, n.Item.ID)
		}
		c.mu.Unlock()
		if ok && strings.TrimSpace(b.String()) != "" {
			_ = c.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: b.String()})
		}
	case "userMessage":
		if text := userMessageText(n.Item.Content); text != "" {
			_ = c.emit(protocol.EventUserMessage, protocol.AgentMessagePayload{Text: text})
		}
	case "commandExecution":
		status := "completed"
		if n.Item.Status == "declined" || n.Item.Status == "failed" {
			status = "failed"
		}
		_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{Tool: "Command", Status: status})
		_ = c.emit(protocol.EventCommand, protocol.CommandPayload{
			Command: n.Item.Command, ExitCode: n.Item.ExitCode, Output: truncate(n.Item.Output, 2000),
		})
	case "fileChange":
		status := "completed"
		if n.Item.Status == "declined" || n.Item.Status == "failed" {
			status = "failed"
		}
		_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{Tool: "FileChange", Status: status})
		for _, p := range fileChangePaths(n.Item.Changes) {
			_ = c.emit(protocol.EventFileChange, protocol.FileChangePayload{Path: p, Summary: "updated"})
		}
	case "mcpToolCall", "dynamicToolCall":
		_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{Tool: firstNonEmpty(n.Item.Server, n.Item.Tool), Status: "completed"})
	}
}

func (c *Codex) handleAgentDelta(params json.RawMessage) {
	var n struct {
		ItemID string `json:"itemId"`
		Delta  string `json:"delta"`
	}
	if err := json.Unmarshal(params, &n); err != nil {
		return
	}
	c.mu.Lock()
	if b, ok := c.messages[n.ItemID]; ok {
		b.WriteString(n.Delta)
	}
	c.mu.Unlock()
}

// followLoop periodically checks which threads are loaded in the app-server.
// If the user switches focus inside the TUI (e.g. `/resume` to a historical
// session), a new thread appears that we did not create; we follow it so the
// daemon keeps managing exactly the session the user is looking at.
func (c *Codex) followLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.followOnce()
		}
	}
}

func (c *Codex) followOnce() {
	res, err := c.requestSync("thread/loaded/list", map[string]any{}, 5*time.Second)
	if err != nil {
		return
	}
	var list struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(res, &list); err != nil {
		return
	}
	c.mu.Lock()
	known := make(map[string]bool, len(c.knownThreads))
	for k := range c.knownThreads {
		known[k] = true
	}
	c.mu.Unlock()
	for _, tid := range list.Data {
		if tid == "" || known[tid] {
			continue
		}
		c.followThread(tid)
		return // handle one switch per tick
	}
}

// followThread resumes a newly-loaded thread (the TUI switched to it),
// subscribes, and moves the daemon's focus to it.
func (c *Codex) followThread(tid string) {
	r, err := c.requestSync("thread/resume", map[string]any{"threadId": tid}, 5*time.Second)
	if err != nil {
		log.Printf("codex[%s] follow resume %s: %v", c.id, tid, err)
		return
	}
	var thr struct {
		Thread struct {
			ID  string `json:"id"`
			Cwd string `json:"cwd,omitempty"`
		} `json:"thread"`
	}
	_ = json.Unmarshal(r, &thr)
	c.mu.Lock()
	c.threadID = tid
	c.knownThreads[tid] = true
	c.turnActive = false
	if thr.Thread.Cwd != "" {
		c.cwd = thr.Thread.Cwd
	}
	c.mu.Unlock()
	short := tid
	if len(short) > 8 {
		short = short[:8]
	}
	_ = c.emit(protocol.EventNotify, protocol.NotifyPayload{
		Level:   "info",
		Message: "会话焦点已切换到 " + short + "（TUI 内 /resume）",
	})
	log.Printf("codex[%s] followed TUI to thread %s", c.id, tid)
}

func (c *Codex) handleTurnCompleted(params json.RawMessage) {
	var n struct {
		Turn struct {
			Status string `json:"status"`
		} `json:"turn"`
	}
	_ = json.Unmarshal(params, &n)
	c.mu.Lock()
	c.turnActive = false
	c.mu.Unlock()
	c.flushMessages()
	status := protocol.StatusDone
	if n.Turn.Status == "failed" {
		status = protocol.StatusError
	}
	_ = c.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: status})
}

func (c *Codex) handleServerRequest(method string, id json.RawMessage, params json.RawMessage) {
	requestID := ""
	if json.Unmarshal(id, &requestID) != nil {
		var n int
		if json.Unmarshal(id, &n) != nil {
			return
		}
		requestID = strconv.Itoa(n)
	}
	var req struct {
		ItemID      string `json:"itemId"`
		Command     string `json:"command"`
		Reason      string `json:"reason"`
		Permissions []struct {
			Name string `json:"name"`
		} `json:"permissions"`
		Changes []struct {
			Path string `json:"path"`
		} `json:"changes"`
	}
	_ = json.Unmarshal(params, &req)
	var kind, action, summary string
	switch method {
	case "item/commandExecution/requestApproval":
		kind = "command"
		action = "Command"
		summary = firstNonEmpty(req.Command, req.Reason)
	case "item/fileChange/requestApproval":
		kind = "file"
		action = "FileChange"
		var paths []string
		for _, ch := range req.Changes {
			paths = append(paths, ch.Path)
		}
		summary = strings.Join(paths, ", ")
	case "item/permissions/requestApproval":
		kind = "permissions"
		action = "Permissions"
		var names []string
		for _, p := range req.Permissions {
			names = append(names, p.Name)
		}
		summary = strings.Join(names, ", ")
	default:
		log.Printf("codex[%s] unhandled server request %s", c.id, method)
		return
	}
	p := pendingApproval{kind: kind, action: action, summary: summary}
	if kind == "permissions" {
		for _, perm := range req.Permissions {
			p.requested = append(p.requested, perm.Name)
		}
	}
	c.mu.Lock()
	c.pending[requestID] = p
	c.mu.Unlock()
	_ = c.emit(protocol.EventApprovalReq, protocol.ApprovalRequestPayload{
		RequestID: requestID,
		Action:    action,
		Summary:   summary,
		Options:   []string{"approve", "reject"},
	})
}

func (c *Codex) flushMessages() {
	c.mu.Lock()
	msgs := make([]string, 0, len(c.messages))
	for id, b := range c.messages {
		if strings.TrimSpace(b.String()) != "" {
			msgs = append(msgs, b.String())
		}
		delete(c.messages, id)
	}
	c.mu.Unlock()
	for _, text := range msgs {
		_ = c.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: text})
	}
}

func (c *Codex) emit(typ string, payload any) error {
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
		return nil
	}
}

func fileChangePaths(raw json.RawMessage) []string {
	var changes []struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(raw, &changes); err != nil {
		return nil
	}
	var out []string
	for _, ch := range changes {
		if ch.Path != "" {
			out = append(out, ch.Path)
		}
	}
	return out
}

// userMessageText extracts the text of a userMessage item's content blocks.
func userMessageText(raw json.RawMessage) string {
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var b strings.Builder
	for _, blk := range blocks {
		if blk.Type == "text" {
			b.WriteString(blk.Text)
		}
	}
	return strings.TrimSpace(b.String())
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…[truncated]"
}

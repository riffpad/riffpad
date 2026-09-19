package codex

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

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
		c.mu.Lock()
		restore := c.restoreThread
		c.mu.Unlock()
		if restore != "" {
			_ = c.request("thread/resume", map[string]any{"threadId": restore})
		} else {
			_ = c.request("thread/start", map[string]any{"cwd": c.cwd})
		}
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
		// A command renders as a single "$ cmd" row: started here (no exit
		// code) for the spinner, completed in handleItemCompleted (with exit
		// code) for the green check. No tool_call row — it would duplicate
		// the command row with no output to expand.
		if n.Item.Command != "" {
			_ = c.emit(protocol.EventCommand, protocol.CommandPayload{Command: n.Item.Command})
		}
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
		failed := n.Item.Status == "declined" || n.Item.Status == "failed"
		// Ensure a non-nil exit code so the client resolves the row: codex
		// omits exitCode on declined/failed items, and a nil exit code would
		// leave the spinner spinning (the client treats undefined exit as run).
		exit := n.Item.ExitCode
		if exit == nil {
			code := 0
			if failed {
				code = 1
			}
			exit = &code
		}
		_ = c.emit(protocol.EventCommand, protocol.CommandPayload{
			Command: n.Item.Command, ExitCode: exit, Output: truncate(n.Item.Output, 2000),
		})
	case "fileChange":
		status := "completed"
		if n.Item.Status == "declined" || n.Item.Status == "failed" {
			status = "failed"
		}
		paths := fileChangePaths(n.Item.Changes)
		_ = c.emit(protocol.EventToolCall, protocol.ToolCallPayload{
			Tool: "FileChange", Status: status, Summary: strings.Join(paths, ", "),
		})
		for _, p := range paths {
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
	// The turn is over but the session is still alive and waiting for input;
	// "done" is reserved for real session end.
	status := protocol.StatusWaitingInput
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
	// Default to decline when no viewer answers in time, mirroring the attach
	// hook path; SendApproval deletes the pending entry, so a resolution that
	// already happened wins the race and the timer turns into a no-op.
	timeout := approvalTimeout
	go func() {
		select {
		case <-time.After(timeout):
			if err := c.SendApproval(requestID, "reject"); err == nil {
				_ = c.emit(protocol.EventApprovalResolved, protocol.ApprovalResolvedPayload{
					RequestID: requestID,
					Decision:  "reject",
				})
			}
		case <-c.stopCh:
		}
	}()
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

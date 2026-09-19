package kimi

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

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

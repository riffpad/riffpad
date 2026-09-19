package claude

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

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
	// The first output of a turn flips the session to running so clients
	// show the "agent is running" indicator. Turn end (result) flips it back
	// to waiting_input. Only emit on the transition: stream-json delivers
	// many assistant chunks per turn.
	hasOutput := false
	for _, block := range m.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			hasOutput = true
			break
		}
		if block.Type == "tool_use" {
			hasOutput = true
			break
		}
	}
	if hasOutput {
		c.mu.Lock()
		first := !c.agentActive
		c.agentActive = true
		c.mu.Unlock()
		if first {
			_ = c.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusRunning})
		}
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

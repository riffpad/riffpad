package daemon

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

// Kimi hook endpoints. The kimi adapter runs the real interactive TUI with a
// per-session config.toml whose [[hooks]] POST JSON to these routes
// (?session=<daemon id>). PreToolUse is a synchronous gate: the handler holds
// the request until a viewer decides, then returns allow/deny to the hook
// runner (JSON hookSpecificOutput.permissionDecision).

type kimiHookPayload struct {
	SessionID    string          `json:"session_id"`
	CWD          string          `json:"cwd"`
	HookEvent    string          `json:"hook_event_name"`
	ToolName     string          `json:"tool_name"`
	ToolInput    map[string]any  `json:"tool_input"`
	ToolOutput   any             `json:"tool_output"`
	Error        string          `json:"error"`
	Prompt       json.RawMessage `json:"prompt"`
	Reason       string          `json:"reason"`
	Source       string          `json:"source"`
	Notification *struct {
		Sink             string `json:"sink"`
		NotificationType string `json:"notification_type"`
		Title            string `json:"title"`
		Body             string `json:"body"`
		Severity         string `json:"severity"`
	} `json:"notification"`
}

// kimiGatedTools are the tools Kimi would itself ask about in interactive
// mode; their PreToolUse becomes a phone approval gate. Everything else is
// auto-allowed (only a started event is emitted).
func kimiGatedTools() map[string]bool {
	return map[string]bool{
		"Shell":          true,
		"Bash":           true,
		"WriteFile":      true,
		"StrReplaceFile": true,
		"BackgroundTask": true,
	}
}

// kimiPromptText extracts the plain text from a UserPromptSubmit payload:
// newer kimi-code sends a ContentPart array, older sends a string.
func kimiPromptText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(p.Text)
		}
		return b.String()
	}
	return ""
}

func kimiStringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func kimiHookSessionID(r *http.Request, p kimiHookPayload) string {
	if sid := r.URL.Query().Get("session"); sid != "" {
		return sid
	}
	return p.SessionID
}

func (s *Server) kimiSession(r *http.Request, p kimiHookPayload) (*session, bool) {
	sid := kimiHookSessionID(r, p)
	s.mu.Lock()
	sess, ok := s.sessions[sid]
	s.mu.Unlock()
	return sess, ok
}

func (s *Server) handleKimiHook(w http.ResponseWriter, r *http.Request) {
	event := strings.TrimPrefix(r.URL.Path, "/hooks/kimi/")
	if event == "" || strings.Contains(event, "/") {
		http.NotFound(w, r)
		return
	}
	var p kimiHookPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	sess, ok := s.kimiSession(r, p)
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	sid := kimiHookSessionID(r, p)

	switch event {
	case "session-start":
		sess.status = protocol.StatusRunning
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "session-end":
		sess.status = protocol.StatusDone
		if ev, err := protocol.NewEvent(sid, protocol.EventSessionEnd,
			protocol.SessionEndPayload{Reason: p.Reason}); err == nil {
			s.pumpEvent(sess, ev)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "user-prompt-submit":
		if text := kimiPromptText(p.Prompt); text != "" {
			if ev, err := protocol.NewEvent(sid, protocol.EventUserMessage,
				protocol.PromptPayload{Text: text}); err == nil {
				s.pumpEvent(sess, ev)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "stop":
		if ev, err := protocol.NewEvent(sid, protocol.EventAgentStatus,
			protocol.AgentStatusPayload{Status: protocol.StatusWaitingInput}); err == nil {
			s.pumpEvent(sess, ev)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "pre-tool-use":
		s.handleKimiPreToolUse(w, sess, sid, p)
	case "post-tool-use":
		s.handleKimiPostToolUse(w, sess, sid, p, false)
	case "post-tool-use-failure":
		s.handleKimiPostToolUse(w, sess, sid, p, true)
	case "notification":
		level := "info"
		if p.Notification != nil {
			switch p.Notification.NotificationType {
			case "idle_prompt", "agent_needs_input":
				level = "waiting"
			case "agent_completed":
				level = "completed"
			}
		}
		msg := ""
		if p.Notification != nil {
			msg = p.Notification.Body
			if msg == "" {
				msg = p.Notification.Title
			}
		}
		if msg != "" {
			if ev, err := protocol.NewEvent(sid, protocol.EventNotify,
				protocol.NotifyPayload{Level: level, Message: msg}); err == nil {
				s.pumpEvent(sess, ev)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleKimiPreToolUse(w http.ResponseWriter, sess *session, sid string, p kimiHookPayload) {
	tool := p.ToolName
	// Emit the started row first so the timeline shows activity even while
	// waiting for a phone decision.
	if ev, err := protocol.NewEvent(sid, protocol.EventToolCall, protocol.ToolCallPayload{
		Tool:    tool,
		Status:  "started",
		Summary: kimiToolSummary(p),
		Args:    p.ToolInput,
	}); err == nil {
		s.pumpEvent(sess, ev)
	}

	allow := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":      "PreToolUse",
			"permissionDecision": "allow",
		},
	}
	if !kimiGatedTools()[tool] {
		writeJSON(w, http.StatusOK, allow)
		return
	}

	reqID := "kimi-hook-" + protocol.NewID()
	ch := make(chan string, 1)
	s.mu.Lock()
	s.pendingHooks[reqID] = ch
	s.mu.Unlock()
	if ev, err := protocol.NewEvent(sid, protocol.EventApprovalReq, protocol.ApprovalRequestPayload{
		RequestID: reqID,
		Action:    tool,
		Summary:   kimiToolSummary(p),
		Options:   []string{"approve", "reject"},
		Args:      p.ToolInput,
	}); err == nil {
		s.pumpEvent(sess, ev)
	}

	decision := "deny"
	select {
	case d := <-ch:
		if d == "approve" {
			decision = "allow"
		}
	case <-time.After(approvalHookTimeout):
	}
	s.mu.Lock()
	delete(s.pendingHooks, reqID)
	s.mu.Unlock()

	if decision == "allow" {
		writeJSON(w, http.StatusOK, allow)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": "Rejected from Riffpad phone.",
		},
	})
}

func (s *Server) handleKimiPostToolUse(w http.ResponseWriter, sess *session, sid string, p kimiHookPayload, failed bool) {
	tool := p.ToolName
	if failed {
		if ev, err := protocol.NewEvent(sid, protocol.EventToolCall, protocol.ToolCallPayload{
			Tool: tool, Status: "failed", Summary: kimiToolSummary(p),
		}); err == nil {
			s.pumpEvent(sess, ev)
		}
		if p.Error != "" {
			if ev, err := protocol.NewEvent(sid, protocol.EventNotify,
				protocol.NotifyPayload{Level: "error", Message: p.Error}); err == nil {
				s.pumpEvent(sess, ev)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	switch tool {
	case "Shell", "Bash":
		cmd := kimiStringField(p.ToolInput, "command")
		exit := 0
		if ev, err := protocol.NewEvent(sid, protocol.EventCommand,
			protocol.CommandPayload{Command: cmd, ExitCode: &exit}); err == nil {
			s.pumpEvent(sess, ev)
		}
	case "WriteFile", "StrReplaceFile":
		path := kimiStringField(p.ToolInput, "path")
		if ev, err := protocol.NewEvent(sid, protocol.EventFileChange,
			protocol.FileChangePayload{Path: path, Summary: "updated"}); err == nil {
			s.pumpEvent(sess, ev)
		}
	}
	if ev, err := protocol.NewEvent(sid, protocol.EventToolCall, protocol.ToolCallPayload{
		Tool: tool, Status: "completed", Summary: kimiToolSummary(p),
	}); err == nil {
		s.pumpEvent(sess, ev)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func kimiToolSummary(p kimiHookPayload) string {
	if cmd := kimiStringField(p.ToolInput, "command"); cmd != "" {
		return cmd
	}
	if path := kimiStringField(p.ToolInput, "path"); path != "" {
		return path
	}
	return p.ToolName
}

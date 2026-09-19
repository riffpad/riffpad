package daemon

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

// hookPayload is the common shape of Claude Code hook input JSON.
type hookPayload struct {
	HookEventName string         `json:"hook_event_name"`
	SessionID     string         `json:"session_id"`
	CWD           string         `json:"cwd"`
	ToolUseID     string         `json:"tool_use_id"`
	ToolName      string         `json:"tool_name"`
	ToolInput     map[string]any `json:"tool_input"`
	ToolUse       map[string]any `json:"tool_use"`
	Input         map[string]any `json:"input"`
	Message       string         `json:"message"`
	Prompt        string         `json:"prompt"`
	Delta         string         `json:"delta"`
	Final         bool           `json:"final"`
	MessageID     string         `json:"message_id"`
	Notification  *struct {
		// Claude Code sends notification_type; older/community payloads used
		// type, so both are accepted (see notificationType).
		Type             string `json:"type"`
		NotificationType string `json:"notification_type"`
		Message          string `json:"message"`
	} `json:"notification"`
	Error  string `json:"error"`
	Reason string `json:"reason"`
	Source string `json:"source"`
}

func decodeHook(r *http.Request) (hookPayload, bool) {
	var p hookPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return p, false
	}
	return p, true
}

// hookSessionID prefers the daemon session id passed in the hook URL
// (?session=<id>) — hosted interactive Claude registers its per-session
// hooks with that param — and falls back to Claude's own session id for
// classic attach sessions.
func hookSessionID(r *http.Request, p hookPayload) string {
	if sid := r.URL.Query().Get("session"); sid != "" {
		return sid
	}
	return p.SessionID
}

func (p *hookPayload) toolName() string {
	if p.ToolName != "" {
		return p.ToolName
	}
	if name, ok := p.ToolUse["name"].(string); ok {
		return name
	}
	return ""
}

func (p *hookPayload) toolInput() map[string]any {
	if p.ToolInput != nil {
		return p.ToolInput
	}
	if in, ok := p.ToolUse["input"].(map[string]any); ok {
		return in
	}
	if p.Input != nil {
		return p.Input
	}
	return nil
}

// notificationType returns the Notification hook's type, preferring the
// official notification_type field over the legacy type field.
func (p *hookPayload) notificationType() string {
	if p.Notification == nil {
		return ""
	}
	if p.Notification.NotificationType != "" {
		return p.Notification.NotificationType
	}
	return p.Notification.Type
}

func (p *hookPayload) summary() string {
	input := p.toolInput()
	switch p.toolName() {
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			return cmd
		}
	case "Write", "Edit", "MultiEdit", "NotepadEdit":
		path, _ := input["file_path"].(string)
		if path == "" {
			path, _ = input["path"].(string)
		}
		return "file: " + path
	}
	if p.Message != "" {
		return p.Message
	}
	return p.toolName()
}

func (s *Server) handleHookSessionStart(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sid := hookSessionID(r, p)
	s.attachSession(sid, p.CWD)
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookSessionEnd(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sid := hookSessionID(r, p)
	sess := s.getSession(sid)
	if sess != nil {
		sess.setStatus(protocol.StatusDone)
		ev, err := protocol.NewEvent(sid, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: p.Reason})
		if err == nil {
			s.pumpEvent(sess, ev)
		}
		s.announceSessions()
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookPreToolUse(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sid := hookSessionID(r, p)
	sess := s.attachSession(sid, p.CWD)
	if name := p.toolName(); name == "Bash" {
		// Bash renders as a single "$ cmd" row: started here (no exit code)
		// for the spinner, completed in handleHookPostToolUse (with exit
		// code) for the green check. No tool_call row — it would duplicate
		// the command row (#216, mirrored from the codex/claude adapters).
		if cmd, _ := p.toolInput()["command"].(string); cmd != "" {
			ev, err := protocol.NewEvent(sid, protocol.EventCommand, protocol.CommandPayload{Command: cmd})
			if err == nil {
				s.pumpEvent(sess, ev)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	ev, err := protocol.NewEvent(sid, protocol.EventToolCall, protocol.ToolCallPayload{
		Tool:    p.toolName(),
		Status:  "started",
		Summary: p.summary(),
		Args:    p.toolInput(),
	})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookPostToolUse(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sid := hookSessionID(r, p)
	sess := s.attachSession(sid, p.CWD)
	name := p.toolName()
	input := p.toolInput()
	switch name {
	case "Bash":
		cmd, _ := input["command"].(string)
		if cmd == "" {
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
		// Attach hooks don't expose the shell exit status; default to 0 so the
		// row resolves to done instead of spinning forever (the client treats
		// a missing exit code as "running").
		exit := 0
		ev, err := protocol.NewEvent(sid, protocol.EventCommand, protocol.CommandPayload{Command: cmd, ExitCode: &exit})
		if err == nil {
			s.pumpEvent(sess, ev)
		}
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	case "Write", "Edit", "MultiEdit", "NotepadEdit":
		path, _ := input["file_path"].(string)
		if path == "" {
			path, _ = input["path"].(string)
		}
		ev, err := protocol.NewEvent(sid, protocol.EventFileChange, protocol.FileChangePayload{Path: path, Summary: "updated"})
		if err == nil {
			s.pumpEvent(sess, ev)
		}
	}
	// Carry the same summary/args as the "started" event so the client's
	// in-place merge keys match; otherwise it shows a duplicate row
	// (spinner + completed) for the same tool call (#210).
	ev, err := protocol.NewEvent(sid, protocol.EventToolCall, protocol.ToolCallPayload{
		Tool: name, Status: "completed", Summary: p.summary(), Args: p.toolInput(),
	})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookUserPromptSubmit(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	if p.Prompt == "" {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	sid := hookSessionID(r, p)
	sess := s.attachSession(sid, p.CWD)
	// A prompt means the agent starts a new turn: flip the session back to
	// running so the client shows the activity indicator immediately (#253).
	sess.mu.Lock()
	st := sess.status
	sess.mu.Unlock()
	if st != protocol.StatusRunning {
		if ev, err := protocol.NewEvent(sid, protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusRunning}); err == nil {
			s.pumpEvent(sess, ev)
		}
	}
	ev, err := protocol.NewEvent(sid, protocol.EventUserMessage, protocol.PromptPayload{Text: p.Prompt})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookMessageDisplay(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sid := hookSessionID(r, p)
	sess := s.attachSession(sid, p.CWD)
	// MessageDisplay means the agent started producing output: flip the
	// session back to running so clients show the activity indicator. The
	// idle_prompt notification flips it back to waiting_input (#253).
	sess.mu.Lock()
	st := sess.status
	sess.mu.Unlock()
	if st != protocol.StatusRunning {
		if ev, err := protocol.NewEvent(sid, protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusRunning}); err == nil {
			s.pumpEvent(sess, ev)
		}
	}
	// Accumulate per message_id; emit once when the final batch arrives so the
	// timeline shows whole assistant messages instead of many partial cards.
	if p.MessageID != "" && !p.Final {
		s.mu.Lock()
		s.messageBuf[p.MessageID] += p.Delta
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	text := p.Delta
	if p.MessageID != "" {
		s.mu.Lock()
		text = s.messageBuf[p.MessageID] + p.Delta
		delete(s.messageBuf, p.MessageID)
		s.mu.Unlock()
	}
	if text == "" {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	ev, err := protocol.NewEvent(sid, protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: text})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

// handleHookPostToolUseFailure receives Claude Code's PostToolUseFailure hook,
// which fires when a tool call failed (e.g. a Bash command exited non-zero).
// PostToolUse is not sent for failures, so without this handler the started
// row would keep its spinner forever. The failed status carries the same
// summary/args as the started event so the client merges in place (#260).
func (s *Server) handleHookPostToolUseFailure(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sid := hookSessionID(r, p)
	sess := s.attachSession(sid, p.CWD)
	name := p.toolName()
	input := p.toolInput()
	if name == "Bash" {
		cmd, _ := input["command"].(string)
		if cmd != "" {
			exit := 1
			ev, err := protocol.NewEvent(sid, protocol.EventCommand, protocol.CommandPayload{
				Command: cmd, ExitCode: &exit, Output: p.Error,
			})
			if err == nil {
				s.pumpEvent(sess, ev)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	ev, err := protocol.NewEvent(sid, protocol.EventToolCall, protocol.ToolCallPayload{
		Tool: name, Status: "failed", Summary: p.summary(), Args: p.toolInput(),
	})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	if p.Error != "" {
		if ev, err := protocol.NewEvent(sid, protocol.EventNotify,
			protocol.NotifyPayload{Level: "error", Message: p.Error}); err == nil {
			s.pumpEvent(sess, ev)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookNotification(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	if p.Notification != nil && p.notificationType() == "permission_prompt" {
		// Permission prompts are surfaced by the PermissionRequest hook.
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	level := "info"
	notifType := ""
	if p.Notification != nil {
		notifType = p.notificationType()
		switch notifType {
		case "idle_prompt", "agent_needs_input":
			level = "waiting"
		case "agent_completed":
			level = "completed"
		}
		p.Message = p.Notification.Message
	}
	needsTurnReset := notifType == "idle_prompt" || notifType == "agent_needs_input"
	if !needsTurnReset && p.Message == "" {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	sid := hookSessionID(r, p)
	sess := s.attachSession(sid, p.CWD)
	if needsTurnReset {
		// The agent finished a turn and is waiting for the next prompt.
		sess.mu.Lock()
		st := sess.status
		sess.mu.Unlock()
		if st != protocol.StatusWaitingInput {
			if ev, err := protocol.NewEvent(sid, protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusWaitingInput}); err == nil {
				s.pumpEvent(sess, ev)
			}
		}
	}
	if p.Message == "" {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	s.log.Printf("notification hook session=%s type=%s msg=%q", sid, notifType, p.Message)
	ev, err := protocol.NewEvent(sid, protocol.EventNotify, protocol.NotifyPayload{Level: level, Message: p.Message})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

// handleHookStop receives Claude Code's Stop hook, which fires when the
// previous turn finishes (unlike idle_prompt, which can lag behind by ~60s).
// A live session that is still marked running immediately flips back to
// waiting_input so the client's activity indicator clears (#257).
func (s *Server) handleHookStop(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sid := hookSessionID(r, p)
	sess := s.attachSession(sid, p.CWD)
	sess.mu.Lock()
	st := sess.status
	sess.mu.Unlock()
	if st != protocol.StatusWaitingInput {
		if ev, err := protocol.NewEvent(sid, protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusWaitingInput}); err == nil {
			s.pumpEvent(sess, ev)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookPermission(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sid := hookSessionID(r, p)
	sess := s.attachSession(sid, p.CWD)
	reqID := "hook-" + protocol.NewID()
	ch := make(chan string, 1)
	s.mu.Lock()
	s.pendingHooks[reqID] = ch
	s.mu.Unlock()
	s.log.Printf("permission hook request session=%s tool=%s req=%s", sid, p.toolName(), reqID)
	ev, err := protocol.NewEvent(sid, protocol.EventApprovalReq, protocol.ApprovalRequestPayload{
		RequestID: reqID,
		Action:    p.toolName(),
		Summary:   p.summary(),
		Options:   []string{"approve", "reject"},
		Args:      p.toolInput(),
	})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	decision := "deny"
	timedOut := false
	select {
	case d := <-ch:
		if d == "approve" {
			decision = "allow"
		}
	case <-time.After(approvalHookTimeout):
		timedOut = true
	}
	// Drop the pending entry (on the timeout path it would otherwise leak):
	// a late approval_response then misses pendingHooks and the viewer gets
	// an explicit "expired" notify instead of being silently swallowed.
	s.mu.Lock()
	delete(s.pendingHooks, reqID)
	s.mu.Unlock()
	if timedOut {
		// A viewer-initiated resolution is broadcast in Server.dispatch; the
		// timeout path must settle the card on every viewer itself (#171).
		s.broadcastApprovalResolved(sess, reqID, "reject", "")
	}
	s.log.Printf("permission hook resolved session=%s req=%s decision=%s", p.SessionID, reqID, decision)
	// Claude Code 2.1.220 expects the decision inside hookSpecificOutput,
	// not the legacy permissionDecision field.
	decisionObj := map[string]any{"behavior": decision}
	if decision == "allow" {
		decisionObj["updatedInput"] = p.toolInput()
	} else {
		decisionObj["message"] = "用户拒绝了该操作"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName": "PermissionRequest",
			"decision":      decisionObj,
		},
	})
}

// hookInputText extracts a displayable string from hook payload fields.
func hookInputText(p hookPayload) string {
	parts := []string{}
	if p.Message != "" {
		parts = append(parts, p.Message)
	}
	if p.Notification != nil && p.Notification.Message != "" {
		parts = append(parts, p.Notification.Message)
	}
	return strings.Join(parts, " ")
}

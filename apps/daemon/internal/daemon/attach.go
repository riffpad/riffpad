package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

// hookAdapter is a no-op adapter for attached (hook-driven) sessions. Events
// are pushed directly through the daemon pump by hook handlers; approval and
// prompt routing use the daemon's pendingHooks map instead of the adapter.
type hookAdapter struct{}

func (hookAdapter) ID() string                       { return "" }
func (hookAdapter) Start(_ context.Context) error { return nil }
func (hookAdapter) Events() <-chan protocol.Event    { return nil }
func (hookAdapter) Meta() protocol.SessionStartPayload { return protocol.SessionStartPayload{} }
func (hookAdapter) SendApproval(_, _ string) error   { return nil }
func (hookAdapter) SendPrompt(string) error          { return nil }
func (hookAdapter) Alive() bool                      { return true }
func (hookAdapter) Stop() error                      { return nil }

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
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"notification"`
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

// attachSession returns the daemon session for a Claude session, creating it
// lazily on first hook activity.
func (s *Server) attachSession(claudeSID, cwd string) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[claudeSID]; ok {
		return sess
	}
	name := claudeSID
	if cwd != "" {
		name = filepath.Base(cwd)
	}
	sess := &session{
		id:      claudeSID,
		meta:    protocol.SessionStartPayload{Name: name, CLI: "claude (attach)", Cwd: cwd},
		adapter: hookAdapter{},
		status:  protocol.StatusRunning,
		clients: map[*client]struct{}{},
	}
	s.sessions[claudeSID] = sess
	s.log.Printf("attached session %s cwd=%s", claudeSID, cwd)
	ev, err := protocol.NewEvent(claudeSID, protocol.EventSessionStart, sess.meta)
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	return sess
}

func (s *Server) handleHookSessionStart(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	s.attachSession(p.SessionID, p.CWD)
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookSessionEnd(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sess := s.getSession(p.SessionID)
	if sess != nil {
		sess.status = protocol.StatusDone
		ev, err := protocol.NewEvent(p.SessionID, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: p.Reason})
		if err == nil {
			s.pumpEvent(sess, ev)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookPreToolUse(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sess := s.attachSession(p.SessionID, p.CWD)
	ev, err := protocol.NewEvent(p.SessionID, protocol.EventToolCall, protocol.ToolCallPayload{
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
	sess := s.attachSession(p.SessionID, p.CWD)
	name := p.toolName()
	input := p.toolInput()
	switch name {
	case "Bash":
		cmd, _ := input["command"].(string)
		ev, err := protocol.NewEvent(p.SessionID, protocol.EventCommand, protocol.CommandPayload{Command: cmd})
		if err == nil {
			s.pumpEvent(sess, ev)
		}
	case "Write", "Edit", "MultiEdit", "NotepadEdit":
		path, _ := input["file_path"].(string)
		if path == "" {
			path, _ = input["path"].(string)
		}
		ev, err := protocol.NewEvent(p.SessionID, protocol.EventFileChange, protocol.FileChangePayload{Path: path, Summary: "updated"})
		if err == nil {
			s.pumpEvent(sess, ev)
		}
	}
	ev, err := protocol.NewEvent(p.SessionID, protocol.EventToolCall, protocol.ToolCallPayload{Tool: name, Status: "completed"})
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
	sess := s.attachSession(p.SessionID, p.CWD)
	ev, err := protocol.NewEvent(p.SessionID, protocol.EventUserMessage, protocol.PromptPayload{Text: p.Prompt})
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
	sess := s.attachSession(p.SessionID, p.CWD)
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
	ev, err := protocol.NewEvent(p.SessionID, protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: text})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookNotification(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	if p.Notification != nil && p.Notification.Type == "permission_prompt" {
		// Permission prompts are surfaced by the PermissionRequest hook.
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	level := "info"
	if p.Notification != nil {
		switch p.Notification.Type {
		case "idle_prompt", "agent_needs_input":
			level = "waiting"
		case "agent_completed":
			level = "completed"
		}
		p.Message = p.Notification.Message
	}
	if p.Message == "" {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	sess := s.attachSession(p.SessionID, p.CWD)
	ev, err := protocol.NewEvent(p.SessionID, protocol.EventNotify, protocol.NotifyPayload{Level: level, Message: p.Message})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookPermission(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeHook(r)
	if !ok || p.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid hook payload")
		return
	}
	sess := s.attachSession(p.SessionID, p.CWD)
	reqID := "hook-" + protocol.NewID()
	ch := make(chan string, 1)
	s.mu.Lock()
	s.pendingHooks[reqID] = ch
	s.mu.Unlock()
	ev, err := protocol.NewEvent(p.SessionID, protocol.EventApprovalReq, protocol.ApprovalRequestPayload{
		RequestID: reqID,
		Action:    p.toolName(),
		Summary:   p.summary(),
		Options:   []string{"approve", "reject"},
	})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	decision := "deny"
	select {
	case d := <-ch:
		if d == "approve" {
			decision = "allow"
		}
	case <-time.After(10 * time.Minute):
	}
	writeJSON(w, http.StatusOK, map[string]string{"permissionDecision": decision})
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

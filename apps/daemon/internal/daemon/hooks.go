package daemon

import (
	"net/http"

	"github.com/riffpad/riffpad/packages/protocol"
)

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

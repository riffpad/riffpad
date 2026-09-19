package daemon

import (
	"net/http"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

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

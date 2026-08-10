package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

// attachAdapter is the adapter for attached (hook-driven) sessions. Events
// are pushed directly through the daemon pump by hook handlers; approvals use
// the pendingHooks map; prompts are injected into the user's tmux pane.
type attachAdapter struct {
	server *Server
	cwd    string
}

func (a *attachAdapter) ID() string                    { return "" }
func (a *attachAdapter) Start(_ context.Context) error { return nil }
func (a *attachAdapter) Events() <-chan protocol.Event { return nil }
func (a *attachAdapter) Meta() protocol.SessionStartPayload {
	return protocol.SessionStartPayload{}
}

// SendApproval on an attachAdapter is only reached when the request is not in
// the server's pendingHooks map (see Server.dispatch), i.e. the hook already
// timed out or was resolved — so it always reports the approval as expired.
func (a *attachAdapter) SendApproval(_, _ string) error {
	return fmt.Errorf("approval expired or already handled")
}
func (a *attachAdapter) SendPrompt(text string) error {
	return a.server.injectPrompt(a.cwd, text)
}
func (a *attachAdapter) Alive() bool { return true }
func (a *attachAdapter) Stop() error { return nil }

// approvalHookTimeout bounds how long a PermissionRequest hook waits for a
// viewer decision before defaulting to deny. A var so tests can shrink it.
var approvalHookTimeout = 10 * time.Minute

// attachIdleTimeout bounds how long an attach session may go without any hook
// event before the sweeper marks it ended (#170). Hook activity is the only
// liveness signal for a claude process running in the user's own tmux; after
// `kill -9` the SessionEnd hook never fires, so without this the session
// would stay "running" forever. 30min is long enough that normal thinking
// pauses never trip it; if it does trip while claude is merely idle, the next
// hook revives the session in attachSession. A var so tests can shrink it.
var attachIdleTimeout = 30 * time.Minute

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

// attachSession returns the daemon session for a Claude session, creating it
// lazily on first hook activity.
func (s *Server) attachSession(claudeSID, cwd string) *session {
	s.mu.Lock()
	sess, ok := s.sessions[claudeSID]
	s.mu.Unlock()
	if ok {
		sess.mu.Lock()
		_, restored := sess.adapter.(*restoredAdapter)
		ended := sess.ended
		if restored {
			// The agent outlived a daemon restart (#170): the real claude
			// process is still running in the user's terminal and its hooks
			// keep firing. Swap the read-only restoredAdapter for a live
			// attachAdapter in place so prompts/approvals work again.
			sess.adapter = &attachAdapter{server: s, cwd: cwd}
		}
		if restored || ended {
			// Any hook proves the agent is alive: revive sessions previously
			// marked ended (e.g. by the sweeper's idle timeout while the user
			// simply left claude open).
			sess.status = protocol.StatusRunning
			sess.ended = false
			sess.lastSeen = time.Now()
		}
		sess.mu.Unlock()
		if restored {
			s.log.Printf("reattached session %s: restored adapter swapped for live attach adapter", claudeSID)
		} else if ended {
			s.log.Printf("session %s revived by hook activity", claudeSID)
		}
		if restored || ended {
			s.persistSession(sess)
			s.announceSessions()
		}
		return sess
	}
	name := claudeSID
	if cwd != "" {
		name = filepath.Base(cwd)
	}
	sess = &session{
		id:       claudeSID,
		meta:     protocol.SessionStartPayload{Name: name, CLI: "claude (attach)", Cwd: cwd},
		adapter:  &attachAdapter{server: s, cwd: cwd},
		status:   protocol.StatusRunning,
		lastSeen: time.Now(),
		clients:  map[*client]struct{}{},
	}
	s.mu.Lock()
	if existing, ok := s.sessions[claudeSID]; ok {
		// Lost a creation race with a concurrent hook: fall back to the
		// existing session (revive path above handles restored/ended).
		s.mu.Unlock()
		return existing
	}
	s.sessions[claudeSID] = sess
	s.mu.Unlock()
	s.log.Printf("attached session %s cwd=%s", claudeSID, cwd)
	ev, err := protocol.NewEvent(claudeSID, protocol.EventSessionStart, sess.meta)
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	s.announceSessions()
	return sess
}

// injectPrompt types text into the tmux pane running claude in cwd and presses
// Enter. This is how web/remote instructions reach an interactive Claude Code
// session that the user opened themselves.
func (s *Server) injectPrompt(cwd, text string) error {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}\t#{pane_current_command}\t#{pane_current_path}").Output()
	if err != nil {
		return fmt.Errorf("tmux 不可用：%w（请把 claude 放进 tmux 再试）", err)
	}
	paneID := findClaudePane(string(out), cwd)
	if paneID == "" {
		return fmt.Errorf("未找到运行在 %s 的 claude tmux 面板（请把 claude 放进 tmux 再试）", cwd)
	}
	if err := exec.Command("tmux", "send-keys", "-t", paneID, "-l", text).Run(); err != nil {
		return fmt.Errorf("tmux send-keys: %w", err)
	}
	if err := exec.Command("tmux", "send-keys", "-t", paneID, "Enter").Run(); err != nil {
		return fmt.Errorf("tmux send-keys Enter: %w", err)
	}
	s.log.Printf("prompt injected into tmux pane=%s cwd=%s", paneID, cwd)
	return nil
}

// findClaudePane picks a pane whose current command looks like claude and
// whose path matches cwd.
func findClaudePane(tmuxOutput, cwd string) string {
	for _, line := range strings.Split(tmuxOutput, "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		id, cmd, path := parts[0], parts[1], parts[2]
		if strings.Contains(cmd, "claude") && path == cwd {
			return id
		}
	}
	return ""
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
		sess.status = protocol.StatusDone
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

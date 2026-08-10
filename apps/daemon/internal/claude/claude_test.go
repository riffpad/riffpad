package claude

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

func TestSendPromptEchoPolicy(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	if !c.interactive {
		t.Fatal("expected interactive mode by default")
	}
	// PTY not attached yet (lazy start / fallback headless): keep the echo.
	if !c.promptEcho() {
		t.Fatal("expected echo before the interactive PTY is attached")
	}
	c.pty = &os.File{}
	// Interactive TUI: the UserPromptSubmit hook already streams the prompt
	// back to viewers; echoing here would duplicate every client message.
	if c.promptEcho() {
		t.Fatal("interactive TUI must not echo; the UserPromptSubmit hook provides the message")
	}
	c.interactive = false
	if !c.promptEcho() {
		t.Fatal("headless stream-json mode needs the local echo")
	}
}

func TestHandleLineAssistantAndUser(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1", Name: "demo"})
	c.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`))
	c.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}]}}`))
	c.handleLine([]byte(`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"src"}]}}`))

	// Event order for a Bash call:
	//   agent_status(running) · agent_message · command(started, no exit code)
	//   · command(completed, exit code)
	// Bash renders as a single row: the started command (no exit code) makes
	// the client show a spinner; the completed command carries the exit code
	// and transitions it to green. No tool_call row for Bash (it duplicates).
	var got []protocol.Event
	for i := 0; i < 4; i++ {
		select {
		case ev := <-c.Events():
			got = append(got, ev)
		default:
			t.Fatalf("expected %d events, got %d", 4, len(got))
		}
	}
	if got[0].Type != protocol.EventAgentStatus {
		t.Fatalf("expected running agent_status, got %s", got[0].Type)
	}
	var runStatus protocol.AgentStatusPayload
	if err := got[0].DecodePayload(&runStatus); err != nil {
		t.Fatal(err)
	}
	if runStatus.Status != protocol.StatusRunning {
		t.Fatalf("expected running, got %q", runStatus.Status)
	}
	if got[1].Type != protocol.EventAgentMessage {
		t.Fatalf("expected agent_message, got %s", got[1].Type)
	}
	if got[2].Type != protocol.EventCommand {
		t.Fatalf("expected started command, got %s", got[2].Type)
	}
	var cmdStart protocol.CommandPayload
	if err := got[2].DecodePayload(&cmdStart); err != nil {
		t.Fatal(err)
	}
	if cmdStart.Command != "ls" || cmdStart.ExitCode != nil {
		t.Fatalf("unexpected started command (want no exit code): %+v", cmdStart)
	}
	if got[3].Type != protocol.EventCommand {
		t.Fatalf("expected completed command, got %s", got[3].Type)
	}
	var cmdDone protocol.CommandPayload
	if err := got[3].DecodePayload(&cmdDone); err != nil {
		t.Fatal(err)
	}
	if cmdDone.Command != "ls" || cmdDone.ExitCode == nil || *cmdDone.ExitCode != 0 {
		t.Fatalf("unexpected completed command (want exit code 0): %+v", cmdDone)
	}
}

func TestAssistantEmitsRunningOncePerTurn(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})

	// First assistant chunk: running, then the message.
	c.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`))
	ev := <-c.Events()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected running agent_status, got %s", ev.Type)
	}
	var st protocol.AgentStatusPayload
	if err := ev.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusRunning {
		t.Fatalf("expected running, got %q", st.Status)
	}
	ev = <-c.Events()
	if ev.Type != protocol.EventAgentMessage {
		t.Fatalf("expected agent_message, got %s", ev.Type)
	}

	// Second chunk of the same turn: message only, no second running event.
	c.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":" world"}]}}`))
	ev = <-c.Events()
	if ev.Type != protocol.EventAgentMessage {
		t.Fatalf("expected agent_message, got %s", ev.Type)
	}
	select {
	case extra := <-c.Events():
		t.Fatalf("unexpected extra event: %s", extra.Type)
	default:
	}

	// Turn result resets the flag for the next turn.
	c.handleLine([]byte(`{"type":"result","subtype":"success"}`))
	ev = <-c.Events()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected agent_status, got %s", ev.Type)
	}
	if err := ev.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusWaitingInput {
		t.Fatalf("expected waiting_input, got %q", st.Status)
	}
}

func TestTurnResultEmitsWaitingInput(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"type":"result","subtype":"success"}`))
	ev := <-c.Events()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected agent_status, got %s", ev.Type)
	}
	var p protocol.AgentStatusPayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.Status != protocol.StatusWaitingInput {
		t.Fatalf("expected waiting_input after a turn, got %q", p.Status)
	}
}

func TestHandleControlRequest(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"type":"control_request","request_id":"r1","message":{"type":"request_permission","tool_use":{"name":"Bash","input":{"command":"rm -rf build"}}}}`))
	ev := <-c.Events()
	if ev.Type != protocol.EventApprovalReq {
		t.Fatalf("expected approval_request, got %s", ev.Type)
	}
	var p protocol.ApprovalRequestPayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.RequestID != "r1" || p.Action != "Bash" {
		t.Fatalf("unexpected approval: %+v", p)
	}

	if err := c.SendApproval("r1", "approve"); err != nil {
		t.Fatal(err)
	}
	// Rejecting twice must fail (request already resolved).
	if err := c.SendApproval("r1", "approve"); err == nil {
		t.Fatal("expected error for already-resolved approval")
	}
}

func TestApprovalTimeoutDefaultsDeny(t *testing.T) {
	old := approvalTimeout
	approvalTimeout = 50 * time.Millisecond
	defer func() { approvalTimeout = old }()

	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"type":"control_request","request_id":"r9","message":{"type":"request_permission","tool_use":{"name":"Bash","input":{"command":"rm -rf build"}}}}`))
	ev := <-c.Events()
	if ev.Type != protocol.EventApprovalReq {
		t.Fatalf("expected approval_request, got %s", ev.Type)
	}
	// No viewer answers: the wait must time out into deny and settle the card
	// on every viewer via approval_resolved (#171).
	select {
	case ev := <-c.Events():
		if ev.Type != protocol.EventApprovalResolved {
			t.Fatalf("expected approval_resolved, got %s", ev.Type)
		}
		var p protocol.ApprovalResolvedPayload
		if err := ev.DecodePayload(&p); err != nil {
			t.Fatal(err)
		}
		if p.RequestID != "r9" || p.Decision != "reject" {
			t.Fatalf("unexpected resolution: %+v", p)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("approval did not time out into a resolution")
	}
	// The pending entry is gone: a late answer fails instead of applying.
	if err := c.SendApproval("r9", "approve"); err == nil {
		t.Fatal("expected error for timed-out approval")
	}
}

func TestResultEmitsStatusOnly(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"type":"result","subtype":"success","result":"done"}`))
	select {
	case ev := <-c.Events():
		if ev.Type != protocol.EventAgentStatus {
			t.Fatalf("expected agent_status, got %s", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("expected agent_status event")
	}
	select {
	case ev := <-c.Events():
		t.Fatalf("expected no session_end after turn result in host mode, got %s", ev.Type)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSystemApiRetryBecomesNotify(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"type":"system","subtype":"api_retry","attempt":2,"max_retries":10,"error":"rate_limit"}`))
	ev := <-c.Events()
	if ev.Type != protocol.EventNotify {
		t.Fatalf("expected notify, got %s", ev.Type)
	}
	var p protocol.NotifyPayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.Level != "waiting" || !strings.Contains(p.Message, "rate_limit") {
		t.Fatalf("unexpected notify: %+v", p)
	}
}

func TestWriteSettingsHookShape(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1", DataDir: t.TempDir(), HookBase: "http://127.0.0.1:8787"})
	if err := c.writeSettings(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(c.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Hooks map[string][]struct {
			Matcher string           `json:"matcher"`
			Hooks   []map[string]any `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("settings not valid JSON: %v\n%s", err, data)
	}
	n := s.Hooks["Notification"]
	if len(n) != 1 {
		t.Fatalf("expected 1 Notification hook entry, got %d", len(n))
	}
	if len(n[0].Hooks) != 1 {
		t.Fatalf("expected hooks array with 1 entry, got %d", len(n[0].Hooks))
	}
}

func TestWriteSettingsInteractiveRegistersAllHooks(t *testing.T) {
	c := New(adapter.CreateRequest{
		ID:        "s1",
		DataDir:   t.TempDir(),
		HookBase:  "http://127.0.0.1:8787",
		HookToken: "tok",
	})
	if !c.interactive {
		t.Fatal("expected interactive mode by default")
	}
	if err := c.writeSettings(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(c.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Hooks map[string][]struct {
			Matcher string           `json:"matcher"`
			Hooks   []map[string]any `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("settings not valid JSON: %v\n%s", err, data)
	}
	want := []string{
		"SessionStart", "SessionEnd", "UserPromptSubmit", "MessageDisplay",
		"PreToolUse", "PostToolUse", "PermissionRequest", "Notification", "Stop",
	}
	if len(s.Hooks) != len(want) {
		t.Fatalf("expected %d hook events, got %d (%v)", len(want), len(s.Hooks), keysOf(s.Hooks))
	}
	for _, name := range want {
		entries := s.Hooks[name]
		if len(entries) != 1 || len(entries[0].Hooks) != 1 {
			t.Fatalf("hook %s: expected 1 matcher with 1 hook, got %+v", name, entries)
		}
		u, _ := entries[0].Hooks[0]["url"].(string)
		if !strings.Contains(u, "?session=s1") || !strings.Contains(u, "token=tok") {
			t.Fatalf("hook %s url missing session/token: %q", name, u)
		}
	}
}

func keysOf(m map[string][]struct {
	Matcher string           `json:"matcher"`
	Hooks   []map[string]any `json:"hooks"`
}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestHookCallbackAutoAllowed(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"type":"control_request","request_id":"r2","request":{"subtype":"hook_callback","callback_id":"hook_user_prompt","input":{"hook_event_name":"UserPromptSubmit"}}}`))
	select {
	case ev := <-c.Events():
		t.Fatalf("expected no approval event for hook callback, got %s", ev.Type)
	case <-time.After(50 * time.Millisecond):
	}
}

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

func TestHandleLineAssistantAndUser(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1", Name: "demo"})
	c.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`))
	c.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}]}}`))
	c.handleLine([]byte(`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"src"}]}}`))

	var got []protocol.Event
	for i := 0; i < 4; i++ {
		select {
		case ev := <-c.Events():
			got = append(got, ev)
		default:
			t.Fatalf("expected %d events, got %d", 4, len(got))
		}
	}
	if got[0].Type != protocol.EventAgentMessage {
		t.Fatalf("expected agent_message, got %s", got[0].Type)
	}
	if got[1].Type != protocol.EventToolCall {
		t.Fatalf("expected tool_call, got %s", got[1].Type)
	}
	var tc protocol.ToolCallPayload
	if err := got[1].DecodePayload(&tc); err != nil {
		t.Fatal(err)
	}
	if tc.Tool != "Bash" || tc.Summary != "ls" {
		t.Fatalf("unexpected tool call: %+v", tc)
	}
	if got[2].Type != protocol.EventCommand {
		t.Fatalf("expected command, got %s", got[2].Type)
	}
	// The completed tool_call must carry the same summary/args as the
	// started one so the client can merge it in place.
	var done protocol.ToolCallPayload
	if err := got[3].DecodePayload(&done); err != nil {
		t.Fatal(err)
	}
	if done.Tool != "Bash" || done.Status != "completed" || done.Summary != "ls" {
		t.Fatalf("unexpected completed tool call: %+v", done)
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

func TestHookCallbackAutoAllowed(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"type":"control_request","request_id":"r2","request":{"subtype":"hook_callback","callback_id":"hook_user_prompt","input":{"hook_event_name":"UserPromptSubmit"}}}`))
	select {
	case ev := <-c.Events():
		t.Fatalf("expected no approval event for hook callback, got %s", ev.Type)
	case <-time.After(50 * time.Millisecond):
	}
}

package claude

import (
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

func TestHandleLineAssistantAndUser(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1", Name: "demo"})
	c.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`))
	c.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}]}}`))
	c.handleLine([]byte(`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"src"}]}}`))

	var got []protocol.Event
	for i := 0; i < 3; i++ {
		select {
		case ev := <-c.Events():
			got = append(got, ev)
		default:
			t.Fatalf("expected %d events, got %d", 3, len(got))
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

func TestResultEndsSession(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"type":"result","subtype":"success","result":"done"}`))
	ev := <-c.Events()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected agent_status, got %s", ev.Type)
	}
	ev2 := <-c.Events()
	if ev2.Type != protocol.EventSessionEnd {
		t.Fatalf("expected session_end, got %s", ev2.Type)
	}
}

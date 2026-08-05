package kimi

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

type fakeWriteCloser struct {
	bytes.Buffer
}

func (f *fakeWriteCloser) Close() error { return nil }

func nextEvent(t *testing.T, c *Kimi) protocol.Event {
	t.Helper()
	select {
	case ev := <-c.Events():
		return ev
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return protocol.Event{}
	}
}

func TestInitializeAndSessionNew(t *testing.T) {
	k := New(adapter.CreateRequest{ID: "s1", Cwd: "/tmp/proj"})
	k.handleLine([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"agentInfo":{"name":"Kimi Code CLI","version":"x"}}}`))
	k.handleLine([]byte(`{"jsonrpc":"2.0","id":2,"result":{"sessionId":"session_abc","configOptions":[]}}`))
	select {
	case <-k.readyCh:
	case <-time.After(time.Second):
		t.Fatal("expected ready after session/new")
	}
	if k.sessionID != "session_abc" {
		t.Fatalf("unexpected session id %q", k.sessionID)
	}
}

func TestAgentMessageChunksAggregated(t *testing.T) {
	k := New(adapter.CreateRequest{ID: "s1"})
	k.handleLine([]byte(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"Hel"}}}}`))
	k.handleLine([]byte(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"lo"}}}}`))
	k.handleLine([]byte(`{"jsonrpc":"2.0","id":3,"result":{"stopReason":"end_turn"}}`))
	ev := nextEvent(t, k)
	if ev.Type != protocol.EventAgentMessage {
		t.Fatalf("expected agent_message, got %s", ev.Type)
	}
	var p protocol.AgentMessagePayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.Text != "Hello" {
		t.Fatalf("expected aggregated text %q, got %q", "Hello", p.Text)
	}
	ev = nextEvent(t, k)
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected agent_status, got %s", ev.Type)
	}
}

func TestToolCallEvents(t *testing.T) {
	k := New(adapter.CreateRequest{ID: "s1"})
	k.handleLine([]byte(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"tool_call","toolCall":{"toolCallId":"t1","title":"Run command","kind":"shell","status":"running","rawInput":{"command":"ls"}}}}}`))
	k.handleLine([]byte(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"tool_call_update","toolCall":{"toolCallId":"t1","title":"Run command","status":"completed","rawOutput":"src"}}}}`))
	ev := nextEvent(t, k)
	if ev.Type != protocol.EventToolCall {
		t.Fatalf("expected tool_call, got %s", ev.Type)
	}
	var p protocol.ToolCallPayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.Tool != "Run command" || p.Status != "started" {
		t.Fatalf("unexpected tool call: %+v", p)
	}
	ev = nextEvent(t, k)
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.Status != "completed" {
		t.Fatalf("expected completed, got %s", p.Status)
	}
}

func TestPermissionRequestAndApproval(t *testing.T) {
	k := New(adapter.CreateRequest{ID: "s1"})
	fw := &fakeWriteCloser{}
	k.stdin = fw
	k.handleLine([]byte(`{"jsonrpc":"2.0","id":"perm_1","method":"session/request_permission","params":{"sessionId":"s1","toolCall":{"toolCallId":"t1","title":"Write file","kind":"write","rawInput":{"filePath":"/tmp/x"}},"options":[{"optionId":"allow","name":"Allow","kind":"allow_once"},{"optionId":"reject","name":"Reject","kind":"reject_once"}]}}`))
	ev := nextEvent(t, k)
	if ev.Type != protocol.EventApprovalReq {
		t.Fatalf("expected approval_request, got %s", ev.Type)
	}
	var p protocol.ApprovalRequestPayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.RequestID != "perm_1" || p.Action != "Write file" {
		t.Fatalf("unexpected approval: %+v", p)
	}
	if err := k.SendApproval("perm_1", "approve"); err != nil {
		t.Fatal(err)
	}
	out := fw.String()
	if !strings.Contains(out, `"optionId":"allow"`) || !strings.Contains(out, `"outcome":"selected"`) {
		t.Fatalf("unexpected approval response: %s", out)
	}
	if err := k.SendApproval("perm_1", "approve"); err == nil {
		t.Fatal("expected error for already-resolved approval")
	}
}

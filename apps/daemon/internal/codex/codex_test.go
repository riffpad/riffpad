package codex

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

func nextEvent(t *testing.T, c *Codex) protocol.Event {
	t.Helper()
	select {
	case ev := <-c.Events():
		return ev
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return protocol.Event{}
	}
}

func TestInitializeAndThreadStart(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1", Cwd: "/tmp/proj"})
	c.handleLine([]byte(`{"id":1,"result":{"userAgent":"riffpad/0.146.1"}}`))
	c.handleLine([]byte(`{"id":2,"result":{"thread":{"id":"thr_123"}}}`))
	select {
	case <-c.ready:
	case <-time.After(time.Second):
		t.Fatal("expected ready after thread/start")
	}
	if c.threadID != "thr_123" {
		t.Fatalf("unexpected thread id %q", c.threadID)
	}
}

func TestAgentMessageAggregation(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"method":"item/started","params":{"item":{"id":"msg1","type":"agentMessage"}}}`))
	c.handleLine([]byte(`{"method":"item/agentMessage/delta","params":{"threadId":"t","turnId":"u","itemId":"msg1","delta":"Hel"}}`))
	c.handleLine([]byte(`{"method":"item/agentMessage/delta","params":{"threadId":"t","turnId":"u","itemId":"msg1","delta":"lo"}}`))
	c.handleLine([]byte(`{"method":"item/completed","params":{"item":{"id":"msg1","type":"agentMessage","text":"Hello"}}}`))
	ev := nextEvent(t, c)
	if ev.Type != protocol.EventAgentMessage {
		t.Fatalf("expected agent_message, got %s", ev.Type)
	}
	var p protocol.AgentMessagePayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.Text != "Hello" {
		t.Fatalf("expected %q, got %q", "Hello", p.Text)
	}
}

func TestTurnCompletedStatus(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"id":3,"result":{"turn":{"id":"t1","status":"inProgress"}}}`))
	ev := nextEvent(t, c)
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected agent_status, got %s", ev.Type)
	}
	c.handleLine([]byte(`{"method":"turn/completed","params":{"turn":{"id":"t1","status":"completed"}}}`))
	ev = nextEvent(t, c)
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected agent_status, got %s", ev.Type)
	}
	if c.turnActive {
		t.Fatal("turn should be inactive after completion")
	}
}

func TestUserMessageEmitted(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	c.handleLine([]byte(`{"method":"item/completed","params":{"item":{"id":"u1","type":"userMessage","content":[{"type":"text","text":"hello from tui"}]}}}`))
	ev := nextEvent(t, c)
	if ev.Type != protocol.EventUserMessage {
		t.Fatalf("expected user_message, got %s", ev.Type)
	}
	var p protocol.AgentMessagePayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.Text != "hello from tui" {
		t.Fatalf("expected text %q, got %q", "hello from tui", p.Text)
	}
}

func TestCommandApproval(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	fw := &fakeWriteCloser{}
	c.sendFn = func(d []byte) error {
		_, err := fw.Write(d)
		return err
	}
	c.handleLine([]byte(`{"id":"req_1","method":"item/commandExecution/requestApproval","params":{"itemId":"it1","threadId":"thr1","turnId":"tu1","command":"rm -rf build","reason":"clean"}}`))
	ev := nextEvent(t, c)
	if ev.Type != protocol.EventApprovalReq {
		t.Fatalf("expected approval_request, got %s", ev.Type)
	}
	var p protocol.ApprovalRequestPayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.RequestID != "req_1" || p.Action != "Command" || p.Summary != "rm -rf build" {
		t.Fatalf("unexpected approval: %+v", p)
	}
	if err := c.SendApproval("req_1", "approve"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fw.String(), `"decision":"accept"`) {
		t.Fatalf("unexpected approval response: %s", fw.String())
	}
	if err := c.SendApproval("req_1", "approve"); err == nil {
		t.Fatal("expected error for already-resolved approval")
	}
}

func TestApprovalTimeoutDefaultsDecline(t *testing.T) {
	old := approvalTimeout
	approvalTimeout = 50 * time.Millisecond
	defer func() { approvalTimeout = old }()

	c := New(adapter.CreateRequest{ID: "s1"})
	fw := &fakeWriteCloser{}
	c.sendFn = func(d []byte) error {
		_, err := fw.Write(d)
		return err
	}
	c.handleLine([]byte(`{"id":"req_9","method":"item/commandExecution/requestApproval","params":{"itemId":"it1","threadId":"thr1","turnId":"tu1","command":"rm -rf build","reason":"clean"}}`))
	ev := nextEvent(t, c)
	if ev.Type != protocol.EventApprovalReq {
		t.Fatalf("expected approval_request, got %s", ev.Type)
	}
	// No viewer answers: the request must time out into decline and settle the
	// card on every viewer via approval_resolved (#171).
	ev = nextEvent(t, c)
	if ev.Type != protocol.EventApprovalResolved {
		t.Fatalf("expected approval_resolved, got %s", ev.Type)
	}
	var p protocol.ApprovalResolvedPayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.RequestID != "req_9" || p.Decision != "reject" {
		t.Fatalf("unexpected resolution: %+v", p)
	}
	if !strings.Contains(fw.String(), `"decision":"decline"`) {
		t.Fatalf("expected decline answered to app-server, got %s", fw.String())
	}
	if err := c.SendApproval("req_9", "approve"); err == nil {
		t.Fatal("expected error for timed-out approval")
	}
}

func TestPermissionsApproval(t *testing.T) {
	c := New(adapter.CreateRequest{ID: "s1"})
	fw := &fakeWriteCloser{}
	c.sendFn = func(d []byte) error {
		_, err := fw.Write(d)
		return err
	}
	c.handleLine([]byte(`{"id":"req_2","method":"item/permissions/requestApproval","params":{"itemId":"it2","threadId":"thr1","turnId":"tu1","permissions":[{"name":"network","reason":"download"}]}}`))
	ev := nextEvent(t, c)
	if ev.Type != protocol.EventApprovalReq {
		t.Fatalf("expected approval_request, got %s", ev.Type)
	}
	if err := c.SendApproval("req_2", "approve"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fw.String(), `"permissions":["network"]`) {
		t.Fatalf("unexpected approval response: %s", fw.String())
	}
}

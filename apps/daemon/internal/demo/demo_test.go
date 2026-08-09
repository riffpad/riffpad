package demo

import (
	"context"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

func collectEvents(t *testing.T, d *Demo, untilType string, timeout time.Duration) []protocol.Event {
	t.Helper()
	var out []protocol.Event
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-d.Events():
			if !ok {
				t.Fatal("event channel closed unexpectedly")
			}
			out = append(out, ev)
			if ev.Type == untilType {
				return out
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s; got %d events", untilType, len(out))
		}
	}
}

func approvalRequestID(t *testing.T, ev protocol.Event) string {
	t.Helper()
	var p protocol.ApprovalRequestPayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	return p.RequestID
}

func TestDemoWelcomeTimelineAndApproval(t *testing.T) {
	old := Delay
	Delay = 0
	t.Cleanup(func() { Delay = old })

	d := New(adapter.CreateRequest{ID: "demo-1", Cwd: "/tmp"})
	if err := d.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	evs := collectEvents(t, d, protocol.EventApprovalReq, 5*time.Second)
	types := make([]string, 0, len(evs))
	for _, ev := range evs {
		types = append(types, ev.Type)
	}
	// Scripted order: running → agent msg → tool started → command →
	// tool completed → file change → approval.
	want := []string{
		protocol.EventAgentStatus,
		protocol.EventAgentMessage,
		protocol.EventToolCall,
		protocol.EventCommand,
		protocol.EventToolCall,
		protocol.EventFileChange,
		protocol.EventApprovalReq,
	}
	if len(types) != len(want) {
		t.Fatalf("unexpected event count: %v", types)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("event %d: want %s, got %s (%v)", i, want[i], types[i], types)
		}
	}

	if err := d.SendApproval(approvalRequestID(t, evs[len(evs)-1]), "approve"); err != nil {
		t.Fatal(err)
	}
	evs = collectEvents(t, d, protocol.EventAgentStatus, 5*time.Second)
	last := evs[len(evs)-1]
	var st protocol.AgentStatusPayload
	if err := last.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusWaitingInput {
		t.Fatalf("expected waiting_input after approval, got %s", st.Status)
	}
}

func TestDemoReplyApprovalPath(t *testing.T) {
	old := Delay
	Delay = 0
	t.Cleanup(func() { Delay = old })

	d := New(adapter.CreateRequest{ID: "demo-2", Cwd: "/tmp"})
	if err := d.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Resolve the welcome approval so the session is waiting for input.
	evs := collectEvents(t, d, protocol.EventApprovalReq, 5*time.Second)
	if err := d.SendApproval(approvalRequestID(t, evs[len(evs)-1]), "approve"); err != nil {
		t.Fatal(err)
	}
	collectEvents(t, d, protocol.EventAgentStatus, 5*time.Second)

	if err := d.SendPrompt("trigger approval"); err != nil {
		t.Fatal(err)
	}
	evs = collectEvents(t, d, protocol.EventApprovalReq, 5*time.Second)
	var sawUser bool
	for _, ev := range evs {
		if ev.Type == protocol.EventUserMessage {
			sawUser = true
		}
	}
	if !sawUser {
		t.Fatal("expected user message echo in approval path")
	}
	if err := d.SendApproval(approvalRequestID(t, evs[len(evs)-1]), "reject"); err != nil {
		t.Fatal(err)
	}
	evs = collectEvents(t, d, protocol.EventAgentStatus, 5*time.Second)
	last := evs[len(evs)-1]
	var st protocol.AgentStatusPayload
	if err := last.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusWaitingInput {
		t.Fatalf("expected waiting_input after reject, got %s", st.Status)
	}
}

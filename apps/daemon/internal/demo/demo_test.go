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
	if types[0] != protocol.EventAgentStatus {
		t.Fatalf("timeline must start with running status, got %v", types)
	}
	// Interleaved workflow markers must appear in order: tool activity →
	// file change → command → the approval-gated removal.
	if !containsSubseq(types, []string{
		protocol.EventToolCall,
		protocol.EventFileChange,
		protocol.EventCommand,
		protocol.EventToolCall,
		protocol.EventApprovalReq,
	}) {
		t.Fatalf("timeline missing interleaved workflow markers: %v", types)
	}
	msgCount := 0
	for _, typ := range types {
		if typ == protocol.EventAgentMessage {
			msgCount++
		}
	}
	if msgCount < 3 {
		t.Fatalf("expected several interleaved agent messages, got %d (%v)", msgCount, types)
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

func containsSubseq(haystack, needle []string) bool {
	i := 0
	for _, h := range haystack {
		if i < len(needle) && h == needle[i] {
			i++
		}
	}
	return i == len(needle)
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

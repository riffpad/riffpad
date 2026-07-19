package service

import (
	"testing"

	"github.com/riffpad/riffpad/apps/api/internal/agent"
)

func TestWorkspaceManager_MessageHistory(t *testing.T) {
	m := NewWorkspaceManager()
	ws, err := m.Create("")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	if got := len(m.Messages(ws.ID)); got != 0 {
		t.Fatalf("expected empty history, got %d messages", got)
	}

	m.SetMessages(ws.ID, []agent.Message{
		{Role: agent.RoleUser, Content: "hi"},
		{Role: agent.RoleAssistant, Content: "hello"},
	})

	msgs := m.Messages(ws.ID)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "hi" || msgs[1].Content != "hello" {
		t.Fatalf("message content mismatch: %v", msgs)
	}

	// Mutating the returned slice should not affect internal storage.
	msgs[0].Content = "changed"
	stored := m.Messages(ws.ID)
	if stored[0].Content != "hi" {
		t.Fatalf("returned slice was not a copy")
	}
}

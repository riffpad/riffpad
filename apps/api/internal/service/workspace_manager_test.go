package service

import (
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/api/internal/agent"
	"github.com/riffpad/riffpad/apps/api/internal/domain"
)

// fakeStore is an in-memory WorkspaceStore for unit tests.
type fakeStore struct {
	workspaces map[string]*domain.Workspace
	messages   map[string][]agent.Message
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		workspaces: map[string]*domain.Workspace{},
		messages:   map[string][]agent.Message{},
	}
}

func (s *fakeStore) CreateWorkspace(ws *domain.Workspace) error {
	s.workspaces[ws.ID] = ws
	return nil
}

func (s *fakeStore) GetWorkspace(id string) (*domain.Workspace, bool) {
	ws, ok := s.workspaces[id]
	return ws, ok
}

func (s *fakeStore) ListWorkspaces(ownerID string) ([]domain.Workspace, error) {
	var out []domain.Workspace
	for _, ws := range s.workspaces {
		if ws.OwnerID == ownerID {
			out = append(out, *ws)
		}
	}
	return out, nil
}

func (s *fakeStore) UpdateWorkspace(id string, fields map[string]any) error {
	ws, ok := s.workspaces[id]
	if !ok {
		return nil
	}
	if v, ok := fields["name"]; ok {
		str := v.(string)
		ws.Name = &str
	}
	if v, ok := fields["description"]; ok {
		str := v.(string)
		ws.Description = &str
	}
	if v, ok := fields["is_pinned"]; ok {
		ws.IsPinned = v.(bool)
	}
	if v, ok := fields["status"]; ok {
		ws.Status = domain.WorkspaceStatus(v.(string))
	}
	return nil
}

func (s *fakeStore) TouchWorkspace(id string, at time.Time) error {
	if ws, ok := s.workspaces[id]; ok {
		ws.LastActiveAt = at
	}
	return nil
}

func (s *fakeStore) DeleteWorkspace(id string) error {
	delete(s.workspaces, id)
	delete(s.messages, id)
	return nil
}

func (s *fakeStore) AppendMessages(workspaceID string, msgs []agent.Message) error {
	s.messages[workspaceID] = append(s.messages[workspaceID], msgs...)
	return nil
}

func (s *fakeStore) ListMessages(workspaceID string) ([]agent.Message, error) {
	msgs := s.messages[workspaceID]
	out := make([]agent.Message, len(msgs))
	copy(out, msgs)
	return out, nil
}

func (s *fakeStore) MessageCount(workspaceID string) (int64, error) {
	return int64(len(s.messages[workspaceID])), nil
}

func TestWorkspaceManager_MessageHistory(t *testing.T) {
	m := NewWorkspaceManager(newFakeStore())
	ws, err := m.Create("")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	if got := len(m.Messages(ws.ID)); got != 0 {
		t.Fatalf("expected empty history, got %d messages", got)
	}

	m.SetMessages(ws.ID, []agent.Message{
		{Role: agent.RoleUser, Content: "hi", Timestamp: 1},
		{Role: agent.RoleAssistant, Content: "hello", Timestamp: 2},
	})

	msgs := m.Messages(ws.ID)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "hi" || msgs[1].Content != "hello" {
		t.Fatalf("message content mismatch: %v", msgs)
	}

	// SetMessages must append only the new tail, not re-append everything.
	m.SetMessages(ws.ID, []agent.Message{
		{Role: agent.RoleUser, Content: "hi", Timestamp: 1},
		{Role: agent.RoleAssistant, Content: "hello", Timestamp: 2},
		{Role: agent.RoleUser, Content: "again", Timestamp: 3},
	})
	msgs = m.Messages(ws.ID)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages after append, got %d", len(msgs))
	}
	if msgs[2].Content != "again" {
		t.Fatalf("expected appended tail, got %v", msgs)
	}

	// Mutating the returned slice should not affect internal storage.
	msgs[0].Content = "changed"
	stored := m.Messages(ws.ID)
	if stored[0].Content != "hi" {
		t.Fatalf("returned slice was not a copy")
	}
}

package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/riffpad/riffpad/apps/api/internal/agent"
	"github.com/riffpad/riffpad/apps/api/internal/domain"
	"github.com/riffpad/riffpad/apps/api/internal/idgen"
	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
)

// WorkspaceStore is the persistent state the manager depends on. It is
// satisfied by repository.Repositories (Postgres) and by in-memory fakes in
// tests.
type WorkspaceStore interface {
	CreateWorkspace(ws *domain.Workspace) error
	GetWorkspace(id string) (*domain.Workspace, bool)
	ListWorkspaces(ownerID string) ([]domain.Workspace, error)
	TouchWorkspace(id string, at time.Time) error
	UpdateWorkspace(id string, fields map[string]any) error
	DeleteWorkspace(id string) error
	AppendMessages(workspaceID string, msgs []agent.Message) error
	ListMessages(workspaceID string) ([]agent.Message, error)
	MessageCount(workspaceID string) (int64, error)
}

// WorkspaceManager is a facade over persistent state (Postgres, via the store)
// and transient state (active sandboxes, held in memory). Workspace metadata
// and message history survive restarts; sandboxes are recreated lazily on
// demand.
type WorkspaceManager struct {
	repo      WorkspaceStore
	sandboxes map[string]sandbox.Sandbox
	mu        sync.Mutex
}

func NewWorkspaceManager(repo WorkspaceStore) *WorkspaceManager {
	return &WorkspaceManager{
		repo:      repo,
		sandboxes: make(map[string]sandbox.Sandbox),
	}
}

func (m *WorkspaceManager) Create(ownerID string) (*domain.Workspace, error) {
	now := time.Now()
	ws := &domain.Workspace{
		ID:           idgen.ID(),
		Slug:         idgen.Slug(),
		OwnerID:      ownerID,
		Status:       domain.WorkspaceStatusWarm,
		LastActiveAt: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := m.repo.CreateWorkspace(ws); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	return ws, nil
}

func (m *WorkspaceManager) Get(id string) (*domain.Workspace, bool) {
	return m.repo.GetWorkspace(id)
}

func (m *WorkspaceManager) List(ownerID string) ([]domain.Workspace, error) {
	return m.repo.ListWorkspaces(ownerID)
}

func (m *WorkspaceManager) Update(id string, fields map[string]any) error {
	return m.repo.UpdateWorkspace(id, fields)
}

func (m *WorkspaceManager) Delete(id string) error {
	m.mu.Lock()
	if box, ok := m.sandboxes[id]; ok {
		delete(m.sandboxes, id)
		_ = box.Destroy()
	}
	m.mu.Unlock()
	return m.repo.DeleteWorkspace(id)
}

// Sandbox returns the active sandbox for a workspace, creating it lazily.
// After a restart the local sandbox root may still exist on disk and is reused.
func (m *WorkspaceManager) Sandbox(id string) (sandbox.Sandbox, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if box, ok := m.sandboxes[id]; ok {
		return box, true
	}
	if _, ok := m.repo.GetWorkspace(id); !ok {
		return nil, false
	}
	box, err := sandbox.NewLocal(id)
	if err != nil {
		return nil, false
	}
	m.sandboxes[id] = box
	return box, true
}

func (m *WorkspaceManager) Messages(id string) []agent.Message {
	msgs, err := m.repo.ListMessages(id)
	if err != nil {
		return nil
	}
	return msgs
}

// SetMessages appends the messages that are new compared to what is already
// persisted. Callers must serialize runs per workspace (the WS handler does
// this with a per-connection mutex).
func (m *WorkspaceManager) SetMessages(id string, msgs []agent.Message) {
	count, err := m.repo.MessageCount(id)
	if err != nil {
		return
	}
	if int64(len(msgs)) <= count {
		return
	}
	newMsgs := msgs[count:]
	if err := m.repo.AppendMessages(id, newMsgs); err == nil {
		_ = m.repo.TouchWorkspace(id, time.Now())
	}
}

func (m *WorkspaceManager) ListFiles(ctx context.Context, id string, dir string) ([]sandbox.FileInfo, error) {
	box, ok := m.Sandbox(id)
	if !ok {
		return nil, fmt.Errorf("sandbox not found")
	}
	return box.ListFiles(ctx, dir)
}

func (m *WorkspaceManager) ListFilesRecursive(ctx context.Context, id string, dir string) ([]sandbox.FileInfo, error) {
	box, ok := m.Sandbox(id)
	if !ok {
		return nil, fmt.Errorf("sandbox not found")
	}

	var result []sandbox.FileInfo
	var walk func(string) error
	walk = func(current string) error {
		files, err := box.ListFiles(ctx, current)
		if err != nil {
			return err
		}
		for _, f := range files {
			result = append(result, f)
			if f.IsDir {
				if err := walk(f.Path); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(dir); err != nil {
		return nil, err
	}
	return result, nil
}

func (m *WorkspaceManager) ReadFile(ctx context.Context, id string, path string) (string, error) {
	box, ok := m.Sandbox(id)
	if !ok {
		return "", fmt.Errorf("sandbox not found")
	}
	return box.ReadFile(ctx, path)
}

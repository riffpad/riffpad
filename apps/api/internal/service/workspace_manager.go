package service

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/riffpad/riffpad/apps/api/internal/agent"
	"github.com/riffpad/riffpad/apps/api/internal/domain"
	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
)

type WorkspaceManager struct {
	workspaces map[string]*domain.Workspace
	sandboxes  map[string]sandbox.Sandbox
	messages   map[string][]agent.Message
	mu         sync.RWMutex
}

func NewWorkspaceManager() *WorkspaceManager {
	return &WorkspaceManager{
		workspaces: make(map[string]*domain.Workspace),
		sandboxes:  make(map[string]sandbox.Sandbox),
		messages:   make(map[string][]agent.Message),
	}
}

func (m *WorkspaceManager) Create(ownerID string) (*domain.Workspace, error) {
	ws := &domain.Workspace{
		ID:      generateID(),
		Slug:    generateSlug(),
		OwnerID: ownerID,
		Status:  domain.WorkspaceStatusWarm,
	}
	box, err := sandbox.NewLocal(ws.ID)
	if err != nil {
		return nil, fmt.Errorf("create sandbox: %w", err)
	}

	m.workspaces[ws.ID] = ws
	m.sandboxes[ws.ID] = box
	return ws, nil
}

func (m *WorkspaceManager) Get(id string) (*domain.Workspace, bool) {
	ws, ok := m.workspaces[id]
	return ws, ok
}

func (m *WorkspaceManager) Sandbox(id string) (sandbox.Sandbox, bool) {
	box, ok := m.sandboxes[id]
	return box, ok
}

func (m *WorkspaceManager) Messages(id string) []agent.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msgs := m.messages[id]
	out := make([]agent.Message, len(msgs))
	copy(out, msgs)
	return out
}

func (m *WorkspaceManager) SetMessages(id string, msgs []agent.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]agent.Message, len(msgs))
	copy(out, msgs)
	m.messages[id] = out
}

func (m *WorkspaceManager) ListFiles(ctx context.Context, id string, dir string) ([]sandbox.FileInfo, error) {
	box, ok := m.sandboxes[id]
	if !ok {
		return nil, fmt.Errorf("sandbox not found")
	}
	return box.ListFiles(ctx, dir)
}

func (m *WorkspaceManager) ReadFile(ctx context.Context, id string, path string) (string, error) {
	box, ok := m.sandboxes[id]
	if !ok {
		return "", fmt.Errorf("sandbox not found")
	}
	return box.ReadFile(ctx, path)
}

func generateID() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

func generateSlug() string {
	adjectives := []string{"amber", "velvet", "silent", "brisk", "lunar", "solar", "neon", "quiet", "bold", "swift"}
	nouns := []string{"spark", "breeze", "wave", "pulse", "orbit", "flame", "drift", "echo", "glint", "nova"}
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	return fmt.Sprintf("%s-%s-%d", adj, noun, rand.Intn(100))
}

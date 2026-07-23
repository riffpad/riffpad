package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/riffpad/riffpad/apps/api/internal/agent"
	"github.com/riffpad/riffpad/apps/api/internal/domain"
	"github.com/riffpad/riffpad/apps/api/internal/idgen"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repositories struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Repositories {
	return &Repositories{DB: db}
}

// Connect opens the Postgres connection and runs dev-time AutoMigrate.
// A proper migration tool replaces AutoMigrate before production (see
// docs/multi-workspace-design.md).
func Connect(databaseURL string) (*Repositories, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := db.AutoMigrate(&domain.Workspace{}, &domain.Message{}); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	return &Repositories{DB: db}, nil
}

// --- Workspaces ---

func (r *Repositories) CreateWorkspace(ws *domain.Workspace) error {
	return r.DB.Create(ws).Error
}

func (r *Repositories) GetWorkspace(id string) (*domain.Workspace, bool) {
	var ws domain.Workspace
	if err := r.DB.First(&ws, "id = ?", id).Error; err != nil {
		return nil, false
	}
	return &ws, true
}

func (r *Repositories) ListWorkspaces(ownerID string) ([]domain.Workspace, error) {
	var list []domain.Workspace
	err := r.DB.
		Where("owner_id = ?", ownerID).
		Order("is_pinned DESC, last_active_at DESC").
		Find(&list).Error
	return list, err
}

// UpdateWorkspace applies a partial update to a workspace. Only the keys
// present in fields are written.
func (r *Repositories) UpdateWorkspace(id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.DB.Model(&domain.Workspace{}).
		Where("id = ?", id).
		Updates(fields).Error
}

func (r *Repositories) TouchWorkspace(id string, at time.Time) error {
	return r.DB.Model(&domain.Workspace{}).
		Where("id = ?", id).
		Update("last_active_at", at).Error
}

func (r *Repositories) DeleteWorkspace(id string) error {
	if err := r.DB.Where("workspace_id = ?", id).Delete(&domain.Message{}).Error; err != nil {
		return err
	}
	return r.DB.Where("id = ?", id).Delete(&domain.Workspace{}).Error
}

// --- Messages ---

// AppendMessages persists the given agent messages for a workspace, assigning
// sequential Seq values after the current maximum. Callers must serialize
// concurrent runs for the same workspace.
func (r *Repositories) AppendMessages(workspaceID string, msgs []agent.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	var maxSeq int
	if err := r.DB.Model(&domain.Message{}).
		Where("workspace_id = ?", workspaceID).
		Select("COALESCE(MAX(seq), 0)").
		Scan(&maxSeq).Error; err != nil {
		return err
	}

	rows := make([]domain.Message, 0, len(msgs))
	for i, m := range msgs {
		row := domain.Message{
			ID:          idgen.ID(),
			WorkspaceID: workspaceID,
			Seq:         maxSeq + 1 + i,
			Role:        string(m.Role),
			Content:     m.Content,
			ToolCallID:  m.ToolCallID,
			Name:        m.Name,
			CreatedAt:   time.UnixMilli(m.Timestamp),
		}
		if len(m.ToolCalls) > 0 {
			if data, err := json.Marshal(m.ToolCalls); err == nil {
				row.ToolCalls = data
			}
		}
		if len(m.Metadata) > 0 {
			if data, err := json.Marshal(m.Metadata); err == nil {
				row.Metadata = data
			}
		}
		rows = append(rows, row)
	}
	return r.DB.Create(&rows).Error
}

// ListMessages returns the full conversation history for a workspace,
// ordered for replay.
func (r *Repositories) ListMessages(workspaceID string) ([]agent.Message, error) {
	var rows []domain.Message
	if err := r.DB.
		Where("workspace_id = ?", workspaceID).
		Order("seq ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	msgs := make([]agent.Message, 0, len(rows))
	for _, row := range rows {
		m := agent.Message{
			Role:       agent.Role(row.Role),
			Content:    row.Content,
			ToolCallID: row.ToolCallID,
			Name:       row.Name,
			Timestamp:  row.CreatedAt.UnixMilli(),
		}
		if len(row.ToolCalls) > 0 {
			var calls []agent.ToolCall
			if err := json.Unmarshal(row.ToolCalls, &calls); err == nil {
				m.ToolCalls = calls
			}
		}
		if len(row.Metadata) > 0 {
			m.Metadata = map[string]any{}
			_ = json.Unmarshal(row.Metadata, &m.Metadata)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (r *Repositories) MessageCount(workspaceID string) (int64, error) {
	var count int64
	err := r.DB.Model(&domain.Message{}).
		Where("workspace_id = ?", workspaceID).
		Count(&count).Error
	return count, err
}

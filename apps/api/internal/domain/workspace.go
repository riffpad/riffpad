package domain

import "time"

type WorkspaceStatus string

const (
	WorkspaceStatusWarm        WorkspaceStatus = "WARM"
	WorkspaceStatusHibernating WorkspaceStatus = "HIBERNATING"
	WorkspaceStatusArchived    WorkspaceStatus = "ARCHIVED"
)

type Workspace struct {
	ID           string          `json:"id" gorm:"primaryKey"`
	Slug         string          `json:"slug" gorm:"uniqueIndex"`
	Name         *string         `json:"name"`
	OwnerID      string          `json:"ownerId"`
	Status       WorkspaceStatus `json:"status" gorm:"default:WARM"`
	SandboxID    *string         `json:"sandboxId"`
	BaseImage    string          `json:"baseImage" gorm:"default:riffpad-node-python"`
	LastActiveAt time.Time       `json:"lastActiveAt"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

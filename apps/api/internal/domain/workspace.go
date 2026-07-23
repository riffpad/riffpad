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
	Description  *string         `json:"description" gorm:"type:text"`
	OwnerID      string          `json:"ownerId" gorm:"index"`
	Status       WorkspaceStatus `json:"status" gorm:"default:WARM"`
	IsPinned     bool            `json:"isPinned" gorm:"default:false"`
	SandboxID    *string         `json:"sandboxId"`
	BaseImage    string          `json:"baseImage" gorm:"default:riffpad-node-python"`
	LastActiveAt time.Time       `json:"lastActiveAt"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

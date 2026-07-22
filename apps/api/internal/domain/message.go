package domain

import "time"

// Message is the persisted form of an agent conversation message.
// It is a subset of the TSD §4 Message model (Phase 1).
type Message struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspaceId" gorm:"index"`
	Seq         int       `json:"seq"`
	Role        string    `json:"role"`
	Content     string    `json:"content" gorm:"type:text"`
	ToolCalls   []byte    `json:"toolCalls,omitempty" gorm:"type:jsonb"`
	ToolCallID  string    `json:"toolCallId,omitempty"`
	Name        string    `json:"name,omitempty"`
	Metadata    []byte    `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt   time.Time `json:"createdAt"`
}

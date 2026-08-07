package adapter

import (
	"context"

	"github.com/riffpad/riffpad/packages/protocol"
)

// Session is the daemon's view of a wrapped coding-agent session.
type Session interface {
	ID() string
	Start(ctx context.Context) error
	Events() <-chan protocol.Event
	Meta() protocol.SessionStartPayload
	SendApproval(requestID, decision string) error
	SendPrompt(text string) error
	Alive() bool
	Stop() error
}

// CreateRequest describes a session to start.
type CreateRequest struct {
	ID        string
	Name      string
	CLI       string
	Binary    string
	Cwd       string
	Prompt    string
	DataDir   string
	HookBase  string
	HookToken string // local API token appended to hook URLs (see daemon.localAuth)
}

// Factory creates a session from a CreateRequest.
type Factory func(ctx context.Context, req CreateRequest) (Session, error)

package adapter

import (
	"context"
	"io"

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

// TerminalSession is implemented by adapters that expose an interactive PTY
// (e.g. Claude foreground mode). The daemon uses it to bridge the vendor TUI
// to a local CLI console over WebSocket.
type TerminalSession interface {
	AttachPTY() (Terminal, error)
}

// Terminal is one console attached to a session PTY. Read blocks until the
// vendor process writes output; Write sends input; Resize propagates the
// local window size; Close detaches this console (the process keeps running).
type Terminal interface {
	io.ReadWriteCloser
	Resize(cols, rows uint16) error
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

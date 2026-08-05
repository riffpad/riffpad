// Package protocol defines the shared event protocol and E2EE envelope
// used between the Riffpad daemon, relay, and mobile/web clients.
package protocol

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Event types (see docs/tsd.md §4.3).
const (
	EventSessionStart = "session_start"
	EventSessionEnd   = "session_end"
	EventAgentStatus  = "agent_status"
	EventAgentMessage = "agent_message"
	EventUserMessage  = "user_message"
	EventToolCall     = "tool_call"
	EventFileChange   = "file_change"
	EventCommand      = "command"
	EventApprovalReq  = "approval_request"
	EventApprovalResp = "approval_response"
	EventPrompt       = "prompt"
	EventControl      = "control"
	EventNotify       = "notify"
)

// Agent status values.
const (
	StatusRunning      = "running"
	StatusWaitingInput = "waiting_input"
	StatusDone         = "done"
	StatusError        = "error"
)

// Event is the wire-level event container. Payload holds the typed JSON
// payload for the event type.
type Event struct {
	ID        string          `json:"id"`
	SessionID string          `json:"sessionId"`
	Timestamp int64           `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type SessionStartPayload struct {
	Name    string `json:"name,omitempty"`
	CLI     string `json:"cli,omitempty"`
	Cwd     string `json:"cwd,omitempty"`
	Version string `json:"version,omitempty"`
}

type SessionEndPayload struct {
	Reason string `json:"reason,omitempty"`
}

type AgentStatusPayload struct {
	Status string `json:"status"`
}

type AgentMessagePayload struct {
	Text    string `json:"text"`
	Partial bool   `json:"partial,omitempty"`
}

type ToolCallPayload struct {
	Tool    string         `json:"tool"`
	Status  string         `json:"status"` // started | completed | failed
	Summary string         `json:"summary,omitempty"`
	Args    map[string]any `json:"args,omitempty"`
}

type FileChangePayload struct {
	Path    string `json:"path"`
	Summary string `json:"summary,omitempty"`
}

type CommandPayload struct {
	Command  string `json:"command"`
	ExitCode *int   `json:"exitCode,omitempty"`
	Output   string `json:"output,omitempty"`
}

type ApprovalRequestPayload struct {
	RequestID string         `json:"requestId"`
	Action    string         `json:"action"`
	Summary   string         `json:"summary,omitempty"`
	Options   []string       `json:"options"`
	Args      map[string]any `json:"args,omitempty"`
}

type ApprovalResponsePayload struct {
	RequestID string `json:"requestId"`
	Decision  string `json:"decision"` // approve | reject
	Condition string `json:"condition,omitempty"`
}

type PromptPayload struct {
	Text string `json:"text"`
}

type ControlPayload struct {
	Action string `json:"action"` // pause | resume | stop | ping | pong
}

type NotifyPayload struct {
	Level   string `json:"level"` // info | waiting | completed | error
	Message string `json:"message"`
}

// NewID returns a random 16-byte hex identifier.
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// NewEvent builds an Event with a typed payload marshaled into Payload.
func NewEvent(sessionID, typ string, payload any) (Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal payload: %w", err)
	}
	if typ == "" {
		return Event{}, fmt.Errorf("event type is required")
	}
	if sessionID == "" {
		return Event{}, fmt.Errorf("sessionId is required")
	}
	return Event{
		ID:        NewID(),
		SessionID: sessionID,
		Timestamp: nowMillis(),
		Type:      typ,
		Payload:   raw,
	}, nil
}

// DecodePayload unmarshals the payload into v.
func (e Event) DecodePayload(v any) error {
	if len(e.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(e.Payload, v)
}

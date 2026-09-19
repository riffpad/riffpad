package daemon

import (
	"encoding/json"
	"net/http"
	"strings"
)

// hookPayload is the common shape of Claude Code hook input JSON.
type hookPayload struct {
	HookEventName string         `json:"hook_event_name"`
	SessionID     string         `json:"session_id"`
	CWD           string         `json:"cwd"`
	ToolUseID     string         `json:"tool_use_id"`
	ToolName      string         `json:"tool_name"`
	ToolInput     map[string]any `json:"tool_input"`
	ToolUse       map[string]any `json:"tool_use"`
	Input         map[string]any `json:"input"`
	Message       string         `json:"message"`
	Prompt        string         `json:"prompt"`
	Delta         string         `json:"delta"`
	Final         bool           `json:"final"`
	MessageID     string         `json:"message_id"`
	Notification  *struct {
		// Claude Code sends notification_type; older/community payloads used
		// type, so both are accepted (see notificationType).
		Type             string `json:"type"`
		NotificationType string `json:"notification_type"`
		Message          string `json:"message"`
	} `json:"notification"`
	Error  string `json:"error"`
	Reason string `json:"reason"`
	Source string `json:"source"`
}

func decodeHook(r *http.Request) (hookPayload, bool) {
	var p hookPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return p, false
	}
	return p, true
}

// hookSessionID prefers the daemon session id passed in the hook URL
// (?session=<id>) — hosted interactive Claude registers its per-session
// hooks with that param — and falls back to Claude's own session id for
// classic attach sessions.
func hookSessionID(r *http.Request, p hookPayload) string {
	if sid := r.URL.Query().Get("session"); sid != "" {
		return sid
	}
	return p.SessionID
}

func (p *hookPayload) toolName() string {
	if p.ToolName != "" {
		return p.ToolName
	}
	if name, ok := p.ToolUse["name"].(string); ok {
		return name
	}
	return ""
}

func (p *hookPayload) toolInput() map[string]any {
	if p.ToolInput != nil {
		return p.ToolInput
	}
	if in, ok := p.ToolUse["input"].(map[string]any); ok {
		return in
	}
	if p.Input != nil {
		return p.Input
	}
	return nil
}

// notificationType returns the Notification hook's type, preferring the
// official notification_type field over the legacy type field.
func (p *hookPayload) notificationType() string {
	if p.Notification == nil {
		return ""
	}
	if p.Notification.NotificationType != "" {
		return p.Notification.NotificationType
	}
	return p.Notification.Type
}

func (p *hookPayload) summary() string {
	input := p.toolInput()
	switch p.toolName() {
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			return cmd
		}
	case "Write", "Edit", "MultiEdit", "NotepadEdit":
		path, _ := input["file_path"].(string)
		if path == "" {
			path, _ = input["path"].(string)
		}
		return "file: " + path
	}
	if p.Message != "" {
		return p.Message
	}
	return p.toolName()
}

// hookInputText extracts a displayable string from hook payload fields.
func hookInputText(p hookPayload) string {
	parts := []string{}
	if p.Message != "" {
		parts = append(parts, p.Message)
	}
	if p.Notification != nil && p.Notification.Message != "" {
		parts = append(parts, p.Notification.Message)
	}
	return strings.Join(parts, " ")
}

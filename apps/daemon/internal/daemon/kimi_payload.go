package daemon

import (
	"encoding/json"
	"net/http"
	"strings"
)

type kimiHookPayload struct {
	SessionID    string          `json:"session_id"`
	CWD          string          `json:"cwd"`
	HookEvent    string          `json:"hook_event_name"`
	ToolName     string          `json:"tool_name"`
	ToolInput    map[string]any  `json:"tool_input"`
	ToolOutput   any             `json:"tool_output"`
	Error        string          `json:"error"`
	Prompt       json.RawMessage `json:"prompt"`
	Reason       string          `json:"reason"`
	Source       string          `json:"source"`
	Notification *struct {
		Sink             string `json:"sink"`
		NotificationType string `json:"notification_type"`
		Title            string `json:"title"`
		Body             string `json:"body"`
		Severity         string `json:"severity"`
	} `json:"notification"`
}

// kimiGatedTools are the tools Kimi would itself ask about in interactive
// mode; their PreToolUse becomes a phone approval gate. Everything else is
// auto-allowed (only a started event is emitted).
func kimiGatedTools() map[string]bool {
	return map[string]bool{
		"Shell":          true,
		"Bash":           true,
		"WriteFile":      true,
		"StrReplaceFile": true,
		"BackgroundTask": true,
	}
}

// kimiPromptText extracts the plain text from a UserPromptSubmit payload:
// newer kimi-code sends a ContentPart array, older sends a string.
func kimiPromptText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(p.Text)
		}
		return b.String()
	}
	return ""
}

func kimiStringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func kimiHookSessionID(r *http.Request, p kimiHookPayload) string {
	if sid := r.URL.Query().Get("session"); sid != "" {
		return sid
	}
	return p.SessionID
}

func kimiToolSummary(p kimiHookPayload) string {
	if cmd := kimiStringField(p.ToolInput, "command"); cmd != "" {
		return cmd
	}
	if path := kimiStringField(p.ToolInput, "path"); path != "" {
		return path
	}
	return p.ToolName
}

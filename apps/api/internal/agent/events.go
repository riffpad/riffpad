package agent

import (
	"encoding/json"
	"time"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleSystem    Role = "system"
)

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolCallContent struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Message struct {
	Role       Role        `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	Name       string      `json:"name,omitempty"`
	Timestamp  int64       `json:"timestamp"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type ToolCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Function  struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolResult struct {
	ToolCallID string
	Name       string
	Content    string
	IsError    bool
}

type AgentEvent struct {
	Type        string      `json:"type"`
	Content     string      `json:"content,omitempty"`
	Name        string      `json:"name,omitempty"`
	Args        any         `json:"args,omitempty"`
	Result      any         `json:"result,omitempty"`
	Path        string      `json:"path,omitempty"`
	ToolCallID  string      `json:"toolCallId,omitempty"`
	ToolName    string      `json:"toolName,omitempty"`
	IsError     bool        `json:"isError,omitempty"`
	Message     *Message    `json:"message,omitempty"`
	Timestamp   int64       `json:"timestamp"`
}

func NewUserMessage(content string) Message {
	return Message{
		Role:      RoleUser,
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
	}
}

func NewAssistantMessage(content string) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
	}
}

func NewToolResultMessage(result ToolResult) Message {
	return Message{
		Role:       RoleTool,
		ToolCallID: result.ToolCallID,
		Name:       result.Name,
		Content:    result.Content,
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]any{
			"isError": result.IsError,
		},
	}
}

func (m Message) ToOpenAIMessage() map[string]any {
	msg := map[string]any{
		"role": string(m.Role),
	}
	if m.Content != "" {
		msg["content"] = m.Content
	}
	if len(m.ToolCalls) > 0 {
		msg["tool_calls"] = m.ToolCalls
	}
	if m.ToolCallID != "" {
		msg["tool_call_id"] = m.ToolCallID
		msg["name"] = m.Name
	}
	return msg
}

func ToolCallsFromContent(content string) ([]ToolCall, error) {
	var calls []ToolCall
	if err := json.Unmarshal([]byte(content), &calls); err != nil {
		return nil, err
	}
	return calls, nil
}

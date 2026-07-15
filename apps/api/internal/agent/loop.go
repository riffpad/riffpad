package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
	"github.com/sashabaranov/go-openai"
)

type EventEmitter func(event AgentEvent) error

type LoopConfig struct {
	Client      *openai.Client
	Model       string
	SystemPrompt string
	Sandbox     sandbox.Sandbox
	BeforeToolCall func(ctx context.Context, toolCall ToolCall, args json.RawMessage) (bool, string)
	AfterToolCall  func(ctx context.Context, toolCall ToolCall, result string, isError bool)
}

type Loop struct {
	config  LoopConfig
	emitter EventEmitter
}

func NewLoop(config LoopConfig, emitter EventEmitter) *Loop {
	return &Loop{
		config:  config,
		emitter: emitter,
	}
}

func (l *Loop) Run(ctx context.Context, messages []Message) error {
	if err := l.emit(AgentEvent{Type: "agent_start"}); err != nil {
		return err
	}

	currentMessages := make([]Message, len(messages))
	copy(currentMessages, messages)

	maxTurns := 15
	for turn := 0; turn < maxTurns; turn++ {
		if err := l.emit(AgentEvent{Type: "turn_start"}); err != nil {
			return err
		}

		assistantMsg, err := l.callLLM(ctx, currentMessages)
		if err != nil {
			return l.emitError(err)
		}

		if err := l.emit(AgentEvent{
			Type:      "message_start",
			Message:   &assistantMsg,
			Timestamp: assistantMsg.Timestamp,
		}); err != nil {
			return err
		}

		if err := l.emit(AgentEvent{
			Type:      "message_end",
			Message:   &assistantMsg,
			Timestamp: assistantMsg.Timestamp,
		}); err != nil {
			return err
		}

		currentMessages = append(currentMessages, assistantMsg)

		if len(assistantMsg.ToolCalls) == 0 {
			if err := l.emit(AgentEvent{
				Type:      "turn_end",
				Message:   &assistantMsg,
				Timestamp: assistantMsg.Timestamp,
			}); err != nil {
				return err
			}
			break
		}

		toolResults, err := l.executeToolCalls(ctx, assistantMsg.ToolCalls)
		if err != nil {
			return l.emitError(err)
		}

		for _, result := range toolResults {
			msg := NewToolResultMessage(result)
			currentMessages = append(currentMessages, msg)
		}

		if err := l.emit(AgentEvent{
			Type:      "turn_end",
			Message:   &assistantMsg,
			Timestamp: assistantMsg.Timestamp,
		}); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return l.emit(AgentEvent{
		Type:      "agent_end",
		Timestamp: now(),
	})
}

func (l *Loop) callLLM(ctx context.Context, messages []Message) (Message, error) {
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: l.config.SystemPrompt,
	})

	for _, m := range messages {
		openaiMessages = append(openaiMessages, toOpenAIMessage(m))
	}

	resp, err := l.config.Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       l.config.Model,
		Messages:    openaiMessages,
		Tools:       ToolDefinitions(),
		ToolChoice:  "auto",
	})
	if err != nil {
		return Message{}, fmt.Errorf("llm request: %w", err)
	}

	if len(resp.Choices) == 0 {
		return Message{}, fmt.Errorf("no choices in llm response")
	}

	choice := resp.Choices[0]
	msg := Message{
		Role:      RoleAssistant,
		Content:   choice.Message.Content,
		Timestamp: now(),
		Metadata: map[string]any{
			"finishReason": choice.FinishReason,
		},
	}

	for _, tc := range choice.Message.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, ToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return msg, nil
}

func (l *Loop) executeToolCalls(ctx context.Context, toolCalls []ToolCall) ([]ToolResult, error) {
	results := make([]ToolResult, 0, len(toolCalls))

	for _, tc := range toolCalls {
		tool := FindTool(tc.Function.Name)
		if tool == nil {
			results = append(results, ToolResult{
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    fmt.Sprintf("Tool %q not found", tc.Function.Name),
				IsError:    true,
			})
			continue
		}

		args := json.RawMessage(tc.Function.Arguments)

		if l.config.BeforeToolCall != nil {
			blocked, reason := l.config.BeforeToolCall(ctx, tc, args)
			if blocked {
				results = append(results, ToolResult{
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Content:    fmt.Sprintf("Blocked: %s", reason),
					IsError:    true,
				})
				continue
			}
		}

		if err := l.emit(AgentEvent{
			Type:       "tool_execution_start",
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
			Args:       jsonAny(args),
			Timestamp:  now(),
		}); err != nil {
			return nil, err
		}

		output, isError, err := tool.Execute(ctx, l.config.Sandbox, tc.ID, args)
		if err != nil {
			isError = true
			output = err.Error()
		}

		if l.config.AfterToolCall != nil {
			l.config.AfterToolCall(ctx, tc, output, isError)
		}

		if err := l.emit(AgentEvent{
			Type:       "tool_execution_end",
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
			Result:     output,
			IsError:    isError,
			Timestamp:  now(),
		}); err != nil {
			return nil, err
		}

		if err := l.emit(AgentEvent{
			Type:       "file_change",
			ToolName:   tc.Function.Name,
			Path:       pathFromArgs(tc.Function.Name, args),
			Timestamp:  now(),
		}); err != nil {
			return nil, err
		}

		results = append(results, ToolResult{
			ToolCallID: tc.ID,
			Name:       tc.Function.Name,
			Content:    output,
			IsError:    isError,
		})

		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
	}

	return results, nil
}

func (l *Loop) emit(event AgentEvent) error {
	if event.Timestamp == 0 {
		event.Timestamp = now()
	}
	return l.emitter(event)
}

func (l *Loop) emitError(err error) error {
	_ = l.emit(AgentEvent{
		Type:      "error",
		Content:   err.Error(),
		Timestamp: now(),
	})
	return err
}

func toOpenAIMessage(m Message) openai.ChatCompletionMessage {
	msg := openai.ChatCompletionMessage{
		Role:    string(m.Role),
		Content: m.Content,
	}
	if m.ToolCallID != "" {
		msg.ToolCallID = m.ToolCallID
	}
	if m.Name != "" {
		msg.Name = m.Name
	}
	for _, tc := range m.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, openai.ToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: openai.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return msg
}

func jsonAny(data []byte) any {
	var v any
	_ = json.Unmarshal(data, &v)
	return v
}

func pathFromArgs(toolName string, args json.RawMessage) string {
	var params struct {
		Path      string `json:"path"`
		Directory string `json:"directory"`
	}
	_ = json.Unmarshal(args, &params)
	if params.Path != "" {
		return params.Path
	}
	return params.Directory
}

func now() int64 {
	return time.Now().UnixMilli()
}

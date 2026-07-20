package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
	"github.com/sashabaranov/go-openai"
)

type EventEmitter func(event AgentEvent) error

type LoopConfig struct {
	Client       *openai.Client
	Model        string
	SystemPrompt string
	Sandbox      sandbox.Sandbox
	Lang         string
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

func (l *Loop) Run(ctx context.Context, messages []Message) ([]Message, error) {
	if err := l.emit(AgentEvent{Type: "agent_start"}); err != nil {
		return messages, err
	}

	currentMessages := make([]Message, len(messages))
	copy(currentMessages, messages)

	maxTurns := 8
	finished := false
	for turn := 0; turn < maxTurns; turn++ {
		if err := l.emit(AgentEvent{Type: "turn_start"}); err != nil {
			return currentMessages, err
		}

		assistantMsg, err := l.callLLM(ctx, currentMessages)
		if err != nil {
			_ = l.emitError(err)
			return currentMessages, err
		}

		if err := l.emit(AgentEvent{
			Type:      "message_end",
			Message:   &assistantMsg,
			Timestamp: assistantMsg.Timestamp,
		}); err != nil {
			return currentMessages, err
		}

		currentMessages = append(currentMessages, assistantMsg)

		if len(assistantMsg.ToolCalls) == 0 {
			if err := l.emit(AgentEvent{
				Type:      "turn_end",
				Message:   &assistantMsg,
				Timestamp: assistantMsg.Timestamp,
			}); err != nil {
				return currentMessages, err
			}
			finished = true
			break
		}

		toolResults, err := l.executeToolCalls(ctx, assistantMsg.ToolCalls)
		if err != nil {
			_ = l.emitError(err)
			return currentMessages, err
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
			return currentMessages, err
		}

		select {
		case <-ctx.Done():
			return currentMessages, ctx.Err()
		default:
		}
	}

	// If we exhausted the turn budget without breaking, tell the user instead of
	// silently ending.
	if !finished {
		if err := l.emit(AgentEvent{
			Type:      "error",
			Content:   "Agent reached the maximum number of turns. Please simplify your request.",
			Timestamp: now(),
		}); err != nil {
			return currentMessages, err
		}
	}

	if err := l.emit(AgentEvent{
		Type:      "agent_end",
		Timestamp: now(),
	}); err != nil {
		return currentMessages, err
	}

	return currentMessages, nil
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

	// Two-layer timeout:
	// - 15s to first token: the provider must start emitting content, reasoning, or
	//   tool calls quickly. This prevents long "Working..." hangs.
	// - 60s overall stream: once tokens start flowing, the whole turn must finish
	//   within one minute.
	llmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	stream, err := l.config.Client.CreateChatCompletionStream(llmCtx, openai.ChatCompletionRequest{
		Model:      l.config.Model,
		Messages:   openaiMessages,
		Tools:      ToolDefinitions(l.config.Lang),
		ToolChoice: "auto",
	})
	if err != nil {
		return Message{}, fmt.Errorf("llm request: %w", err)
	}
	defer stream.Close()

	msg := Message{
		Role:      RoleAssistant,
		Timestamp: now(),
		Metadata:  map[string]any{},
	}

	if err := l.emit(AgentEvent{
		Type:      "message_start",
		Message:   &msg,
		Timestamp: msg.Timestamp,
	}); err != nil {
		return Message{}, err
	}

	// Accumulate text and tool calls across streaming chunks.
	var content strings.Builder
	pendingToolCalls := make(map[int]*ToolCall)

	var sawContent bool

	type recvResult struct {
		response openai.ChatCompletionStreamResponse
		err      error
	}

	// recvWithTimeout reads one chunk with a per-call deadline. Once meaningful
	// content has started flowing, we rely on the overall llmCtx (60s) instead.
	recvWithTimeout := func(timeout time.Duration) (openai.ChatCompletionStreamResponse, error) {
		ch := make(chan recvResult, 1)
		go func() {
			resp, err := stream.Recv()
			ch <- recvResult{resp, err}
		}()
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-llmCtx.Done():
			return openai.ChatCompletionStreamResponse{}, llmCtx.Err()
		case <-timer.C:
			return openai.ChatCompletionStreamResponse{}, context.DeadlineExceeded
		case r := <-ch:
			return r.response, r.err
		}
	}

	// Keep reading until we see meaningful content (text, reasoning, or tool call).
	// Each read before that is capped at 15s so a hung provider cannot leave the
	// user on "Working..." for minutes.
	for !sawContent {
		response, err := recvWithTimeout(15 * time.Second)
		if err != nil {
			if errors.Is(err, io.EOF) {
				msg.Content = content.String()
				return msg, nil
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return Message{}, fmt.Errorf("llm first token timeout after 15s")
			}
			return Message{}, fmt.Errorf("llm stream: %w", err)
		}
		if err := l.processDelta(&response, &content, pendingToolCalls, &sawContent, &msg); err != nil {
			return Message{}, err
		}

		// A provider may send finish_reason without any prior content/reasoning
		// (e.g. for refusals) or before we consider content meaningful. End early
		// so a kept-alive connection does not hang the stream.
		if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {
			break
		}
	}

	// Meaningful content has started; consume the rest under the overall 60s
	// deadline. Once tokens are flowing, a 10s idle timeout keeps the UI from
	// hanging if a provider keeps the connection open after the response ends.
	for {
		response, err := recvWithTimeout(10 * time.Second)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, context.DeadlineExceeded) {
				// Treat an idle timeout as a graceful end of stream. Some providers
				// do not send finish_reason or close the connection promptly.
				break
			}
			return Message{}, fmt.Errorf("llm stream: %w", err)
		}

		if err := l.processDelta(&response, &content, pendingToolCalls, &sawContent, &msg); err != nil {
			return Message{}, err
		}

		// Some providers keep the connection open after sending finish_reason.
		// Treat a non-empty finish_reason as the end of the assistant message
		// so the user isn't left waiting on a hung stream.
		if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {
			break
		}

		select {
		case <-ctx.Done():
			return Message{}, ctx.Err()
		default:
		}
	}

	msg.Content = content.String()

	for i := 0; i < len(pendingToolCalls); i++ {
		if tc, ok := pendingToolCalls[i]; ok {
			msg.ToolCalls = append(msg.ToolCalls, *tc)
		}
	}

	msg.Content = content.String()
	return msg, nil
}

func (l *Loop) processDelta(
	response *openai.ChatCompletionStreamResponse,
	content *strings.Builder,
	pendingToolCalls map[int]*ToolCall,
	sawContent *bool,
	msg *Message,
) error {
	if len(response.Choices) == 0 {
		return nil
	}

	choice := response.Choices[0]
	delta := choice.Delta

	if delta.Content != "" {
		*sawContent = true
		content.WriteString(delta.Content)
		if err := l.emit(AgentEvent{
			Type:      "message_delta",
			Delta:     delta.Content,
			Timestamp: now(),
		}); err != nil {
			return err
		}
	}

	if delta.ReasoningContent != "" {
		*sawContent = true
		if err := l.emit(AgentEvent{
			Type:      "reasoning_delta",
			Delta:     delta.ReasoningContent,
			Timestamp: now(),
		}); err != nil {
			return err
		}
	}

	for _, tc := range delta.ToolCalls {
		idx := 0
		if tc.Index != nil {
			idx = *tc.Index
		}
		if _, ok := pendingToolCalls[idx]; !ok {
			pendingToolCalls[idx] = &ToolCall{Type: "function"}
			// As soon as we know a tool is being chosen, surface a pending tool
			// card so the user sees feedback while the model finishes emitting the
			// tool-call JSON.
			if !*sawContent {
				if err := l.emit(AgentEvent{
					Type:      "message_delta",
					Delta:     "I will use a tool to help with that.\n\n",
					Timestamp: now(),
				}); err != nil {
					return err
				}
				*sawContent = true
				content.WriteString("I will use a tool to help with that.\n\n")
			}
		}
		existing := pendingToolCalls[idx]
		if tc.ID != "" {
			existing.ID = tc.ID
		}
		if tc.Type != "" {
			existing.Type = string(tc.Type)
		}
		if tc.Function.Name != "" {
			existing.Function.Name = tc.Function.Name
		}
		existing.Function.Arguments += tc.Function.Arguments

		// Stream the tool-call construction to the UI so the user sees a pending
		// indicator while the arguments JSON is still being generated.
		if err := l.emit(AgentEvent{
			Type:          "tool_call_delta",
			ToolCallID:    existing.ID,
			ToolCallIndex: idx,
			ToolName:      existing.Function.Name,
			Args:          jsonAny([]byte(existing.Function.Arguments)),
			Timestamp:     now(),
		}); err != nil {
			return err
		}
	}

	if choice.FinishReason != "" {
		msg.Metadata["finishReason"] = choice.FinishReason
	}

	return nil
}

func (l *Loop) executeToolCalls(ctx context.Context, toolCalls []ToolCall) ([]ToolResult, error) {
	results := make([]ToolResult, len(toolCalls))

	type indexedResult struct {
		index  int
		result ToolResult
	}

	var wg sync.WaitGroup
	resultCh := make(chan indexedResult, len(toolCalls))

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(i int, tc ToolCall) {
			defer wg.Done()
			resultCh <- indexedResult{
				index:  i,
				result: l.executeSingleToolCall(ctx, i, tc),
			}
		}(i, tc)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for r := range resultCh {
		results[r.index] = r.result
	}

	return results, ctx.Err()
}

func (l *Loop) executeSingleToolCall(ctx context.Context, index int, tc ToolCall) ToolResult {
	tool := FindTool(tc.Function.Name)
	if tool == nil {
		return ToolResult{
			ToolCallID: tc.ID,
			Name:       tc.Function.Name,
			Content:    fmt.Sprintf("Tool %q not found", tc.Function.Name),
			IsError:    true,
		}
	}

	args := json.RawMessage(tc.Function.Arguments)

	if l.config.BeforeToolCall != nil {
		blocked, reason := l.config.BeforeToolCall(ctx, tc, args)
		if blocked {
			return ToolResult{
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    fmt.Sprintf("Blocked: %s", reason),
				IsError:    true,
			}
		}
	}

	_ = l.emit(AgentEvent{
		Type:          "tool_execution_start",
		ToolCallID:    tc.ID,
		ToolCallIndex: index,
		ToolName:      tc.Function.Name,
		Args:          jsonAny(args),
		Timestamp:     now(),
	})

	output, isError, err := tool.Execute(ctx, l.config.Sandbox, tc.ID, args)
	if err != nil {
		isError = true
		output = err.Error()
	}

	if l.config.AfterToolCall != nil {
		l.config.AfterToolCall(ctx, tc, output, isError)
	}

	_ = l.emit(AgentEvent{
		Type:          "tool_execution_end",
		ToolCallID:    tc.ID,
		ToolCallIndex: index,
		ToolName:      tc.Function.Name,
		Result:        output,
		IsError:       isError,
		Timestamp:     now(),
	})

	_ = l.emit(AgentEvent{
		Type:          "file_change",
		ToolCallIndex: index,
		ToolName:      tc.Function.Name,
		Path:          pathFromArgs(tc.Function.Name, args),
		Timestamp:     now(),
	})

	return ToolResult{
		ToolCallID: tc.ID,
		Name:       tc.Function.Name,
		Content:    output,
		IsError:    isError,
	}
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

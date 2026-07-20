package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
	"github.com/sashabaranov/go-openai"
)

// sseChunk builds one OpenAI streaming chunk.
func sseChunk(delta map[string]any, finishReason string) string {
	choice := map[string]any{"index": 0, "delta": delta}
	if finishReason != "" {
		choice["finish_reason"] = finishReason
	}
	payload, _ := json.Marshal(map[string]any{
		"id":      "chatcmpl-test",
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   "test-model",
		"choices": []map[string]any{choice},
	})
	return fmt.Sprintf("data: %s\n\n", payload)
}

// sseDone marks the end of the stream.
func sseDone() string {
	return "data: [DONE]\n\n"
}

// textDeltas returns SSE chunks for a sequence of text tokens.
func textDeltas(tokens ...string) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteString(sseChunk(map[string]any{"content": t}, ""))
	}
	b.WriteString(sseChunk(map[string]any{}, "stop"))
	b.WriteString(sseDone())
	return b.String()
}

// toolCallDeltas returns SSE chunks for a tool call.
func toolCallDeltas(id string, name string, args string) string {
	var b strings.Builder
	b.WriteString(sseChunk(map[string]any{
		"tool_calls": []map[string]any{
			{"index": 0, "id": id, "type": "function", "function": map[string]any{"name": name}},
		},
	}, ""))
	b.WriteString(sseChunk(map[string]any{
		"tool_calls": []map[string]any{
			{"index": 0, "function": map[string]any{"arguments": args}},
		},
	}, ""))
	b.WriteString(sseChunk(map[string]any{}, "stop"))
	b.WriteString(sseDone())
	return b.String()
}

func runLoopWithServer(t *testing.T, streamBody string, prompt string) ([]AgentEvent, Message, error) {
	return runLoopWithHandler(t, func(r *http.Request) string { return streamBody }, prompt)
}

func runLoopWithHandler(t *testing.T, handler func(r *http.Request) string, prompt string) ([]AgentEvent, Message, error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(handler(r)))
	}))
	defer ts.Close()

	cfg := openai.DefaultConfig("test-key")
	cfg.BaseURL = ts.URL
	client := openai.NewClientWithConfig(cfg)

	sb, err := sandbox.NewLocal("test-" + fmt.Sprint(time.Now().UnixNano()))
	if err != nil {
		return nil, Message{}, err
	}
	defer sb.Destroy()

	loop := NewLoop(LoopConfig{
		Client:       client,
		Model:        "test-model",
		SystemPrompt: "You are a helpful assistant.",
		Sandbox:      sb,
	}, nil)

	var events []AgentEvent
	var finalMsg Message
	loop.emitter = func(event AgentEvent) error {
		events = append(events, event)
		if event.Type == "message_end" && event.Message != nil {
			finalMsg = *event.Message
		}
		return nil
	}

	_, err = loop.Run(context.Background(), []Message{NewUserMessage(prompt)})
	return events, finalMsg, err
}

func TestLoop_TextOnlyResponse(t *testing.T) {
	events, finalMsg, err := runLoopWithServer(t, textDeltas("Hello", "!"), "hi")
	if err != nil {
		t.Fatalf("loop failed: %v", err)
	}

	wantTypes := []string{"agent_start", "turn_start", "message_start", "message_delta", "message_delta", "message_end", "turn_end", "agent_end"}
	gotTypes := make([]string, len(events))
	for i, e := range events {
		gotTypes[i] = e.Type
	}
	if strings.Join(gotTypes, ",") != strings.Join(wantTypes, ",") {
		t.Fatalf("event type mismatch\ngot:  %v\nwant: %v", gotTypes, wantTypes)
	}

	if finalMsg.Role != RoleAssistant {
		t.Fatalf("expected assistant message, got %s", finalMsg.Role)
	}
	if finalMsg.Content != "Hello!" {
		t.Fatalf("expected content 'Hello!', got %q", finalMsg.Content)
	}
	if len(finalMsg.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(finalMsg.ToolCalls))
	}
}

func TestLoop_SingleToolCall(t *testing.T) {
	args, _ := json.Marshal(map[string]string{"path": "hello.txt", "content": "world"})
	first := toolCallDeltas("call-1", "file_write", string(args))
	second := textDeltas("Done")

	callCount := 0
	events, finalMsg, err := runLoopWithHandler(t, func(r *http.Request) string {
		callCount++
		if callCount == 1 {
			return first
		}
		return second
	}, "write a file")
	if err != nil {
		t.Fatalf("loop failed: %v", err)
	}

	if callCount != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", callCount)
	}

	if len(events) < 10 {
		t.Fatalf("expected at least 10 events, got %d: %v", len(events), eventTypes(events))
	}

	if !hasEventType(events, "tool_execution_start") {
		t.Fatalf("expected tool_execution_start event")
	}
	if !hasEventType(events, "tool_execution_end") {
		t.Fatalf("expected tool_execution_end event")
	}
	if !hasEventType(events, "file_change") {
		t.Fatalf("expected file_change event")
	}

	if finalMsg.Role != RoleAssistant {
		t.Fatalf("expected assistant message, got %s", finalMsg.Role)
	}
	if finalMsg.Content != "Done" {
		t.Fatalf("expected content 'Done', got %q", finalMsg.Content)
	}
	if len(finalMsg.ToolCalls) != 0 {
		t.Fatalf("expected final message without tool calls, got %d", len(finalMsg.ToolCalls))
	}
}

func hasEventType(events []AgentEvent, typ string) bool {
	for _, e := range events {
		if e.Type == typ {
			return true
		}
	}
	return false
}

func eventTypes(events []AgentEvent) []string {
	types := make([]string, len(events))
	for i, e := range events {
		types[i] = e.Type
	}
	return types
}

// multiToolCallDeltas returns SSE chunks for N tool calls in one assistant turn.
func multiToolCallDeltas(calls ...struct {
	id   string
	name string
	args string
}) string {
	var b strings.Builder
	for i, c := range calls {
		b.WriteString(sseChunk(map[string]any{
			"tool_calls": []map[string]any{
				{"index": i, "id": c.id, "type": "function", "function": map[string]any{"name": c.name}},
			},
		}, ""))
		b.WriteString(sseChunk(map[string]any{
			"tool_calls": []map[string]any{
				{"index": i, "function": map[string]any{"arguments": c.args}},
			},
		}, ""))
	}
	b.WriteString(sseChunk(map[string]any{}, "stop"))
	b.WriteString(sseDone())
	return b.String()
}

func TestLoop_ReturnsUpdatedHistory(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(textDeltas("Hello")))
	}))
	defer ts.Close()

	cfg := openai.DefaultConfig("test-key")
	cfg.BaseURL = ts.URL
	client := openai.NewClientWithConfig(cfg)

	sb, err := sandbox.NewLocal("test-history-" + fmt.Sprint(time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	defer sb.Destroy()

	loop := NewLoop(LoopConfig{
		Client:       client,
		Model:        "test-model",
		SystemPrompt: "You are a helpful assistant.",
		Sandbox:      sb,
	}, func(event AgentEvent) error { return nil })

	initial := []Message{NewUserMessage("hi")}
	updated, err := loop.Run(context.Background(), initial)
	if err != nil {
		t.Fatalf("loop failed: %v", err)
	}
	if len(updated) != 2 {
		t.Fatalf("expected 2 messages (user + assistant), got %d", len(updated))
	}
	if updated[0].Role != RoleUser {
		t.Fatalf("expected first message role user, got %s", updated[0].Role)
	}
	if updated[1].Role != RoleAssistant {
		t.Fatalf("expected second message role assistant, got %s", updated[1].Role)
	}
	if updated[1].Content != "Hello" {
		t.Fatalf("expected assistant content 'Hello', got %q", updated[1].Content)
	}
}

func TestLoop_ParallelToolCalls(t *testing.T) {
	argsA, _ := json.Marshal(map[string]string{"command": "sleep 0.1 && echo a"})
	argsB, _ := json.Marshal(map[string]string{"command": "sleep 0.1 && echo b"})

	first := multiToolCallDeltas(
		struct{ id, name, args string }{"call-1", "bash_exec", string(argsA)},
		struct{ id, name, args string }{"call-2", "bash_exec", string(argsB)},
	)
	second := textDeltas("Done")

	callCount := 0
	events, finalMsg, err := runLoopWithHandler(t, func(r *http.Request) string {
		callCount++
		if callCount == 1 {
			return first
		}
		return second
	}, "run two commands")
	if err != nil {
		t.Fatalf("loop failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", callCount)
	}
	if finalMsg.Content != "Done" {
		t.Fatalf("expected final content 'Done', got %q", finalMsg.Content)
	}

	var starts []AgentEvent
	var ends []AgentEvent
	for _, e := range events {
		switch e.Type {
		case "tool_execution_start":
			starts = append(starts, e)
		case "tool_execution_end":
			ends = append(ends, e)
		}
	}
	if len(starts) != 2 {
		t.Fatalf("expected 2 tool_execution_start events, got %d", len(starts))
	}
	if len(ends) != 2 {
		t.Fatalf("expected 2 tool_execution_end events, got %d", len(ends))
	}

	// If the two sleeps ran in parallel, the second start event should have
	// fired before the first end event.
	if starts[1].Timestamp >= ends[0].Timestamp {
		t.Fatalf("tool calls appear sequential: second start (%d) is not before first end (%d)", starts[1].Timestamp, ends[0].Timestamp)
	}

}

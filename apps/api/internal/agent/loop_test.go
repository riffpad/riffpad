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

	err = loop.Run(context.Background(), []Message{NewUserMessage(prompt)})
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

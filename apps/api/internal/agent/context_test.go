package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(Message{Role: RoleUser, Content: "hello"}); got != 1 {
		t.Fatalf("expected 1 token for 5 chars, got %d", got)
	}
	if got := EstimateTokens(Message{Role: RoleUser, Content: strings.Repeat("a", 400)}); got != 100 {
		t.Fatalf("expected 100 tokens for 400 chars, got %d", got)
	}
}

func TestShouldCompact_TokenBased(t *testing.T) {
	cfg := ContextConfig{ContextWindow: 1000, ReserveTokens: 200}
	// Below threshold: 1000 - 200 = 800.
	messages := []Message{
		NewUserMessage(strings.Repeat("a", 800)), // ~200 tokens
	}
	if ShouldCompact(messages, cfg) {
		t.Fatalf("should not compact below context window minus reserve")
	}
	// Above threshold.
	messages = append(messages, NewUserMessage(strings.Repeat("b", 2404))) // ~801 tokens total
	if !ShouldCompact(messages, cfg) {
		t.Fatalf("should compact when tokens exceed context window minus reserve")
	}
}

func TestShouldCompact_NoTurnCount(t *testing.T) {
	// Many short turns should not trigger compaction on their own.
	cfg := ContextConfig{ContextWindow: 10000, ReserveTokens: 2000}
	messages := []Message{}
	for i := 0; i < 50; i++ {
		messages = append(messages, NewUserMessage("hi"), NewAssistantMessage("hello"))
	}
	if ShouldCompact(messages, cfg) {
		t.Fatalf("should not compact based on turn count")
	}
}

func TestFindCutPoint_KeepsTurnBoundary(t *testing.T) {
	messages := []Message{
		NewUserMessage(strings.Repeat("a", 40)),  // ~10 tokens
		NewAssistantMessage(strings.Repeat("b", 40)), // ~10 tokens
		NewUserMessage(strings.Repeat("c", 40)),  // ~10 tokens
		NewAssistantMessage(strings.Repeat("d", 40)), // ~10 tokens
	}
	// Budget lands on the user message that starts the last turn.
	if got := findCutPoint(messages, 15); got.Index != 2 {
		t.Fatalf("expected cut at turn boundary 2, got %d", got.Index)
	}
	// Keep plenty; should keep all.
	if got := findCutPoint(messages, 10000); got.Index != 0 {
		t.Fatalf("expected keep all (0), got %d", got.Index)
	}
}

func assistantWithToolCall(content string, tc ToolCall) Message {
	m := NewAssistantMessage(content)
	m.ToolCalls = []ToolCall{tc}
	return m
}

func TestFindCutPoint_NeverOnToolResult(t *testing.T) {
	messages := []Message{
		NewUserMessage("first"),
		NewAssistantMessage("second"),
		NewUserMessage("read file"),
		assistantWithToolCall("", ToolCall{ID: "1", Type: "function", Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{Name: "file_read", Arguments: `{"path":"x"}`}}),
		NewToolResultMessage(ToolResult{ToolCallID: "1", Name: "file_read", Content: strings.Repeat("a", 200)}),
		NewUserMessage("last"),
	}
	// Budget falls inside the tool result. The cut must never land on the tool
	// result itself; instead we split at the preceding assistant (tool call).
	got := findCutPoint(messages, 30)
	if got.Index != 3 {
		t.Fatalf("expected cut at assistant message 3, got %d", got.Index)
	}
	if messages[got.Index].Role != RoleAssistant {
		t.Fatalf("expected cut on assistant message, got %s", messages[got.Index].Role)
	}
	if !got.SplitTurn {
		t.Fatalf("expected split turn because the oversized turn exceeds keep budget")
	}
}

func TestFindCutPoint_SplitTurn(t *testing.T) {
	messages := []Message{
		NewUserMessage("first"),
		NewAssistantMessage("second"),
		NewUserMessage("read huge file"),
		assistantWithToolCall("", ToolCall{ID: "1", Type: "function", Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{Name: "file_read", Arguments: `{"path":"x"}`}}),
		NewToolResultMessage(ToolResult{ToolCallID: "1", Name: "file_read", Content: strings.Repeat("a", 4000)}),
	}
	// The tool result alone is ~1000 tokens, which exceeds the recent budget.
	// The legal boundary before the budget-aligned index is the assistant
	// message with the tool call, and rolling back to the turn start would keep
	// too much. Expect a split turn.
	got := findCutPoint(messages, 50)
	if !got.SplitTurn {
		t.Fatalf("expected split turn, got %+v", got)
	}
	if got.Index != 3 {
		t.Fatalf("expected cut at assistant message 3, got %d", got.Index)
	}
	if got.TurnStart != 2 {
		t.Fatalf("expected turn start 2, got %d", got.TurnStart)
	}
}

func TestCompact_SummarizesOldHistory(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-test",
			"object": "chat.completion",
			"created": 1,
			"model": "test-model",
			"choices": [{"index":0,"message":{"role":"assistant","content":"## Goal\nBuild a page.\n\n## Progress\n### Done\n- [x] Created index.html"},"finish_reason":"stop"}]
		}`))
	}))
	defer ts.Close()

	cfg := openai.DefaultConfig("test-key")
	cfg.BaseURL = ts.URL
	client := openai.NewClientWithConfig(cfg)

	ctxCfg := ContextConfig{
		ContextWindow:    128000,
		ReserveTokens:    8000,
		KeepRecentTokens: 1, // force everything old to be summarized
	}

	messages := []Message{
		NewUserMessage("build a landing page"),
		NewAssistantMessage("I will create index.html"),
		NewUserMessage("ok"),
	}

	compacted, err := Compact(context.Background(), client, "test-model", messages, ctxCfg)
	if err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	if len(compacted) != 2 {
		t.Fatalf("expected 2 messages (summary + recent), got %d", len(compacted))
	}
	if compacted[0].Role != RoleSystem {
		t.Fatalf("expected summary as system message, got %s", compacted[0].Role)
	}
	if compacted[0].Metadata["kind"] != "history" {
		t.Fatalf("expected history summary, got %v", compacted[0].Metadata["kind"])
	}
	if !strings.Contains(compacted[0].Content, "## Goal") {
		t.Fatalf("expected structured summary, got %q", compacted[0].Content)
	}
	if compacted[1].Content != "ok" {
		t.Fatalf("expected recent user message kept verbatim, got %q", compacted[1].Content)
	}
}

func TestCompact_SplitTurn(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-test",
			"object": "chat.completion",
			"created": 1,
			"model": "test-model",
			"choices": [{"index":0,"message":{"role":"assistant","content":"## Goal\nContinue.\n\n## Progress\n### Done\n- [x] Read file"},"finish_reason":"stop"}]
		}`))
	}))
	defer ts.Close()

	cfg := openai.DefaultConfig("test-key")
	cfg.BaseURL = ts.URL
	client := openai.NewClientWithConfig(cfg)

	ctxCfg := ContextConfig{
		ContextWindow:    128000,
		ReserveTokens:    8000,
		KeepRecentTokens: 50, // force a split inside the oversized turn
	}

	messages := []Message{
		NewUserMessage("old question"),
		NewAssistantMessage("old answer"),
		NewUserMessage("read huge file"),
		assistantWithToolCall("", ToolCall{ID: "1", Type: "function", Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{Name: "file_read", Arguments: `{"path":"x"}`}}),
		NewToolResultMessage(ToolResult{ToolCallID: "1", Name: "file_read", Content: strings.Repeat("a", 4000)}),
	}

	compacted, err := Compact(context.Background(), client, "test-model", messages, ctxCfg)
	if err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 summarization calls (history + turn prefix), got %d", callCount)
	}
	if len(compacted) != 4 {
		t.Fatalf("expected 4 messages (history summary + turn prefix summary + 2 recent), got %d", len(compacted))
	}
	if compacted[0].Metadata["kind"] != "history" {
		t.Fatalf("expected first summary kind history, got %v", compacted[0].Metadata["kind"])
	}
	if compacted[1].Metadata["kind"] != "turn-prefix" {
		t.Fatalf("expected second summary kind turn-prefix, got %v", compacted[1].Metadata["kind"])
	}
	if compacted[2].Role != RoleAssistant {
		t.Fatalf("expected recent assistant message kept verbatim, got %s", compacted[2].Role)
	}
	if compacted[3].Role != RoleTool {
		t.Fatalf("expected recent tool result kept verbatim, got %s", compacted[3].Role)
	}
}

func TestCompact_FallsBackToTruncationOnError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cfg := openai.DefaultConfig("test-key")
	cfg.BaseURL = ts.URL
	client := openai.NewClientWithConfig(cfg)

	ctxCfg := ContextConfig{
		ContextWindow:    128000,
		ReserveTokens:    8000,
		KeepRecentTokens: 1,
	}

	messages := []Message{
		NewUserMessage("old"),
		NewAssistantMessage("older"),
		NewUserMessage("recent"),
	}

	compacted, err := Compact(context.Background(), client, "test-model", messages, ctxCfg)
	if err == nil {
		t.Fatalf("expected error on summarization failure")
	}
	if len(compacted) != 1 || compacted[0].Content != "recent" {
		t.Fatalf("expected fallback to recent messages only, got %v", compacted)
	}
}

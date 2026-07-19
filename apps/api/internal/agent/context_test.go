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

func TestShouldCompact_Turns(t *testing.T) {
	cfg := ContextConfig{ContextWindow: 128000, ThresholdRatio: 0.6, MaxTurnsBeforeCompact: 3}
	messages := []Message{
		NewUserMessage("a"),
		NewAssistantMessage("b"),
		NewUserMessage("c"),
		NewAssistantMessage("d"),
	}
	if ShouldCompact(messages, cfg) {
		t.Fatalf("should not compact below max turns")
	}
	messages = append(messages, NewUserMessage("e"), NewAssistantMessage("f"), NewUserMessage("g"))
	if !ShouldCompact(messages, cfg) {
		t.Fatalf("should compact after exceeding max turns")
	}
}

func TestShouldCompact_Tokens(t *testing.T) {
	cfg := ContextConfig{ContextWindow: 1000, ThresholdRatio: 0.5, MaxTurnsBeforeCompact: 100}
	messages := []Message{
		NewUserMessage(strings.Repeat("a", 1000)), // ~250 tokens
	}
	if ShouldCompact(messages, cfg) {
		t.Fatalf("should not compact below threshold")
	}
	messages = append(messages, NewUserMessage(strings.Repeat("b", 2004))) // ~751 tokens total
	if !ShouldCompact(messages, cfg) {
		t.Fatalf("should compact above threshold")
	}
}

func TestCutPoint_KeepsTurnBoundary(t *testing.T) {
	messages := []Message{
		NewUserMessage("first"),
		NewAssistantMessage("second"),
		NewUserMessage("third"),
		NewAssistantMessage("fourth"),
	}
	// Keep only ~1 token; should fall back to start of current turn.
	if got := CutPoint(messages, 1); got != 2 {
		t.Fatalf("expected cut at turn boundary 2, got %d", got)
	}
	// Keep plenty; should keep all.
	if got := CutPoint(messages, 10000); got != 0 {
		t.Fatalf("expected keep all (0), got %d", got)
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
		ContextWindow:         128000,
		ThresholdRatio:        0.6,
		MaxTurnsBeforeCompact: 100,
		ReserveTokens:         8000,
		KeepRecentTokens:      1, // force everything old to be summarized
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
	if !strings.Contains(compacted[0].Content, "## Goal") {
		t.Fatalf("expected structured summary, got %q", compacted[0].Content)
	}
	if compacted[1].Content != "ok" {
		t.Fatalf("expected recent user message kept verbatim, got %q", compacted[1].Content)
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
		ContextWindow:         128000,
		ThresholdRatio:        0.6,
		MaxTurnsBeforeCompact: 100,
		ReserveTokens:         8000,
		KeepRecentTokens:      1,
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

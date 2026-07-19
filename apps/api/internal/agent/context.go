package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// ContextConfig controls when and how conversation history is compacted.
type ContextConfig struct {
	// ContextWindow is the model's maximum context length in tokens.
	ContextWindow int
	// ThresholdRatio triggers compaction when estimated tokens exceed ContextWindow * ThresholdRatio.
	ThresholdRatio float64
	// MaxTurnsBeforeCompact triggers compaction when the conversation exceeds this many user/assistant turns.
	MaxTurnsBeforeCompact int
	// ReserveTokens is the budget reserved for the model's response and compaction summary.
	ReserveTokens int
	// KeepRecentTokens is the approximate token budget kept verbatim after compaction.
	KeepRecentTokens int
}

// DefaultContextConfig returns sensible defaults for a 128k context window.
func DefaultContextConfig() ContextConfig {
	return ContextConfig{
		ContextWindow:         128000,
		ThresholdRatio:        0.6,
		MaxTurnsBeforeCompact: 20,
		ReserveTokens:         8000,
		KeepRecentTokens:      16000,
	}
}

func (c ContextConfig) thresholdTokens() int {
	return int(float64(c.ContextWindow) * c.ThresholdRatio)
}

// EstimateTokens returns a conservative token estimate for a message using a
// simple character heuristic. This is intentionally cheap and local; providers
// report exact usage asynchronously.
func EstimateTokens(m Message) int {
	var chars int
	switch m.Role {
	case RoleUser, RoleTool, RoleSystem:
		chars = len(m.Content)
		for _, tc := range m.ToolCalls {
			chars += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
	case RoleAssistant:
		chars = len(m.Content)
		for _, tc := range m.ToolCalls {
			chars += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
	}
	// Approximate 1 token ~= 4 chars for English/code; this over-estimates CJK
	// slightly, which is fine for a safety heuristic.
	return max(1, chars/4)
}

// EstimateMessagesTokens sums token estimates for a slice of messages.
func EstimateMessagesTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(m)
	}
	return total
}

// ShouldCompact returns true when history exceeds the configured turn or token threshold.
func ShouldCompact(messages []Message, cfg ContextConfig) bool {
	turns := countUserAssistantTurns(messages)
	if cfg.MaxTurnsBeforeCompact > 0 && turns > cfg.MaxTurnsBeforeCompact {
		return true
	}
	if cfg.ContextWindow > 0 {
		threshold := cfg.thresholdTokens()
		if threshold > 0 && EstimateMessagesTokens(messages) > threshold {
			return true
		}
	}
	return false
}

func countUserAssistantTurns(messages []Message) int {
	turns := 0
	for _, m := range messages {
		if m.Role == RoleUser {
			turns++
		}
	}
	return turns
}

// CutPoint selects the index of the first message to keep verbatim. Messages
// before this index will be summarized.
func CutPoint(messages []Message, keepRecentTokens int) int {
	if keepRecentTokens <= 0 {
		return len(messages)
	}
	accumulated := 0
	for i := len(messages) - 1; i >= 0; i-- {
		accumulated += EstimateTokens(messages[i])
		if accumulated >= keepRecentTokens {
			// Never cut inside a tool-call pair: walk back to the start of the turn.
			for j := i; j >= 0; j-- {
				if messages[j].Role == RoleUser {
					return j
				}
			}
			return i
		}
	}
	return 0
}

// Compact summarizes older messages and returns a cache-friendly message list:
//   [system summary of old history] + [recent messages kept verbatim]
//
// The summary is placed as a system message so it sits immediately after the
// main system prompt in the final LLM request, keeping the prefix stable and
// cache-friendly.
func Compact(
	ctx context.Context,
	client *openai.Client,
	model string,
	messages []Message,
	cfg ContextConfig,
) ([]Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	cut := CutPoint(messages, cfg.KeepRecentTokens)
	if cut == 0 {
		// Everything is recent; nothing to compact.
		return messages, nil
	}

	history := messages[:cut]
	recent := messages[cut:]

	summary, err := summarizeMessages(ctx, client, model, history, cfg.ReserveTokens)
	if err != nil {
		// If summarization fails, fall back to truncating oldest messages. This
		// preserves correctness at the cost of losing some context.
		return recent, fmt.Errorf("compaction summarization failed, truncated oldest messages: %w", err)
	}

	summaryMsg := Message{
		Role:      RoleSystem,
		Content:   summary,
		Timestamp: now(),
		Metadata: map[string]any{
			"compaction":    true,
			"tokensBefore":  EstimateMessagesTokens(messages),
			"recentMessages": len(recent),
		},
	}

	out := make([]Message, 0, 1+len(recent))
	out = append(out, summaryMsg)
	out = append(out, recent...)
	return out, nil
}

const compactionSystemPrompt = `You are a context summarization assistant. Read the conversation and produce a structured summary that another AI assistant will use to continue the work.

Use this EXACT format:

## Goal
[What is the user trying to accomplish?]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned by the user]
- [Or "(none)" if none were mentioned]

## Progress
### Done
- [x] [Completed tasks/changes]

### In Progress
- [ ] [Current work]

### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [Ordered list of what should happen next]

## Critical Context
- [Any data, examples, file paths, function names, or error messages needed to continue]
- [Or "(none)" if not applicable]

Keep each section concise. Preserve exact file paths, function names, and error messages. Do NOT continue the conversation. Do NOT respond to questions in the conversation.`

func summarizeMessages(
	ctx context.Context,
	client *openai.Client,
	model string,
	messages []Message,
	reserveTokens int,
) (string, error) {
	llmMessages := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	for _, m := range messages {
		llmMessages = append(llmMessages, toOpenAIMessage(m))
	}

	maxTokens := min(2048, max(512, reserveTokens/4))
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     model,
		Messages: append([]openai.ChatCompletionMessage{{
			Role:    openai.ChatMessageRoleSystem,
			Content: compactionSystemPrompt,
		}}, llmMessages...),
		MaxTokens: maxTokens,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty summarization response")
	}
	return resp.Choices[0].Message.Content, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func jsonCompact(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

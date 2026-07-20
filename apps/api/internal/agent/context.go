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
	// ReserveTokens is the budget reserved for the model's response and compaction summary.
	ReserveTokens int
	// KeepRecentTokens is the approximate token budget kept verbatim after compaction.
	KeepRecentTokens int
}

// DefaultContextConfig returns sensible defaults for a 128k context window.
func DefaultContextConfig() ContextConfig {
	return ContextConfig{
		ContextWindow:    128000,
		ReserveTokens:    16384,
		KeepRecentTokens: 20480,
	}
}

// ShouldCompact returns true when the estimated token count approaches the
// context window ceiling. We trigger when there is no longer enough headroom
// for both the model's reply and a compaction reserve.
func ShouldCompact(messages []Message, cfg ContextConfig) bool {
	if cfg.ContextWindow <= 0 {
		return false
	}
	threshold := cfg.ContextWindow - cfg.ReserveTokens
	if threshold <= 0 {
		return false
	}
	return EstimateMessagesTokens(messages) > threshold
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

// CutPoint selects the first message to keep verbatim. Messages before this
// index are candidates for summarization. The returned struct also reports
// whether the cut falls inside a single oversized turn; in that case the
// caller should summarize the prefix of that turn separately.
type CutPoint struct {
	// Index of the first message to keep verbatim.
	Index int
	// SplitTurn is true when the budget-aligned cut landed inside a turn that
	// is itself larger than KeepRecentTokens. The caller must summarize
	// messages[TurnStart:Index] as a turn prefix and keep messages[Index:] as
	// the recent suffix.
	SplitTurn bool
	// TurnStart is the index of the user message that begins the turn
	// containing the cut. Meaningful only when SplitTurn is true.
	TurnStart int
}

// findCutPoint walks backward from the newest message, accumulating tokens
// until KeepRecentTokens is reached. It then searches backward for the nearest
// legal cut boundary.
//
// Legal boundaries are user, assistant, and system messages. Tool results are
// never cut points because each tool result is bound to its preceding tool
// call; splitting the pair leaves the model with a dangling call or result.
//
// When the accumulated budget lands inside a turn, we normally roll back to
// the turn's start (the user message). If that rollback would keep far more
// than the budget because the turn itself is huge, we instead report a split
// turn so the caller can summarize the turn's prefix.
func findCutPoint(messages []Message, keepRecentTokens int) CutPoint {
	if keepRecentTokens <= 0 {
		return CutPoint{Index: len(messages)}
	}
	if len(messages) == 0 {
		return CutPoint{Index: 0}
	}

	accumulated := 0
	budgetIndex := -1
	for i := len(messages) - 1; i >= 0; i-- {
		accumulated += EstimateTokens(messages[i])
		if accumulated >= keepRecentTokens {
			budgetIndex = i
			break
		}
	}
	if budgetIndex < 0 {
		// Whole history fits in the recent budget.
		return CutPoint{Index: 0}
	}

	// Find the nearest non-tool message at or before budgetIndex. This is the
	// natural semantic boundary for a cut; tool results are always kept with
	// their preceding tool-call message.
	candidate := -1
	for i := budgetIndex; i >= 0; i-- {
		if messages[i].Role != RoleTool {
			candidate = i
			break
		}
	}
	if candidate < 0 {
		// No legal boundary found; keep everything to avoid corrupting history.
		return CutPoint{Index: 0}
	}

	// If the candidate is already a user message, it is a clean turn start.
	if messages[candidate].Role == RoleUser {
		return CutPoint{Index: candidate}
	}

	// The candidate is an assistant/system message inside a turn. Find the
	// user message that begins this turn.
	turnStart := 0
	for i := candidate; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			turnStart = i
			break
		}
	}

	// If the entire turn fits in the recent budget, roll back to the turn
	// start to keep the turn intact.
	turnTokens := EstimateMessagesTokens(messages[turnStart:])
	if turnTokens <= keepRecentTokens {
		return CutPoint{Index: turnStart}
	}

	// The turn itself exceeds the recent budget. Split it: summarize the turn
	// prefix up to the candidate and keep the suffix verbatim.
	return CutPoint{
		Index:     candidate,
		SplitTurn: true,
		TurnStart: turnStart,
	}
}

// Compact summarizes older messages and returns a cache-friendly message list.
// The output format depends on where the cut point lands:
//
//   - Normal cut at a turn boundary:
//     [system summary of messages[:cut]] + [recent messages]
//
//   - Split turn (a single turn exceeds KeepRecentTokens):
//     [system summary of messages[:turnStart]] +
//     [system summary of messages[turnStart:cut]] +
//     [recent messages]
//
// Summaries are placed as system messages immediately after the main system
// prompt in the final LLM request, keeping the prefix stable and cache-friendly.
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

	cut := findCutPoint(messages, cfg.KeepRecentTokens)
	if cut.Index == 0 && !cut.SplitTurn {
		// Everything is recent; nothing to compact.
		return messages, nil
	}

	out := make([]Message, 0, 2+len(messages)-cut.Index)

	if cut.SplitTurn {
		// Summarize history before the oversized turn.
		if cut.TurnStart > 0 {
			historySummary, err := summarizeMessages(ctx, client, model, messages[:cut.TurnStart], cfg.ReserveTokens)
			if err != nil {
				// Fallback: keep only the recent suffix of the oversized turn.
				return appendRecent(messages[cut.Index:], cfg), fmt.Errorf("history summarization failed, truncated oldest messages: %w", err)
			}
			out = append(out, summaryMessage(historySummary, messages, len(messages)-cut.Index, "history"))
		}

		// Summarize the prefix of the oversized turn.
		turnPrefix := messages[cut.TurnStart:cut.Index]
		turnSummary, err := summarizeMessages(ctx, client, model, turnPrefix, cfg.ReserveTokens)
		if err != nil {
			return appendRecent(messages[cut.Index:], cfg), fmt.Errorf("turn prefix summarization failed, truncated turn prefix: %w", err)
		}
		out = append(out, summaryMessage(turnSummary, messages, len(messages)-cut.Index, "turn-prefix"))
	} else {
		history := messages[:cut.Index]
		summary, err := summarizeMessages(ctx, client, model, history, cfg.ReserveTokens)
		if err != nil {
			// If summarization fails, fall back to truncating oldest messages. This
			// preserves correctness at the cost of losing some context.
			return appendRecent(messages[cut.Index:], cfg), fmt.Errorf("compaction summarization failed, truncated oldest messages: %w", err)
		}
		out = append(out, summaryMessage(summary, messages, len(messages)-cut.Index, "history"))
	}

	out = append(out, messages[cut.Index:]...)
	return out, nil
}

func appendRecent(recent []Message, cfg ContextConfig) []Message {
	// Keep as much recent context as the budget allows.
	cut := findCutPoint(recent, cfg.KeepRecentTokens)
	return recent[cut.Index:]
}

func summaryMessage(summary string, original []Message, recentCount int, kind string) Message {
	return Message{
		Role:      RoleSystem,
		Content:   summary,
		Timestamp: now(),
		Metadata: map[string]any{
			"compaction":     true,
			"kind":           kind,
			"tokensBefore":   EstimateMessagesTokens(original),
			"recentMessages": recentCount,
		},
	}
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
		Messages:  append([]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: compactionSystemPrompt}}, llmMessages...),
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

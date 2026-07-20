package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
	"github.com/sashabaranov/go-openai"
)

type PermissionLevel string

const (
	PermissionSafe    PermissionLevel = "safe"
	PermissionConfirm PermissionLevel = "confirm"
	PermissionAuto    PermissionLevel = "auto"
)

type Orchestrator struct {
	client     *openai.Client
	model      string
	permission PermissionLevel
}

func NewOrchestrator(client *openai.Client, model string, permission PermissionLevel) *Orchestrator {
	if permission == "" {
		permission = PermissionConfirm
	}
	return &Orchestrator{
		client:     client,
		model:      model,
		permission: permission,
	}
}

func (o *Orchestrator) SystemPrompt() string {
	var b strings.Builder
	b.WriteString("You are Riffpad, an AI coding assistant inside a sandboxed workspace.\n\n")
	b.WriteString("Your goal is to help the user turn ideas into running prototypes.\n")
	b.WriteString("You can read, write, and list files, and execute shell commands in the workspace.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Prefer small, iterative steps.\n")
	b.WriteString("- Always write complete file contents; do not emit truncated code.\n")
	b.WriteString("- After writing files, run the appropriate command to verify (e.g. `python main.py`, `node index.js`).\n")
	b.WriteString("- If a command fails, read the relevant files, fix the issue, and try again.\n")
	b.WriteString("- Do not access paths outside the workspace.\n")
	b.WriteString("- Do not execute destructive commands like `rm -rf /`.\n")
	b.WriteString("- Only use tools when the user explicitly asks you to act on the workspace.\n")
	b.WriteString("- For greetings, simple questions, or clarification, reply directly without calling any tool.\n")
	b.WriteString("- When multiple independent tools are needed for one request, call them all in parallel in the same turn.\n")
	b.WriteString("- When you use web_search, you MUST cite the sources in your final answer using the exact bracketed format [1], [2], etc. matching the result index.\n")
	b.WriteString("  Correct: \"Beijing is sunny today [1].\" \"AccuWeather and EaseWeather both report clear skies [1][2].\"\n")
	b.WriteString("  Wrong: \"Beijing is sunny today 1.\" \"Beijing is sunny today 12.\" \"Beijing is sunny today [12].\"\n")
	b.WriteString("- When asked to start a dev server, use a background command (e.g. `nohup python -m http.server 8082 > server.log 2>&1 &`) so it does not block.\n")
	b.WriteString("- After starting a server, verify it with `curl http://localhost:<port>` before reporting success.\n\n")
	b.WriteString(fmt.Sprintf("Current permission level: %s.\n", o.permission))
	if o.permission == PermissionSafe {
		b.WriteString("You are in read-only mode. You may only read and list files.\n")
	} else if o.permission == PermissionConfirm {
		b.WriteString("Side-effect tools (write, bash) require user confirmation; the runtime will ask for you.\n")
	} else {
		b.WriteString("You may execute side-effect tools automatically.\n")
	}
	return b.String()
}

func (o *Orchestrator) Run(ctx context.Context, box sandbox.Sandbox, messages []Message, emitter EventEmitter, ctxCfg ContextConfig) ([]Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// Compact history when it grows too large. The compacted list is
	// cache-friendly: main system prompt + summary + recent messages.
	if ShouldCompact(messages, ctxCfg) {
		compacted, err := Compact(ctx, o.client, o.model, messages, ctxCfg)
		if err == nil {
			messages = compacted
		}
		// On compaction failure we continue with the original messages; the loop
		// may still succeed and the next turn will retry compaction.
	}

	// Agent prompts and tool definitions are always in English for cache stability.
	// UI localization is handled by the frontend.
	const lang = "en"
	loop := NewLoop(LoopConfig{
		Client:       o.client,
		Model:        o.model,
		SystemPrompt: o.SystemPrompt(),
		Sandbox:      box,
		Lang:         lang,
		BeforeToolCall: func(ctx context.Context, toolCall ToolCall, args json.RawMessage) (bool, string) {
			if o.permission == PermissionSafe {
				return true, "safe mode: side-effect tools are disabled"
			}
			if o.permission == PermissionConfirm {
				// In a real implementation, this would queue a confirmation request and wait.
				// For the MVP, we auto-allow after emitting a confirmation event.
				_ = emitter(AgentEvent{
					Type:      "confirm_request",
					ToolName:  toolCall.Function.Name,
					Args:      jsonAny(args),
					Timestamp: now(),
				})
			}
			return false, ""
		},
	}, emitter)

	return loop.Run(ctx, messages)
}

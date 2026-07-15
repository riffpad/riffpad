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
	b.WriteString("- Do not execute destructive commands like `rm -rf /`.\n\n")
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

func (o *Orchestrator) Run(ctx context.Context, box sandbox.Sandbox, userPrompt string, emitter EventEmitter) error {
	loop := NewLoop(LoopConfig{
		Client:       o.client,
		Model:        o.model,
		SystemPrompt: o.SystemPrompt(),
		Sandbox:      box,
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

	messages := []Message{NewUserMessage(userPrompt)}
	return loop.Run(ctx, messages)
}

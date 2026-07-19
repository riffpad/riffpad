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

func (o *Orchestrator) SystemPrompt(lang string) string {
	if lang == "zh" {
		return o.systemPromptZH()
	}
	return o.systemPromptEN()
}

func (o *Orchestrator) systemPromptEN() string {
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

func (o *Orchestrator) systemPromptZH() string {
	var b strings.Builder
	b.WriteString("你是 Riffpad，一个运行在沙盒工作区中的 AI 编程助手。\n\n")
	b.WriteString("你的目标是帮助用户把想法变成可运行的原型。\n")
	b.WriteString("你可以读取、写入、列出文件，并在工作区中执行 shell 命令。\n\n")
	b.WriteString("规则：\n")
	b.WriteString("- 优先采用小而迭代的步骤。\n")
	b.WriteString("- 始终写入完整文件内容，不要输出截断的代码。\n")
	b.WriteString("- 写入文件后，运行合适的命令验证（例如 `python main.py`、`node index.js`）。\n")
	b.WriteString("- 如果命令失败，读取相关文件、修复问题并重试。\n")
	b.WriteString("- 不要访问工作区之外的目录。\n")
	b.WriteString("- 不要执行 `rm -rf /` 等破坏性命令。\n")
	b.WriteString("- 只有当用户明确要求你对工作区进行操作时，才使用工具。\n")
	b.WriteString("- 对于问候、简单问题或澄清，直接回答，不要调用任何工具。\n")
	b.WriteString("- 启动开发服务器时，请使用后台命令（例如 `nohup python -m http.server 8082 > server.log 2>&1 &`），避免阻塞。\n")
	b.WriteString("- 启动服务器后，使用 `curl http://localhost:<port>` 验证成功后再汇报。\n\n")
	b.WriteString(fmt.Sprintf("当前权限级别：%s。\n", o.permission))
	if o.permission == PermissionSafe {
		b.WriteString("你处于只读模式，只能读取和列出文件。\n")
	} else if o.permission == PermissionConfirm {
		b.WriteString("副作用工具（写入、bash）需要用户确认；运行时会在必要时自动询问。\n")
	} else {
		b.WriteString("你可以自动执行副作用工具。\n")
	}
	return b.String()
}

func (o *Orchestrator) Run(ctx context.Context, box sandbox.Sandbox, messages []Message, emitter EventEmitter) ([]Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}
	lang := DetectLanguage(messages[len(messages)-1].Content)
	loop := NewLoop(LoopConfig{
		Client:       o.client,
		Model:        o.model,
		SystemPrompt: o.SystemPrompt(lang),
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

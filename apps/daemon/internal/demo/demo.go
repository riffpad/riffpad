// Package demo provides a scripted adapter.Session that replays a unified
// protocol event timeline (thinking, tool spinner → completed, file change,
// approval card, agent reply) so the client UI can be exercised without a
// real coding CLI or any API quota.
package demo

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

// Delay is the pause between scripted events. A var so tests can set it to 0.
var Delay = 450 * time.Millisecond

// Demo is a scripted session.
type Demo struct {
	id     string
	name   string
	cwd    string
	events chan protocol.Event
	stopCh chan struct{}
	doneCh chan struct{}

	mu              sync.Mutex
	started         bool
	approval        map[string]chan string
	nextApprovalSeq int
}

// New creates a scripted demo session.
func New(req adapter.CreateRequest) *Demo {
	return &Demo{
		id:       req.ID,
		name:     req.Name,
		cwd:      req.Cwd,
		events:   make(chan protocol.Event, 64),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		approval: make(map[string]chan string),
	}
}

func (d *Demo) ID() string                    { return d.id }
func (d *Demo) Events() <-chan protocol.Event { return d.events }
func (d *Demo) Meta() protocol.SessionStartPayload {
	return protocol.SessionStartPayload{Name: d.name, CLI: "demo", Cwd: d.cwd}
}

// Start launches the welcome timeline.
func (d *Demo) Start(_ context.Context) error {
	d.mu.Lock()
	if d.started {
		d.mu.Unlock()
		return nil
	}
	d.started = true
	d.mu.Unlock()
	go d.welcome()
	return nil
}

func (d *Demo) emit(typ string, payload any) bool {
	ev, err := protocol.NewEvent(d.id, typ, payload)
	if err != nil {
		return false
	}
	select {
	case d.events <- ev:
		return true
	case <-d.stopCh:
		return false
	}
}

func (d *Demo) sleep() bool {
	if Delay <= 0 {
		return true
	}
	select {
	case <-time.After(Delay):
		return true
	case <-d.stopCh:
		return false
	}
}

func (d *Demo) welcome() {
	defer close(d.doneCh)
	if !d.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusRunning}) {
		return
	}
	if !d.sleep() {
		return
	}
	d.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: "Hi! I'm the **Riffpad demo agent** — a scripted timeline, no real model, no API quota.\n\nTry sending me:\n- `tool` — watch a command spinner turn green\n- `approval` — trigger a phone approval card\n- `markdown` — rich formatting (code, lists, tables)"})
	if !d.sleep() {
		return
	}
	if !d.emitTool("npm test") {
		return
	}
	if !d.sleep() {
		return
	}
	d.emit(protocol.EventFileChange, protocol.FileChangePayload{
		Path: "src/auth/middleware.ts", Summary: "updated",
	})
	if !d.sleep() {
		return
	}
	decided, ok := d.requestApproval("Bash", "rm src/old.ts", "删除不再使用的文件")
	if !ok {
		return
	}
	text := "Approved — `src/old.ts` removed, tests still green."
	if decided != "approve" {
		text = "Rejected — the agent paused and will try another approach."
	}
	if !d.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: text}) {
		return
	}
	if !d.sleep() {
		return
	}
	d.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusWaitingInput})
}

// emitTool plays a Bash tool spinner → command output → completed row.
func (d *Demo) emitTool(command string) bool {
	if !d.emit(protocol.EventToolCall, protocol.ToolCallPayload{
		Tool: "Bash", Status: "started", Summary: command,
	}) {
		return false
	}
	if !d.sleep() {
		return false
	}
	exit := 0
	if !d.emit(protocol.EventCommand, protocol.CommandPayload{
		Command: command, ExitCode: &exit, Output: "42 passed · 0 failed · 1.8s",
	}) {
		return false
	}
	if !d.sleep() {
		return false
	}
	return d.emit(protocol.EventToolCall, protocol.ToolCallPayload{
		Tool: "Bash", Status: "completed", Summary: command,
	})
}

func (d *Demo) requestApproval(action, summary, detail string) (string, bool) {
	d.mu.Lock()
	d.nextApprovalSeq++
	reqID := fmt.Sprintf("demo-approval-%d", d.nextApprovalSeq)
	ch := make(chan string, 1)
	d.approval[reqID] = ch
	d.mu.Unlock()
	if !d.emit(protocol.EventApprovalReq, protocol.ApprovalRequestPayload{
		RequestID: reqID,
		Action:    action,
		Summary:   summary,
		Options:   []string{"approve", "reject"},
		Args:      map[string]any{"detail": detail},
	}) {
		return "", false
	}
	select {
	case dec := <-ch:
		d.mu.Lock()
		delete(d.approval, reqID)
		d.mu.Unlock()
		return dec, true
	case <-d.stopCh:
		return "", false
	}
}

// SendApproval resolves a pending scripted approval.
func (d *Demo) SendApproval(requestID, decision string) error {
	d.mu.Lock()
	ch, ok := d.approval[requestID]
	d.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending approval %s", requestID)
	}
	mapped := "reject"
	if decision == "approve" {
		mapped = "approve"
	}
	ch <- mapped
	return nil
}

// SendPrompt starts a scripted reply turn. Keywords switch the demo path.
func (d *Demo) SendPrompt(text string) error {
	go d.reply(text)
	return nil
}

func (d *Demo) reply(text string) {
	if !d.emit(protocol.EventUserMessage, protocol.PromptPayload{Text: text}) {
		return
	}
	if !d.sleep() {
		return
	}
	if !d.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusRunning}) {
		return
	}
	if !d.sleep() {
		return
	}
	switch {
	case strings.Contains(strings.ToLower(text), "approval"):
		decided, ok := d.requestApproval("WriteFile", "src/new-feature.ts", "创建新功能文件")
		if !ok {
			return
		}
		msg := "Approved — file written."
		if decided != "approve" {
			msg = "Rejected — agent paused."
		}
		d.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: msg})
	case strings.Contains(strings.ToLower(text), "tool"):
		if !d.emitTool("pnpm test") {
			return
		}
		if !d.sleep() {
			return
		}
		d.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: "All checks passed ✅"})
	case strings.Contains(strings.ToLower(text), "markdown") || strings.Contains(strings.ToLower(text), "code"):
		d.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: "Here's a **markdown** demo:\n\n1. First item\n2. Second item\n\n```ts\nconst ok = (x: number) => x * 2;\nconsole.log(ok(21)); // 42\n```\n\n| cli | status |\n|---|---|\n| codex | foreground |\n| claude | foreground |\n| kimi | foreground |"})
	default:
		d.emit(protocol.EventAgentMessage, protocol.AgentMessagePayload{
			Text: "You said: " + text + "\n\nThis is a scripted demo reply — send `tool`, `approval` or `markdown` to explore more UI states.",
		})
	}
	if !d.sleep() {
		return
	}
	d.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusWaitingInput})
}

// Alive reports true until Stop is called.
func (d *Demo) Alive() bool {
	select {
	case <-d.stopCh:
		return false
	default:
		return true
	}
}

// Stop ends the scripted timeline.
func (d *Demo) Stop() error {
	select {
	case <-d.stopCh:
	default:
		close(d.stopCh)
	}
	<-d.doneCh
	return nil
}

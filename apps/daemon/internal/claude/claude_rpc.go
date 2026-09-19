package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/riffpad/riffpad/packages/protocol"
)

func (c *Claude) readLoop(r io.Reader) {
	defer close(c.doneCh)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		c.handleLine(line)
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Wait()
	}
	c.mu.Lock()
	c.exited = true
	c.mu.Unlock()
	_ = c.emit(protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "process_exit"})
}

func (c *Claude) copyStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf("claude[%s] stderr: %s", c.id, scanner.Text())
	}
}

func (c *Claude) handleLine(line []byte) {
	var raw struct {
		Type      string          `json:"type"`
		Subtype   string          `json:"subtype"`
		RequestID string          `json:"request_id"`
		Message   json.RawMessage `json:"message"`
		Request   json.RawMessage `json:"request"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return
	}
	switch raw.Type {
	case "system":
		c.handleSystem(raw.Subtype, line)
	case "control_response":
		// initialize ack; nothing to do.
	case "assistant":
		c.handleAssistant(raw.Message)
	case "user":
		c.handleUser(raw.Message)
	case "control_request", "sdk_control_request":
		body := raw.Message
		if len(body) == 0 {
			body = raw.Request
		}
		c.handleControlRequest(raw.RequestID, body)
	case "result":
		// A turn finished, but the session is still alive and waiting for
		// input; "done" is reserved for real process exit (see readLoop).
		c.mu.Lock()
		c.agentActive = false
		c.mu.Unlock()
		status := protocol.StatusWaitingInput
		if raw.Subtype == "error" || raw.Subtype == "error_max_turns" {
			status = protocol.StatusError
		}
		_ = c.emit(protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: status})
	}
}

func (c *Claude) writeLine(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stdin == nil {
		return fmt.Errorf("stdin not ready")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(c.stdin, string(data)); err != nil {
		return err
	}
	return nil
}

// initControl sends the stream-json control-protocol initialize frame.
// Claude 2.1.x requires camelCase matchers/hookCallbackIds arrays.
func (c *Claude) initControl() error {
	msg := map[string]any{
		"type": "control_request",
		"request": map[string]any{
			"subtype":    "initialize",
			"request_id": "riffpad_init_1",
			"hooks": map[string]any{
				"UserPromptSubmit": []any{
					map[string]any{
						"matchers":        []any{""},
						"hookCallbackIds": []any{"hook_user_prompt"},
					},
				},
			},
			"sdk_mcp_servers": []any{},
		},
	}
	return c.writeLine(msg)
}

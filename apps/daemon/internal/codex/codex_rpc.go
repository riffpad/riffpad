package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"github.com/riffpad/riffpad/packages/protocol"
)

func (c *Codex) request(method string, params map[string]any) error {
	c.mu.Lock()
	id := c.nextRequestID
	c.nextRequestID++
	c.mu.Unlock()
	return c.writeJSON(map[string]any{"method": method, "id": id, "params": params})
}

// requestSync sends a JSON-RPC request and waits for its response.
func (c *Codex) requestSync(method string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	c.mu.Lock()
	id := c.nextRequestID
	c.nextRequestID++
	ch := make(chan json.RawMessage, 1)
	c.pendingRes[id] = ch
	c.mu.Unlock()
	if err := c.writeJSON(map[string]any{"method": method, "id": id, "params": params}); err != nil {
		c.mu.Lock()
		delete(c.pendingRes, id)
		c.mu.Unlock()
		return nil, err
	}
	select {
	case res := <-ch:
		var e struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(res, &e) == nil && e.Message != "" {
			return nil, fmt.Errorf("codex %s: %s", method, e.Message)
		}
		return res, nil
	case <-time.After(timeout):
		c.mu.Lock()
		delete(c.pendingRes, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("request %s timed out", method)
	case <-c.stopCh:
		return nil, fmt.Errorf("session stopped")
	}
}

func (c *Codex) writeJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if c.sendFn != nil {
		return c.sendFn(data)
	}
	if c.conn == nil {
		return fmt.Errorf("app-server not connected")
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Codex) readLoop(conn *websocket.Conn) {
	defer close(c.doneCh)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		c.handleLine(data)
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Wait()
	}
	c.flushMessages()
	c.mu.Lock()
	c.exited = true
	c.mu.Unlock()
	_ = c.emit(protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "process_exit"})
}

func (c *Codex) copyStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf("codex[%s] stderr: %s", c.id, scanner.Text())
	}
}

func (c *Codex) handleLine(line []byte) {
	var msg struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(line, &msg); err != nil {
		return
	}
	if len(msg.Error) > 0 && string(msg.Error) != "null" {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(msg.Error, &e)
		var rid int
		if json.Unmarshal(msg.ID, &rid) == nil {
			c.mu.Lock()
			ch, ok := c.pendingRes[rid]
			if ok {
				delete(c.pendingRes, rid)
			}
			c.mu.Unlock()
			if ok {
				ch <- json.RawMessage(msg.Error)
				return
			}
		}
		c.mu.Lock()
		c.initError = fmt.Errorf("codex app-server error: %s", e.Message)
		c.mu.Unlock()
		c.closeReady()
		_ = c.emit(protocol.EventNotify, protocol.NotifyPayload{Level: "error", Message: e.Message})
		return
	}
	if msg.Method != "" && len(msg.ID) > 0 && string(msg.ID) != "null" {
		c.handleServerRequest(msg.Method, msg.ID, msg.Params)
		return
	}
	if msg.Method != "" {
		c.handleNotification(msg.Method, msg.Params)
		return
	}
	c.handleResponse(msg.ID, msg.Result)
}

func (c *Codex) closeReady() {
	select {
	case <-c.ready:
	default:
		close(c.ready)
	}
}

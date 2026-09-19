package kimi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/riffpad/riffpad/packages/protocol"
)

func (k *Kimi) request(method string, params map[string]any) error {
	k.mu.Lock()
	id := k.nextRequestID
	k.nextRequestID++
	k.mu.Unlock()
	return k.writeJSON(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
}

func (k *Kimi) writeJSON(v any) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.stdin == nil {
		return fmt.Errorf("stdin not ready")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(k.stdin, string(data)); err != nil {
		return err
	}
	return nil
}

func (k *Kimi) readLoop(r io.Reader) {
	defer close(k.doneCh)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		k.handleLine(line)
	}
	if k.cmd != nil && k.cmd.Process != nil {
		_ = k.cmd.Wait()
	}
	k.flushMessage()
	k.mu.Lock()
	k.exited = true
	k.mu.Unlock()
	_ = k.emit(protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "process_exit"})
}

func (k *Kimi) copyStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf("kimi[%s] stderr: %s", k.id, scanner.Text())
	}
}

// handleLine processes one JSON-RPC line from the ACP server.
func (k *Kimi) handleLine(line []byte) {
	var msg struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(line, &msg); err != nil {
		return
	}
	if len(msg.Error) > 0 && string(msg.Error) != "null" {
		var e struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(msg.Error, &e)
		k.mu.Lock()
		k.initError = fmt.Errorf("kimi ACP error: %s", e.Message)
		k.mu.Unlock()
		k.closeReady()
		_ = k.emit(protocol.EventNotify, protocol.NotifyPayload{Level: "error", Message: e.Message})
		return
	}
	if msg.Method != "" {
		k.handleNotification(msg.Method, msg.ID, msg.Params)
		return
	}
	k.handleResponse(msg.ID, msg.Result)
}

func (k *Kimi) closeReady() {
	select {
	case <-k.readyCh:
	default:
		close(k.readyCh)
	}
}

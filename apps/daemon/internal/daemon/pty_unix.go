//go:build !windows

package daemon

import (
	"encoding/base64"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
)

// ptyUpgrader accepts local-only console connections. The localAuth
// middleware has already validated the token; loopback enforcement happens
// at the HTTP layer, so the WS itself trusts the caller.
var ptyUpgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

// handleSessionPTY bridges one local CLI console to a session's interactive
// PTY. Frames: client → {"in": base64} / {"resize": {cols, rows}};
// daemon → {"out": base64} / {"exit": 0}. Only one console per session: a
// new attach replaces (and closes) the previous one, so re-running
// `riffpad run claude` re-attaches cleanly.
func (s *Server) handleSessionPTY(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	ts, ok := sess.getAdapter().(adapter.TerminalSession)
	if !ok {
		writeError(w, http.StatusBadRequest, "session does not expose a PTY")
		return
	}
	term, err := ts.AttachPTY()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	conn, err := ptyUpgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = term.Close()
		return
	}

	// Single-console policy: close any previous console for this session.
	s.mu.Lock()
	if prev, ok := s.ptys[id]; ok {
		_ = prev.Close()
	}
	s.ptys[id] = conn
	s.mu.Unlock()
	defer func() {
		_ = term.Close()
		s.mu.Lock()
		if s.ptys[id] == conn {
			delete(s.ptys, id)
		}
		s.mu.Unlock()
	}()

	var writeMu sync.Mutex
	writeJSONFrame := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(v)
	}

	// PTY → WS. Runs until the process exits (master EOF), then tells the
	// console and tears the connection down.
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := term.Read(buf)
			if n > 0 {
				_ = writeJSONFrame(map[string]any{
					"out": base64.StdEncoding.EncodeToString(buf[:n]),
				})
			}
			if err != nil {
				break
			}
		}
		_ = writeJSONFrame(map[string]any{"exit": 0})
		_ = conn.Close()
	}()

	// WS → PTY: input bytes and resize events.
	for {
		var msg struct {
			In     string `json:"in"`
			Resize *struct {
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			} `json:"resize"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		if msg.In != "" {
			b, err := base64.StdEncoding.DecodeString(msg.In)
			if err == nil {
				_, _ = term.Write(b)
			}
		}
		if msg.Resize != nil {
			_ = term.Resize(msg.Resize.Cols, msg.Resize.Rows)
		}
	}
}

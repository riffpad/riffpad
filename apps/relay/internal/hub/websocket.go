package hub

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket heartbeat parameters, shared by host and viewer connections.
// Pings go out every wsPingPeriod; if no pong (or any other frame) arrives
// within wsPongWait the connection is treated as half-open and torn down.
// Atomics (not constants) so tests can shrink them without data races.
var (
	wsWriteWait  atomic.Int64
	wsPingPeriod atomic.Int64
	wsPongWait   atomic.Int64
)

func init() {
	wsWriteWait.Store(int64(10 * time.Second))
	wsPingPeriod.Store(int64(30 * time.Second))
	wsPongWait.Store(int64(75 * time.Second))
}

func wsWriteDeadline() time.Time    { return time.Now().Add(time.Duration(wsWriteWait.Load())) }
func wsPingInterval() time.Duration { return time.Duration(wsPingPeriod.Load()) }
func wsPongTimeout() time.Duration  { return time.Duration(wsPongWait.Load()) }

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func (h *hostConn) writeLoop() {
	ticker := time.NewTicker(wsPingInterval())
	defer ticker.Stop()
	for {
		select {
		case data := <-h.send:
			_ = h.conn.SetWriteDeadline(wsWriteDeadline())
			h.writeMu.Lock()
			err := h.conn.WriteMessage(websocket.TextMessage, data)
			h.writeMu.Unlock()
			if err != nil {
				return
			}
		case <-ticker.C:
			if err := h.conn.WriteControl(websocket.PingMessage, nil, wsWriteDeadline()); err != nil {
				return
			}
		case <-h.done:
			return
		}
	}
}

func (v *viewerConn) writeLoop() {
	ticker := time.NewTicker(wsPingInterval())
	defer ticker.Stop()
	for {
		select {
		case data := <-v.send:
			_ = v.conn.SetWriteDeadline(wsWriteDeadline())
			if err := v.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			if err := v.conn.WriteControl(websocket.PingMessage, nil, wsWriteDeadline()); err != nil {
				return
			}
		case <-v.done:
			return
		}
	}
}

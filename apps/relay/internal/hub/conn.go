package hub

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type hostConn struct {
	id   string
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
	once sync.Once
	// writeMu serializes direct writes with writeLoop so a final control
	// frame (superseded) can be delivered synchronously before the
	// connection is closed.
	writeMu sync.Mutex
}

func (h *hostConn) closeDone() {
	h.once.Do(func() { close(h.done) })
}

// writeFrame writes one frame directly on the connection, serialized with
// writeLoop. Used for the final superseded notice, which must go out before
// the connection is closed (queueing it on send would race with done).
func (h *hostConn) writeFrame(fr hostFrame) {
	data, err := json.Marshal(fr)
	if err != nil {
		return
	}
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	_ = h.conn.SetWriteDeadline(wsWriteDeadline())
	_ = h.conn.WriteMessage(websocket.TextMessage, data)
}

type viewerConn struct {
	id        string
	deviceID  string
	sessionID string
	host      *hostConn
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	once      sync.Once
}

func (v *viewerConn) closeDone() {
	v.once.Do(func() { close(v.done) })
}

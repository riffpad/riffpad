package hub

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

func (h *Hub) handleViewerWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sid := q.Get("session")
	deviceID := q.Get("device")
	eph := q.Get("eph")
	token := q.Get("token")
	if sid == "" || deviceID == "" || eph == "" || token == "" {
		writeError(w, http.StatusBadRequest, "session, device, eph and token are required")
		return
	}
	u, err := h.store.UserByToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	dev, err := h.store.GetDevice(deviceID)
	if err != nil || dev.OwnerID != u.ID {
		writeError(w, http.StatusUnauthorized, "device not paired for this user")
		return
	}
	host := func() *hostConn {
		h.mu.Lock()
		defer h.mu.Unlock()
		hostID, ok := h.sessionHosts[sid]
		if !ok {
			return nil
		}
		hostRec, err := h.store.GetHost(hostID)
		if err != nil || hostRec.OwnerID != u.ID {
			return nil
		}
		return h.hosts[hostID]
	}()
	if host == nil {
		writeError(w, http.StatusNotFound, "session offline")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	v := &viewerConn{
		id: protocol.NewID(), deviceID: deviceID, sessionID: sid, host: host,
		conn: conn, send: make(chan []byte, 256), done: make(chan struct{}),
	}
	h.mu.Lock()
	h.viewers[v.id] = v
	h.mu.Unlock()
	h.hostSend(host, hostFrame{
		Kind: "join", ViewerID: v.id, SessionID: sid, DeviceID: deviceID,
		Curve: dev.Curve, Pub: dev.PublicKey, Eph: eph,
	})
	go v.writeLoop()
	go h.viewerReadLoop(v)
}

func (h *Hub) viewerReadLoop(v *viewerConn) {
	defer func() {
		h.mu.Lock()
		if h.viewers[v.id] == v {
			delete(h.viewers, v.id)
		}
		h.mu.Unlock()
		h.hostSend(v.host, hostFrame{Kind: "leave", ViewerID: v.id})
		v.closeDone()
		_ = v.conn.Close()
	}()
	v.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	v.conn.SetPongHandler(func(string) error {
		return v.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	})
	for {
		_, data, err := v.conn.ReadMessage()
		if err != nil {
			return
		}
		h.hostSend(v.host, hostFrame{Kind: "viewer", ViewerID: v.id, Data: base64.RawStdEncoding.EncodeToString(data)})
	}
}

// sendToViewer queues one payload for a viewer. The relay cannot inspect the
// encrypted contents, so an overflow may be dropping a critical event: close
// the connection instead of silently dropping, forcing the client to
// reconnect and replay history (#173).
func (h *Hub) sendToViewer(v *viewerConn, payload []byte) {
	select {
	case v.send <- payload:
	default:
		h.dropViewer(v, "send buffer full")
	}
}

// dropViewer removes a viewer and closes its connection. The viewer's read
// loop notices the close and sends the host a "leave" notice.
func (h *Hub) dropViewer(v *viewerConn, reason string) {
	h.mu.Lock()
	if h.viewers[v.id] != v {
		h.mu.Unlock()
		return
	}
	delete(h.viewers, v.id)
	h.mu.Unlock()
	h.log.Printf("viewer dropped (%s) viewer=%s session=%s host=%s", reason, v.id, v.sessionID, v.host.id)
	v.closeDone()
	if v.conn != nil {
		_ = v.conn.Close()
	}
}

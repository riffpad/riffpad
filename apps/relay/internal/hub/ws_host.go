package hub

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

type hostFrame struct {
	Kind string `json:"kind"`

	Sessions []SessionMeta `json:"sessions,omitempty"`

	ViewerID  string `json:"viewerId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	DeviceID  string `json:"deviceId,omitempty"`
	Curve     string `json:"curve,omitempty"`
	Pub       string `json:"pub,omitempty"`
	Eph       string `json:"eph,omitempty"`
	Data      string `json:"data,omitempty"`
}

func (h *Hub) handleHostWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	hostID := q.Get("hostId")
	token := q.Get("token")
	rec, err := h.store.GetHost(hostID)
	if err != nil || token != rec.Secret {
		writeError(w, http.StatusUnauthorized, "bad host credentials")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	host := &hostConn{id: hostID, conn: conn, send: make(chan []byte, 256), done: make(chan struct{})}
	h.mu.Lock()
	if old, ok := h.hosts[hostID]; ok {
		// Tell the replaced connection why it is being dropped: two daemons
		// sharing the same host credentials would otherwise kick each other
		// in an endless reconnect loop (#169).
		old.writeFrame(hostFrame{Kind: protocol.RelayFrameSuperseded})
		old.closeDone()
		_ = old.conn.Close()
	}
	h.hosts[hostID] = host
	h.mu.Unlock()
	h.log.Printf("host connected id=%s", hostID)
	go host.writeLoop()
	go h.hostReadLoop(host)
}

func (h *Hub) hostReadLoop(host *hostConn) {
	defer func() {
		h.removeHost(host)
		host.closeDone()
		_ = host.conn.Close()
	}()
	host.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	host.conn.SetPongHandler(func(string) error {
		return host.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	})
	for {
		_, data, err := host.conn.ReadMessage()
		if err != nil {
			return
		}
		var fr hostFrame
		if json.Unmarshal(data, &fr) != nil {
			continue
		}
		switch fr.Kind {
		case "sessions":
			h.mu.Lock()
			if h.hosts[host.id] == host {
				// Replace only this host's entries: other hosts of the same
				// account (desktop + laptop) keep their sessions (#169). The
				// current-connection check also rejects a stale announce from
				// a superseded connection still draining its read loop.
				for sid, hid := range h.sessionHosts {
					if hid == host.id {
						delete(h.sessionHosts, sid)
						delete(h.sessions, sid)
					}
				}
				for _, s := range fr.Sessions {
					s.HostID = host.id
					if s.LastSeenAt.IsZero() {
						// Older daemons announce without a timestamp; these
						// sessions are live right now, so stamp them here
						// (mirrors Store.UpsertSessions). Otherwise the
						// in-memory snapshot would report Go's zero time and
						// clients would show "739835d ago".
						s.LastSeenAt = time.Now()
					}
					h.sessions[s.ID] = s
					h.sessionHosts[s.ID] = host.id
				}
			}
			h.mu.Unlock()
			_ = h.store.UpsertSessions(host.id, fr.Sessions)
		case "viewer":
			h.mu.Lock()
			v, ok := h.viewers[fr.ViewerID]
			h.mu.Unlock()
			if !ok {
				continue
			}
			payload, err := base64.RawStdEncoding.DecodeString(fr.Data)
			if err != nil {
				continue
			}
			h.sendToViewer(v, payload)
		case protocol.RelayFrameKick:
			// The daemon dropped this viewer (e.g. send-buffer overflow on a
			// critical event, #173): close the browser connection so the
			// client reconnects and replays instead of hanging silently.
			h.mu.Lock()
			v, ok := h.viewers[fr.ViewerID]
			h.mu.Unlock()
			if ok {
				h.dropViewer(v, "kicked by host")
			}
		}
	}
}

func (h *Hub) removeHost(host *hostConn) {
	h.mu.Lock()
	// Only the connection currently registered under this host id may tear
	// down its sessions: after a reconnect the old connection's deferred
	// cleanup runs after the new one has already re-announced, and must not
	// wipe those fresh entries (#169).
	current := h.hosts[host.id] == host
	if current {
		delete(h.hosts, host.id)
		for sid, hid := range h.sessionHosts {
			if hid == host.id {
				delete(h.sessionHosts, sid)
				delete(h.sessions, sid)
			}
		}
	}
	for id, v := range h.viewers {
		if v.host == host {
			delete(h.viewers, id)
			v.closeDone()
			_ = v.conn.Close()
		}
	}
	h.mu.Unlock()
	if current {
		_ = h.store.MarkHostSessionsOffline(host.id)
	}
	h.log.Printf("host disconnected id=%s", host.id)
}

func (h *Hub) hostSend(host *hostConn, fr hostFrame) {
	data, err := json.Marshal(fr)
	if err != nil {
		return
	}
	select {
	case host.send <- data:
	default:
		// Same reasoning as sendToViewer: with no visibility into the payload,
		// any drop may be critical. Closing forces the daemon to reconnect and
		// re-announce, and its viewers to reconnect and replay (#173).
		h.log.Printf("host send buffer full, closing connection host=%s kind=%s", host.id, fr.Kind)
		host.closeDone()
		if host.conn != nil {
			_ = host.conn.Close()
		}
	}
}

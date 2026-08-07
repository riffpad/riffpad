package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

const (
	// historyReplayLimit is how many recent events a viewer gets on connect.
	historyReplayLimit = 100
	// historyQueryLimit caps how many older events a page request may fetch.
	historyQueryLimit = 200
)

// historySlice returns the events strictly before the event with the given
// id, oldest first, at most limit entries. Events are already in
// chronological order.
func historySlice(events []protocol.Event, before string, limit int) []protocol.Event {
	if before == "" || limit <= 0 {
		return nil
	}
	idx := -1
	for i, ev := range events {
		if ev.ID == before {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return nil
	}
	start := idx - limit
	if start < 0 {
		start = 0
	}
	return events[start:idx]
}

type wsTransport struct {
	conn *websocket.Conn
}

func (t *wsTransport) Send(data []byte) error {
	return t.conn.WriteMessage(websocket.TextMessage, data)
}

func (t *wsTransport) Recv() ([]byte, error) {
	_, data, err := t.conn.ReadMessage()
	return data, err
}

func (t *wsTransport) Close() error {
	return t.conn.Close()
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	deviceID := q.Get("device")
	sid := q.Get("session")
	ephRaw := q.Get("eph")
	if deviceID == "" || sid == "" || ephRaw == "" {
		writeError(w, http.StatusBadRequest, "device, session and eph are required")
		return
	}
	s.mu.Lock()
	dev, devOK := s.devices[deviceID]
	s.mu.Unlock()
	if !devOK {
		writeError(w, http.StatusUnauthorized, "device not paired")
		return
	}
	ephPub, err := protocol.DecodeKey(ephRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ephemeral key")
		return
	}
	devPub, err := dev.PublicKeyBytes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "device key unavailable")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	if err := s.attachViewer(&wsTransport{conn: conn}, deviceID, sid, ephPub, dev.Curve, devPub); err != nil {
		_ = conn.Close()
		writeError(w, http.StatusNotFound, err.Error())
	}
}

// attachViewer performs the E2EE handshake for a viewer (local or relay) and
// starts event streaming. devicePubKey is the viewer's paired identity key.
func (s *Server) attachViewer(tr viewerTransport, deviceID, sid string, ephPub []byte, curve protocol.Curve, devicePubKey []byte) error {
	s.mu.Lock()
	sess, sessOK := s.sessions[sid]
	s.mu.Unlock()
	if !sessOK {
		return fmt.Errorf("session not found")
	}
	identity, err := s.keys.Identity(curve)
	if err != nil {
		return fmt.Errorf("server identity unavailable: %w", err)
	}
	deviceSecret, err := protocol.NewDeviceSecret(identity, devicePubKey)
	if err != nil {
		return fmt.Errorf("device secret failed: %w", err)
	}
	serverEph, err := protocol.GenerateKeyPair(curve)
	if err != nil {
		return fmt.Errorf("ephemeral key failed: %w", err)
	}
	ephSecret, err := protocol.ECDH(serverEph, ephPub)
	if err != nil {
		return fmt.Errorf("ephemeral exchange failed: %w", err)
	}
	key, err := protocol.DeriveSessionKey(deviceSecret, ephSecret, sid)
	if err != nil {
		return fmt.Errorf("session key derivation failed: %w", err)
	}
	c := &client{
		deviceID:  deviceID,
		session:   sess,
		key:       key,
		transport: tr,
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		log:       s.log,
	}
	s.log.Printf("viewer connect device=%s session=%s curve=%s", deviceID, sid, curve)
	sess.addClient(c)
	hello := protocol.Hello{
		V:            1,
		Kind:         "hello",
		SessionID:    sid,
		ServerEphPub: protocol.EncodeKey(serverEph.PublicKey),
	}
	helloData, _ := json.Marshal(hello)
	c.sendRaw(helloData)
	replay := sess.snapshotLast(historyReplayLimit)
	s.log.Printf("hello queued device=%s session=%s replay=%d", deviceID, sid, len(replay))
	for _, ev := range replay {
		c.sendEvent(ev)
	}
	go c.writeLoop()
	go c.readLoop(s)
	return nil
}

func (c *client) readLoop(s *Server) {
	defer func() {
		close(c.done)
		c.session.removeClient(c)
		_ = c.transport.Close()
	}()
	for {
		data, err := c.transport.Recv()
		if err != nil {
			return
		}
		var ctrl struct {
			Kind   string `json:"kind"`
			Before string `json:"before"`
			Limit  int    `json:"limit"`
		}
		if json.Unmarshal(data, &ctrl) == nil && ctrl.Kind == "history_query" {
			s.handleHistoryQuery(c, ctrl.Before, ctrl.Limit)
			continue
		}
		var env protocol.Envelope
		if json.Unmarshal(data, &env) != nil {
			continue
		}
		plain, err := env.Open(c.key)
		if err != nil {
			continue
		}
		var ev protocol.Event
		if json.Unmarshal(plain, &ev) != nil {
			continue
		}
		s.log.Printf("client event session=%s type=%s device=%s", c.session.id, ev.Type, c.deviceID)
		s.dispatch(c.session, ev)
	}
}

// handleHistoryQuery sends older events (before the given event id) to the
// viewer: a plaintext "history" marker, the encrypted events, then a
// "history_done" marker.
func (s *Server) handleHistoryQuery(c *client, before string, limit int) {
	if limit <= 0 || limit > historyQueryLimit {
		limit = historyQueryLimit
	}
	key, err := s.sessionEncKey()
	if err != nil {
		return
	}
	events, err := loadSessionEvents(s.dataDir, c.session.id, key)
	if err != nil {
		return
	}
	slice := historySlice(events, before, limit)
	start, _ := json.Marshal(map[string]any{"kind": "history", "count": len(slice)})
	c.sendRaw(start)
	for _, ev := range slice {
		c.sendEvent(ev)
	}
	done, _ := json.Marshal(map[string]any{"kind": "history_done", "count": len(slice)})
	c.sendRaw(done)
}

func (s *Server) dispatch(sess *session, ev protocol.Event) {
	switch ev.Type {
	case protocol.EventApprovalResp:
		var p protocol.ApprovalResponsePayload
		if err := ev.DecodePayload(&p); err != nil {
			return
		}
		decision := "deny"
		if p.Decision == "approve" {
			decision = "approve"
		}
		s.mu.Lock()
		if ch, ok := s.pendingHooks[p.RequestID]; ok {
			delete(s.pendingHooks, p.RequestID)
			s.mu.Unlock()
			ch <- decision
			return
		}
		s.mu.Unlock()
		_ = sess.adapter.SendApproval(p.RequestID, decision)
	case protocol.EventPrompt:
		var p protocol.PromptPayload
		if err := ev.DecodePayload(&p); err != nil {
			return
		}
		if err := sess.adapter.SendPrompt(p.Text); err != nil {
			s.notifySession(sess, "error", "指令发送失败："+err.Error())
		}
	case protocol.EventControl:
		var p protocol.ControlPayload
		if err := ev.DecodePayload(&p); err != nil {
			return
		}
		if p.Action == "stop" {
			if err := sess.adapter.Stop(); err != nil {
				s.notifySession(sess, "error", "停止失败："+err.Error())
			}
		}
	}
}

func (s *Server) notifySession(sess *session, level, message string) {
	ev, err := protocol.NewEvent(sess.id, protocol.EventNotify, protocol.NotifyPayload{Level: level, Message: message})
	if err != nil {
		return
	}
	s.pumpEvent(sess, ev)
}

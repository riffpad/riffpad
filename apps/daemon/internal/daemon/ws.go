package daemon

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
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
	sess, sessOK := s.sessions[sid]
	s.mu.Unlock()
	if !devOK {
		writeError(w, http.StatusUnauthorized, "device not paired")
		return
	}
	if !sessOK {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	ephPub, err := protocol.DecodeKey(ephRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ephemeral key")
		return
	}
	identity, err := s.keys.Identity(dev.Curve)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server identity unavailable")
		return
	}
	devPub, err := dev.PublicKeyBytes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "device key unavailable")
		return
	}
	deviceSecret, err := protocol.NewDeviceSecret(identity, devPub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "device secret failed")
		return
	}
	serverEph, err := protocol.GenerateKeyPair(dev.Curve)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ephemeral key failed")
		return
	}
	ephSecret, err := protocol.ECDH(serverEph, ephPub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ephemeral exchange failed")
		return
	}
	key, err := protocol.DeriveSessionKey(deviceSecret, ephSecret, sid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session key derivation failed")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{
		deviceID: deviceID,
		session:  sess,
		key:      key,
		conn:     conn,
		send:     make(chan []byte, 256),
		done:     make(chan struct{}),
		log:      s.log,
	}
	s.log.Printf("ws connect device=%s session=%s curve=%s", deviceID, sid, dev.Curve)
	sess.addClient(c)
	hello := protocol.Hello{
		V:            1,
		Kind:         "hello",
		SessionID:    sid,
		ServerEphPub: protocol.EncodeKey(serverEph.PublicKey),
	}
	helloData, _ := json.Marshal(hello)
	c.sendRaw(helloData)
	s.log.Printf("ws hello queued device=%s session=%s replay=%d", deviceID, sid, len(sess.snapshot()))
	for _, ev := range sess.snapshot() {
		c.sendEvent(ev)
	}
	go c.writeLoop()
	go c.readLoop(s)
}

func (c *client) readLoop(s *Server) {
	defer func() {
		close(c.done)
		c.session.removeClient(c)
		_ = c.conn.Close()
	}()
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
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

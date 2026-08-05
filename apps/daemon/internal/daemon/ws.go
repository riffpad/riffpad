package daemon

import (
	"encoding/json"
	"net/http"
	"time"

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

func (s *Server) handleHookNotification(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("session")
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid hook body")
		return
	}
	message := ""
	level := "info"
	if n, ok := body["notification"].(map[string]any); ok {
		if m, ok := n["message"].(string); ok {
			message = m
		}
		if t, ok := n["type"].(string); ok {
			switch t {
			case "permission_prompt", "idle_prompt", "agent_needs_input":
				level = "waiting"
			case "agent_completed":
				level = "completed"
			case "auth_success":
				level = "info"
			}
		}
	}
	if message == "" {
		if m, ok := body["message"].(string); ok {
			message = m
		}
	}
	sess := s.getSession(sid)
	if sess != nil && message != "" {
		ev, err := protocol.NewEvent(sid, protocol.EventNotify, protocol.NotifyPayload{Level: level, Message: message})
		if err == nil {
			s.pumpEvent(sess, ev)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleHookPermission(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("session")
	sess := s.getSession(sid)
	if sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	var body struct {
		HookEventName string         `json:"hookEventName"`
		ToolUseID     string         `json:"tool_use_id"`
		ToolUse       map[string]any `json:"tool_use"`
		Message       string         `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid hook body")
		return
	}
	action := "tool"
	summary := body.Message
	if name, ok := body.ToolUse["name"].(string); ok {
		action = name
	}
	if input, ok := body.ToolUse["input"].(map[string]any); ok {
		if cmd, ok := input["command"].(string); ok && summary == "" {
			summary = cmd
		}
		if p, ok := input["file_path"].(string); ok && summary == "" {
			summary = "file: " + p
		}
	}
	reqID := protocol.NewID()
	ch := make(chan string, 1)
	s.mu.Lock()
	s.pendingHooks[reqID] = ch
	s.mu.Unlock()
	ev, err := protocol.NewEvent(sid, protocol.EventApprovalReq, protocol.ApprovalRequestPayload{
		RequestID: reqID,
		Action:    action,
		Summary:   summary,
		Options:   []string{"approve", "reject"},
	})
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	var decision string
	select {
	case decision = <-ch:
	case <-time.After(10 * time.Minute):
		decision = "deny"
	}
	if decision == "approve" {
		decision = "allow"
	} else {
		decision = "deny"
	}
	writeJSON(w, http.StatusOK, map[string]any{"permissionDecision": decision})
}

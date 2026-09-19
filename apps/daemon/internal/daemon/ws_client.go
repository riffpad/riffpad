package daemon

import (
	"encoding/json"

	"github.com/riffpad/riffpad/packages/protocol"
)

func (c *client) readLoop(s *Server) {
	defer func() {
		c.closeDone()
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
		s.dispatch(c, ev)
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

func (s *Server) dispatch(c *client, ev protocol.Event) {
	sess := c.session
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
			s.broadcastApprovalResolved(sess, p.RequestID, p.Decision, c.deviceID)
			return
		}
		s.mu.Unlock()
		if err := sess.getAdapter().SendApproval(p.RequestID, decision); err != nil {
			// The request is unknown to the daemon (hook timed out or already
			// handled): ack the sending viewer so it can correct its UI instead
			// of leaving a "已批准" card that never took effect.
			s.log.Printf("approval expired session=%s req=%s: %v", sess.id, p.RequestID, err)
			note, nerr := protocol.NewEvent(sess.id, protocol.EventNotify, protocol.NotifyPayload{
				Level:     "error",
				Message:   "审批已过期或已被处理，本次操作未生效",
				RequestID: p.RequestID,
			})
			if nerr == nil {
				c.sendEvent(note)
			}
			return
		}
		s.broadcastApprovalResolved(sess, p.RequestID, p.Decision, c.deviceID)
	case protocol.EventPrompt:
		var p protocol.PromptPayload
		if err := ev.DecodePayload(&p); err != nil {
			return
		}
		if err := sess.getAdapter().SendPrompt(p.Text); err != nil {
			s.notifySession(sess, "error", "指令发送失败："+err.Error())
		}
	case protocol.EventControl:
		var p protocol.ControlPayload
		if err := ev.DecodePayload(&p); err != nil {
			return
		}
		if p.Action == "stop" {
			if err := sess.getAdapter().Stop(); err != nil {
				s.notifySession(sess, "error", "停止失败："+err.Error())
			}
		}
	}
}

// broadcastApprovalResolved records and fans out an approval_resolved event so
// every viewer (and future history replays) sees the card as settled (#171).
// deviceID is the viewer that sent the decision, empty for daemon-side
// resolutions such as approval timeouts.
func (s *Server) broadcastApprovalResolved(sess *session, requestID, decision, deviceID string) {
	if decision != "approve" {
		decision = "reject"
	}
	ev, err := protocol.NewEvent(sess.id, protocol.EventApprovalResolved, protocol.ApprovalResolvedPayload{
		RequestID: requestID,
		Decision:  decision,
		DeviceID:  deviceID,
	})
	if err != nil {
		return
	}
	s.pumpEvent(sess, ev)
}

func (s *Server) notifySession(sess *session, level, message string) {
	ev, err := protocol.NewEvent(sess.id, protocol.EventNotify, protocol.NotifyPayload{Level: level, Message: message})
	if err != nil {
		return
	}
	s.pumpEvent(sess, ev)
}

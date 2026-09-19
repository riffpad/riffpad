package daemon

import (
	"github.com/riffpad/riffpad/packages/protocol"
	"net/http"
	"strings"
)

// handleKillswitch stops every agent session, clears all paired devices, and
// asks the relay to revoke cloud devices / disconnect viewers.
func (s *Server) handleKillswitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	s.mu.Lock()
	sessions := make([]*session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.devices = map[string]Device{}
	s.mu.Unlock()
	_ = s.saveDevices()
	for _, sess := range sessions {
		sess.mu.Lock()
		for c := range sess.clients {
			_ = c.transport.Close()
		}
		sess.mu.Unlock()
		s.stopSession(sess.id)
	}
	if s.rc != nil {
		_ = s.relayKillswitch()
	}
	s.log.Printf("killswitch: stopped %d sessions, revoked all devices", len(sessions))
	writeJSON(w, http.StatusOK, map[string]any{"killed": true, "sessions": len(sessions)})
}

func (s *Server) relayKillswitch() error {
	httpURL := s.cfg.RelayURL
	httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
	httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")
	body := strings.NewReader("{}")
	req, err := http.NewRequest(http.MethodPost,
		strings.TrimSuffix(httpURL, "/")+"/api/hosts/"+s.rc.hostID+"/killswitch", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.RelayToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.RelayToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *Server) handleRelayJoin(ji RelayJoin) {
	ephPub, err := protocol.DecodeKey(ji.Eph)
	if err != nil {
		s.log.Printf("relay join invalid eph session=%s", ji.SessionID)
		return
	}
	devPub, err := protocol.DecodeKey(ji.Pub)
	if err != nil {
		s.log.Printf("relay join invalid pub session=%s", ji.SessionID)
		return
	}
	tr := s.rc.viewerTransport(ji.ViewerID)
	if err := s.attachViewer(tr, ji.DeviceID, ji.SessionID, ephPub, ji.Curve, devPub); err != nil {
		s.log.Printf("relay join rejected session=%s device=%s: %v", ji.SessionID, ji.DeviceID, err)
	}
}

func (s *Server) announceSessions() {
	if s.rc == nil {
		return
	}
	s.mu.Lock()
	list := make([]RelaySession, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sess.mu.Lock()
		if sess.ended {
			sess.mu.Unlock()
			continue
		}
		lastSeen := sess.lastSeen
		status := sess.status
		sess.mu.Unlock()
		list = append(list, RelaySession{
			ID: sess.id, Name: sess.meta.Name, CLI: sess.meta.CLI,
			Cwd: sess.meta.Cwd, Status: status, LastSeenAt: lastSeen,
		})
	}
	s.mu.Unlock()
	s.rc.announce(list)
}

package daemon

import (
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

func (s *Server) sweepLoop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.sweepDone:
			return
		case <-t.C:
			s.sweepOnce()
		}
	}
}

func (s *Server) sweepOnce() {
	s.mu.Lock()
	sessions := make([]*session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.mu.Unlock()
	for _, sess := range sessions {
		sess.mu.Lock()
		leaseExpired := sess.lease && time.Since(sess.lastHB) > 20*time.Second
		alreadyEnded := sess.ended
		// Grace period (#170): after a system sleep the sweep ticker and the
		// TUI heartbeat come due at the same time and the sweep can win the
		// race, killing a session whose TUI is still open. Require the lease
		// to stay expired across two consecutive sweeps before closing.
		grace := leaseExpired && !alreadyEnded && !sess.leaseMissed
		if grace {
			sess.leaseMissed = true
		}
		status := sess.status
		sess.mu.Unlock()
		if grace {
			s.log.Printf("session %s lease expired; granting one sweep period before closing", sess.id)
			continue
		}
		if leaseExpired && !alreadyEnded {
			s.log.Printf("session %s lease expired (no local TUI heartbeat); closing", sess.id)
			s.stopSession(sess.id)
			continue
		}
		if status != protocol.StatusRunning {
			continue
		}
		if _, isAttach := sess.getAdapter().(*attachAdapter); isAttach {
			// attachAdapter.Alive is always true (the process lives in the
			// user's tmux), so a `kill -9` on claude — where the SessionEnd
			// hook never fires — would otherwise leave the session "running"
			// forever. Hook activity is the only liveness signal we have; if
			// none arrived within attachIdleTimeout, mark the session ended.
			// False positives (user left claude idle) are harmless: the next
			// hook revives the session in attachSession (#170).
			sess.mu.Lock()
			idleFor := time.Since(sess.lastSeen)
			sess.mu.Unlock()
			if idleFor > attachIdleTimeout {
				ev, _ := protocol.NewEvent(sess.id, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "no_activity"})
				s.pumpEvent(sess, ev)
				s.log.Printf("session %s marked ended (no hook activity for %s)", sess.id, idleFor.Round(time.Second))
				s.announceSessions()
			}
			continue
		}
		if !sess.getAdapter().Alive() {
			ev, _ := protocol.NewEvent(sess.id, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "process_exit"})
			s.pumpEvent(sess, ev)
			s.log.Printf("session %s marked ended (process gone)", sess.id)
			s.announceSessions()
		}
	}
}

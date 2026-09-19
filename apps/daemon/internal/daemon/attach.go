package daemon

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

// attachAdapter is the adapter for attached (hook-driven) sessions. Events
// are pushed directly through the daemon pump by hook handlers; approvals use
// the pendingHooks map; prompts are injected into the user's tmux pane.
type attachAdapter struct {
	server *Server
	cwd    string
}

func (a *attachAdapter) ID() string                    { return "" }
func (a *attachAdapter) Start(_ context.Context) error { return nil }
func (a *attachAdapter) Events() <-chan protocol.Event { return nil }
func (a *attachAdapter) Meta() protocol.SessionStartPayload {
	return protocol.SessionStartPayload{}
}

// SendApproval on an attachAdapter is only reached when the request is not in
// the server's pendingHooks map (see Server.dispatch), i.e. the hook already
// timed out or was resolved — so it always reports the approval as expired.
func (a *attachAdapter) SendApproval(_, _ string) error {
	return fmt.Errorf("approval expired or already handled")
}
func (a *attachAdapter) SendPrompt(text string) error {
	return a.server.injectPrompt(a.cwd, text)
}
func (a *attachAdapter) Alive() bool { return true }
func (a *attachAdapter) Stop() error { return nil }

// approvalHookTimeout bounds how long a PermissionRequest hook waits for a
// viewer decision before defaulting to deny. A var so tests can shrink it.
var approvalHookTimeout = 10 * time.Minute

// attachIdleTimeout bounds how long an attach session may go without any hook
// event before the sweeper marks it ended (#170). Hook activity is the only
// liveness signal for a claude process running in the user's own tmux; after
// `kill -9` the SessionEnd hook never fires, so without this the session
// would stay "running" forever. 30min is long enough that normal thinking
// pauses never trip it; if it does trip while claude is merely idle, the next
// hook revives the session in attachSession. A var so tests can shrink it.
var attachIdleTimeout = 30 * time.Minute

// attachSession returns the daemon session for a Claude session, creating it
// lazily on first hook activity.
func (s *Server) attachSession(claudeSID, cwd string) *session {
	s.mu.Lock()
	sess, ok := s.sessions[claudeSID]
	s.mu.Unlock()
	if ok {
		sess.mu.Lock()
		_, restored := sess.adapter.(*restoredAdapter)
		ended := sess.ended
		if restored {
			// The agent outlived a daemon restart (#170): the real claude
			// process is still running in the user's terminal and its hooks
			// keep firing. Swap the read-only restoredAdapter for a live
			// attachAdapter in place so prompts/approvals work again.
			sess.adapter = &attachAdapter{server: s, cwd: cwd}
		}
		if restored || ended {
			// Any hook proves the agent is alive: revive sessions previously
			// marked ended (e.g. by the sweeper's idle timeout while the user
			// simply left claude open).
			sess.status = protocol.StatusRunning
			sess.ended = false
			sess.lastSeen = time.Now()
		}
		sess.mu.Unlock()
		if restored {
			s.log.Printf("reattached session %s: restored adapter swapped for live attach adapter", claudeSID)
		} else if ended {
			s.log.Printf("session %s revived by hook activity", claudeSID)
		}
		if restored || ended {
			s.persistSession(sess)
			s.announceSessions()
		}
		return sess
	}
	name := claudeSID
	if cwd != "" {
		name = filepath.Base(cwd)
	}
	sess = &session{
		id:       claudeSID,
		meta:     protocol.SessionStartPayload{Name: name, CLI: "claude (attach)", Cwd: cwd},
		adapter:  &attachAdapter{server: s, cwd: cwd},
		status:   protocol.StatusRunning,
		lastSeen: time.Now(),
		clients:  map[*client]struct{}{},
	}
	s.mu.Lock()
	if existing, ok := s.sessions[claudeSID]; ok {
		// Lost a creation race with a concurrent hook: fall back to the
		// existing session (revive path above handles restored/ended).
		s.mu.Unlock()
		return existing
	}
	s.sessions[claudeSID] = sess
	s.mu.Unlock()
	s.log.Printf("attached session %s cwd=%s", claudeSID, cwd)
	ev, err := protocol.NewEvent(claudeSID, protocol.EventSessionStart, sess.meta)
	if err == nil {
		s.pumpEvent(sess, ev)
	}
	s.announceSessions()
	return sess
}

// injectPrompt types text into the tmux pane running claude in cwd and presses
// Enter. This is how web/remote instructions reach an interactive Claude Code
// session that the user opened themselves.
func (s *Server) injectPrompt(cwd, text string) error {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}\t#{pane_current_command}\t#{pane_current_path}").Output()
	if err != nil {
		return fmt.Errorf("tmux 不可用：%w（请把 claude 放进 tmux 再试）", err)
	}
	paneID := findClaudePane(string(out), cwd)
	if paneID == "" {
		return fmt.Errorf("未找到运行在 %s 的 claude tmux 面板（请把 claude 放进 tmux 再试）", cwd)
	}
	if err := exec.Command("tmux", "send-keys", "-t", paneID, "-l", text).Run(); err != nil {
		return fmt.Errorf("tmux send-keys: %w", err)
	}
	if err := exec.Command("tmux", "send-keys", "-t", paneID, "Enter").Run(); err != nil {
		return fmt.Errorf("tmux send-keys Enter: %w", err)
	}
	s.log.Printf("prompt injected into tmux pane=%s cwd=%s", paneID, cwd)
	return nil
}

// findClaudePane picks a pane whose current command looks like claude and
// whose path matches cwd.
func findClaudePane(tmuxOutput, cwd string) string {
	for _, line := range strings.Split(tmuxOutput, "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		id, cmd, path := parts[0], parts[1], parts[2]
		if strings.Contains(cmd, "claude") && path == cwd {
			return id
		}
	}
	return ""
}

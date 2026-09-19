package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

type session struct {
	id      string
	meta    protocol.SessionStartPayload
	adapter adapter.Session
	events  <-chan protocol.Event
	status  string
	ended   bool
	managed bool      // daemon spawned the agent process: Shutdown reclaims it (#170)
	lease   bool      // local TUI attached: session closes when heartbeat lapses
	lastHB  time.Time // last lease heartbeat from the local CLI
	// leaseMissed records that the previous sweep already saw the lease
	// expired; the session is closed only when two consecutive sweeps see no
	// heartbeat, so a post-sleep sweep racing the heartbeat can't kill a live
	// TUI (#170). Reset by every heartbeat.
	leaseMissed bool
	lastSeen    time.Time // last event activity (for dashboard "recent" display)
	created     time.Time
	connect     map[string]string // adapter connect info for restart recovery (e.g. codex socket/threadId)
	mu          sync.Mutex
	pumpMu      sync.Mutex // serializes pumpEvent across the pump, hook handlers, and viewer dispatch (#171)
	seq         uint64     // last assigned event sequence number (#173)
	history     []protocol.Event
	clients     map[*client]struct{}
}

// getAdapter returns the session's adapter under sess.mu. The adapter can be
// swapped in place when a restored session is re-attached by hook activity
// (#170), so readers must not cache sess.adapter without the lock.
func (sess *session) getAdapter() adapter.Session {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.adapter
}

// state returns the session's status and ended flag in one atomic read.
func (sess *session) state() (status string, ended bool) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.status, sess.ended
}

func (sess *session) setState(status string, ended bool) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	sess.status = status
	sess.ended = ended
}

func (sess *session) setStatus(status string) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	sess.status = status
}

func (sess *session) isEnded() bool {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.ended
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		list := make([]map[string]any, 0, len(s.sessions))
		for _, sess := range s.sessions {
			sess.mu.Lock()
			if sess.ended {
				sess.mu.Unlock()
				continue
			}
			lastSeen := sess.lastSeen
			status := sess.status
			sess.mu.Unlock()
			list = append(list, map[string]any{
				"id":       sess.id,
				"name":     sess.meta.Name,
				"cli":      sess.meta.CLI,
				"cwd":      sess.meta.Cwd,
				"status":   status,
				"lastSeen": lastSeen,
			})
		}
		// Stable, meaningful order for the client: most recently active first,
		// zero timestamps (restored sessions without activity) last, id as a
		// tiebreaker. Ranging over a Go map yields a random order per request,
		// which made the client list reshuffle on every 5s poll (#249).
		sort.Slice(list, func(i, j int) bool {
			a := list[i]["lastSeen"].(time.Time)
			b := list[j]["lastSeen"].(time.Time)
			if a.IsZero() != b.IsZero() {
				return !a.IsZero()
			}
			if !a.Equal(b) {
				return a.After(b)
			}
			return list[i]["id"].(string) < list[j]["id"].(string)
		})
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
	case http.MethodPost:
		s.createSession(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		CLI    string `json:"cli"`
		Binary string `json:"binary"`
		Cwd    string `json:"cwd"`
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.CLI == "" {
		req.CLI = "claude"
	}
	if req.Cwd == "" {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		req.Cwd = cwd
	}
	id := protocol.NewID()
	createReq := adapter.CreateRequest{
		ID:        id,
		Name:      req.Name,
		CLI:       req.CLI,
		Binary:    req.Binary,
		Cwd:       req.Cwd,
		Prompt:    req.Prompt,
		DataDir:   s.dataDir,
		HookBase:  fmt.Sprintf("http://127.0.0.1:%d", s.cfg.Port),
		HookToken: s.token,
	}
	factory := s.factory
	if factory == nil {
		factory = DefaultFactory()
	}
	sessAdapter, err := factory(context.Background(), createReq)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sess := &session{
		id:       id,
		meta:     sessAdapter.Meta(),
		adapter:  sessAdapter,
		events:   sessAdapter.Events(),
		status:   protocol.StatusRunning,
		ended:    false,
		managed:  true, // spawned by the daemon: Shutdown reclaims the process
		created:  time.Now(),
		lastSeen: time.Now(),
		clients:  map[*client]struct{}{},
	}
	if req.Prompt == "" {
		sess.status = protocol.StatusWaitingInput
	}
	s.persistSession(sess)
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	s.announceSessions()
	startEv, err := protocol.NewEvent(id, protocol.EventSessionStart, sess.meta)
	if err == nil {
		s.pumpEvent(sess, startEv)
	}
	// Respond with the initial status before starting the adapter, so the
	// create response is deterministic (e.g. waiting_input for an empty
	// prompt) instead of racing with the first agent status event.
	writeJSON(w, http.StatusOK, map[string]any{
		"id":     id,
		"name":   req.Name,
		"cli":    req.CLI,
		"cwd":    req.Cwd,
		"status": sess.status,
		"url":    fmt.Sprintf("http://127.0.0.1:%d/?session=%s&token=%s", s.cfg.Port, id, s.token),
	})
	go func() {
		if err := sessAdapter.Start(context.Background()); err != nil {
			s.log.Printf("session %s start error: %v", id, err)
			ev, _ := protocol.NewEvent(id, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "start_error: " + err.Error()})
			s.pumpEvent(sess, ev)
			return
		}
	}()
	go s.pump(sess)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(id, "/connect") {
		s.handleSessionConnect(w, r, strings.TrimSuffix(id, "/connect"))
		return
	}
	if strings.HasSuffix(id, "/heartbeat") {
		s.handleSessionHeartbeat(w, strings.TrimSuffix(id, "/heartbeat"))
		return
	}
	if strings.HasSuffix(id, "/pty") {
		s.handleSessionPTY(w, r, strings.TrimSuffix(id, "/pty"))
		return
	}
	if !strings.HasSuffix(id, "/stop") {
		http.NotFound(w, r)
		return
	}
	id = strings.TrimSuffix(id, "/stop")
	s.mu.Lock()
	_, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	s.stopSession(id)
	writeJSON(w, http.StatusOK, map[string]any{"stopped": true})
}

// handleSessionHeartbeat renews the local-TUI lease for a session. The first
// heartbeat enables the lease; once enabled, the session is closed if no
// heartbeat arrives within the lease window (see sweepOnce).
func (s *Server) handleSessionHeartbeat(w http.ResponseWriter, id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	s.mu.Unlock()
	if ok {
		sess.mu.Lock()
		sess.lease = true
		sess.lastHB = time.Now()
		sess.leaseMissed = false
		sess.mu.Unlock()
	}
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// stopSession removes a session from the live list and stops its adapter.
func (s *Server) stopSession(id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mu.Unlock()
	if !ok {
		return
	}
	sess.setState(protocol.StatusDone, true)
	s.persistSession(sess)
	s.announceSessions()
	_ = sess.getAdapter().Stop()
	s.log.Printf("session %s stopped", id)
}

// handleSessionConnect returns the local connect info (app-server socket +
// thread id) for adapters that support attaching a local TUI, waiting until
// the session is ready.
func (s *Server) handleSessionConnect(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	ci, ok := sess.getAdapter().(interface {
		ConnectInfo() (socket string, threadID string, err error)
	})
	if !ok {
		writeError(w, http.StatusBadRequest, "session does not support TUI attach")
		return
	}
	socket, threadID, err := ci.ConnectInfo()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"socket": socket, "threadId": threadID})
}

func (s *Server) pump(sess *session) {
	for ev := range sess.events {
		s.pumpEvent(sess, ev)
	}
}

func (s *Server) pumpEvent(sess *session, ev protocol.Event) {
	sess.pumpMu.Lock()
	defer sess.pumpMu.Unlock()
	sess.mu.Lock()
	sess.lastSeen = time.Now()
	sess.mu.Unlock()
	if ev.Type == protocol.EventSessionEnd || ev.Type == protocol.EventAgentStatus {
		var p protocol.AgentStatusPayload
		_ = ev.DecodePayload(&p)
		if ev.Type == protocol.EventSessionEnd {
			sess.setState(protocol.StatusDone, true)
			s.announceSessions()
		} else if p.Status != "" {
			sess.setStatus(p.Status)
		}
	}
	ev = sess.addEvent(ev)
	sess.broadcast(ev)
	s.persistEvent(sess, ev)
	if ev.Type == protocol.EventSessionEnd {
		s.persistSession(sess)
	}
}

func (s *Server) getSession(id string) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[id]
}

package daemon

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/packages/protocol"
)

// TestHookRoutesToHostedSessionByQuery: hosted interactive Claude registers
// hooks with ?session=<daemon id>; the handler must route to that session
// instead of creating a duplicate attach session keyed by Claude's own id.
func TestHookRoutesToHostedSessionByQuery(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, keys, dir, log.New(io.Discard, "", 0), nil)
	fake := &fakeSession{
		id:         "host-1",
		events:     make(chan protocol.Event, 16),
		approvals:  make(chan string, 1),
		prompts:    make(chan string, 1),
		stopCalled: make(chan struct{}, 1),
	}
	sess := &session{
		id:      "host-1",
		meta:    fake.Meta(),
		adapter: fake,
		events:  fake.events,
		status:  protocol.StatusRunning,
		clients: map[*client]struct{}{},
	}
	srv.sessions["host-1"] = sess
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"session_id":"claude-own-id","cwd":"/tmp","message":"done!","notification":{"type":"agent_completed","message":"done!"}}`
	req, err := http.NewRequest(http.MethodPost,
		ts.URL+"/hooks/claude/notification?session=host-1&token="+cfg.LocalToken,
		strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("hook status %d", resp.StatusCode)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.sessions) != 1 {
		t.Fatalf("expected only the hosted session, got %d sessions", len(srv.sessions))
	}
	if _, dup := srv.sessions["claude-own-id"]; dup {
		t.Fatal("duplicate attach session created despite ?session= routing")
	}
	sess = srv.sessions["host-1"]
	if n := len(sess.history); n == 0 || sess.history[n-1].Type != protocol.EventNotify {
		t.Fatalf("expected notify event in hosted session history, got %+v", sess.history)
	}
}

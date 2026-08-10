package daemon

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/packages/protocol"
)

func newKimiHookTestServer(t *testing.T) (*Server, *httptest.Server, *session, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, keys, dir, log.New(io.Discard, "", 0), nil)
	fake := &fakeSession{
		id:         "kimi-1",
		events:     make(chan protocol.Event, 32),
		approvals:  make(chan string, 1),
		prompts:    make(chan string, 1),
		stopCalled: make(chan struct{}, 1),
	}
	sess := &session{
		id:      "kimi-1",
		meta:    fake.Meta(),
		adapter: fake,
		events:  fake.events,
		status:  protocol.StatusRunning,
		clients: map[*client]struct{}{},
	}
	srv.sessions["kimi-1"] = sess
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts, sess, cfg.LocalToken
}

func postKimiHookBody(ts *httptest.Server, event, body, token string) (*http.Response, string, error) {
	req, err := http.NewRequest(http.MethodPost,
		ts.URL+"/hooks/kimi/"+event+"?session=kimi-1&token="+token,
		strings.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw), nil
}

func postKimiHook(t *testing.T, ts *httptest.Server, event, body, token string) (*http.Response, string) {
	t.Helper()
	resp, raw, err := postKimiHookBody(ts, event, body, token)
	if err != nil {
		t.Fatal(err)
	}
	return resp, raw
}

func TestKimiHookUserAndStopEvents(t *testing.T) {
	_, ts, sess, tok := newKimiHookTestServer(t)
	resp, _ := postKimiHook(t, ts, "user-prompt-submit", `{"session_id":"x","cwd":"/tmp","prompt":"hello"}`, tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("user-prompt status %d", resp.StatusCode)
	}
	resp, _ = postKimiHook(t, ts, "stop", `{"session_id":"x","cwd":"/tmp"}`, tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop status %d", resp.StatusCode)
	}
	sess.pumpMu.Lock()
	types := make([]string, 0, len(sess.history))
	for _, ev := range sess.history {
		types = append(types, ev.Type)
	}
	sess.pumpMu.Unlock()
	if len(types) < 2 || types[len(types)-2] != protocol.EventUserMessage || types[len(types)-1] != protocol.EventAgentStatus {
		t.Fatalf("unexpected history types: %v", types)
	}
}

func TestKimiHookPromptFlippedRunningAndIdleFlippedWaiting(t *testing.T) {
	_, ts, sess, tok := newKimiHookTestServer(t)

	// A finished turn (waiting for input): the next prompt must flip the
	// session back to running so the client shows the activity indicator.
	sess.mu.Lock()
	sess.status = protocol.StatusWaitingInput
	sess.mu.Unlock()
	resp, _ := postKimiHook(t, ts, "user-prompt-submit", `{"session_id":"x","cwd":"/tmp","prompt":"hello"}`, tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("user-prompt status %d", resp.StatusCode)
	}

	// idle_prompt notification ends the turn: back to waiting_input.
	resp, _ = postKimiHook(t, ts, "notification", `{"session_id":"x","cwd":"/tmp","notification":{"notification_type":"idle_prompt","body":"Kimi is waiting for your input"}}`, tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notification status %d", resp.StatusCode)
	}

	sess.pumpMu.Lock()
	types := make([]string, 0, len(sess.history))
	for _, ev := range sess.history {
		types = append(types, ev.Type)
	}
	sess.pumpMu.Unlock()
	want := []string{
		protocol.EventAgentStatus, // running
		protocol.EventUserMessage,
		protocol.EventAgentStatus, // waiting_input
		protocol.EventNotify,
	}
	if strings.Join(types, ",") != strings.Join(want, ",") {
		t.Fatalf("history types = %v, want %v", types, want)
	}
}

func TestKimiHookPreToolUseAutoAllowsReadTools(t *testing.T) {
	if !kimiGatedTools()["Bash"] {
		t.Fatal("Bash tool must be gated (kimi-code uses Bash, not Shell)")
	}
	_, ts, sess, tok := newKimiHookTestServer(t)
	resp, body := postKimiHook(t, ts, "pre-tool-use",
		`{"session_id":"x","cwd":"/tmp","tool_name":"ReadFile","tool_input":{"path":"/tmp/a"}}`, tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if !strings.Contains(body, `"permissionDecision":"allow"`) {
		t.Fatalf("expected allow, got %s", body)
	}
	sess.pumpMu.Lock()
	last := sess.history[len(sess.history)-1]
	sess.pumpMu.Unlock()
	if last.Type != protocol.EventToolCall {
		t.Fatalf("expected tool started event, got %s", last.Type)
	}
}

func TestKimiHookPreToolUseGatesShellUntilDecision(t *testing.T) {
	srv, ts, sess, tok := newKimiHookTestServer(t)
	var wg sync.WaitGroup
	var resp *http.Response
	var body string
	var postErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, body, postErr = postKimiHookBody(ts, "pre-tool-use",
			`{"session_id":"x","cwd":"/tmp","tool_name":"Shell","tool_input":{"command":"rm -rf /tmp/x"}}`, tok)
	}()

	// Wait for the pending approval to register, then approve from "phone".
	deadline := time.Now().Add(5 * time.Second)
	for {
		srv.mu.Lock()
		n := len(srv.pendingHooks)
		srv.mu.Unlock()
		if n == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("approval never registered")
		}
		time.Sleep(10 * time.Millisecond)
	}
	srv.mu.Lock()
	for id, ch := range srv.pendingHooks {
		delete(srv.pendingHooks, id)
		ch <- "approve"
	}
	srv.mu.Unlock()
	wg.Wait()
	if postErr != nil {
		t.Fatal(postErr)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if !strings.Contains(body, `"permissionDecision":"allow"`) {
		t.Fatalf("expected allow after approval, got %s", body)
	}
	// History must contain the approval request and a started tool row.
	sess.pumpMu.Lock()
	var hasApproval bool
	for _, ev := range sess.history {
		if ev.Type == protocol.EventApprovalReq {
			hasApproval = true
		}
	}
	sess.pumpMu.Unlock()
	if !hasApproval {
		t.Fatal("approval request event missing from history")
	}
}

func TestKimiHookPostToolUseShell(t *testing.T) {
	_, ts, sess, tok := newKimiHookTestServer(t)
	resp, _ := postKimiHook(t, ts, "post-tool-use",
		`{"session_id":"x","cwd":"/tmp","tool_name":"Shell","tool_input":{"command":"ls"}}`, tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	sess.pumpMu.Lock()
	types := make([]string, 0, len(sess.history))
	for _, ev := range sess.history {
		types = append(types, ev.Type)
	}
	sess.pumpMu.Unlock()
	if types[len(types)-2] != protocol.EventCommand || types[len(types)-1] != protocol.EventToolCall {
		t.Fatalf("unexpected history: %v", types)
	}
}

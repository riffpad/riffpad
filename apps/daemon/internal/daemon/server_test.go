package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/packages/protocol"
)

func TestHistorySlice(t *testing.T) {
	events := []protocol.Event{
		{ID: "e1"}, {ID: "e2"}, {ID: "e3"}, {ID: "e4"}, {ID: "e5"},
	}
	got := historySlice(events, "e3", 2)
	if len(got) != 2 || got[0].ID != "e1" || got[1].ID != "e2" {
		t.Fatalf("unexpected slice: %+v", got)
	}
	got = historySlice(events, "e5", 200)
	if len(got) != 4 || got[0].ID != "e1" {
		t.Fatalf("limit above available: %+v", got)
	}
	if got := historySlice(events, "unknown", 2); got != nil {
		t.Fatalf("unknown anchor should return nil, got %+v", got)
	}
	if got := historySlice(events, "e1", 2); got != nil {
		t.Fatalf("first event has no history, got %+v", got)
	}
}

func TestSnapshotLast(t *testing.T) {
	sess := &session{}
	for i := 0; i < 5; i++ {
		sess.history = append(sess.history, protocol.Event{ID: fmt.Sprintf("e%d", i)})
	}
	got := sess.snapshotLast(3)
	if len(got) != 3 || got[0].ID != "e2" {
		t.Fatalf("unexpected snapshotLast: %+v", got)
	}
	if all := sess.snapshotLast(10); len(all) != 5 {
		t.Fatalf("snapshotLast over count: %d", len(all))
	}
}

type fakeSession struct {
	id           string
	events       chan protocol.Event
	approvals    chan string
	prompts      chan string
	stopCalled   chan struct{}
	lastDecision string
}

func (f *fakeSession) ID() string                    { return f.id }
func (f *fakeSession) Events() <-chan protocol.Event { return f.events }
func (f *fakeSession) Meta() protocol.SessionStartPayload {
	return protocol.SessionStartPayload{Name: "fake", CLI: "fake", Cwd: "/tmp"}
}

func (f *fakeSession) Start(_ context.Context) error {
	go func() {
		ev, _ := protocol.NewEvent(f.id, protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusRunning})
		f.events <- ev
		ev2, _ := protocol.NewEvent(f.id, protocol.EventApprovalReq, protocol.ApprovalRequestPayload{
			RequestID: "req-fake-1", Action: "Bash", Summary: "rm -rf build", Options: []string{"approve", "reject"},
		})
		f.events <- ev2
		decision := <-f.approvals
		ev3, _ := protocol.NewEvent(f.id, protocol.EventAgentStatus, protocol.AgentStatusPayload{Status: protocol.StatusDone})
		f.events <- ev3
		ev4, _ := protocol.NewEvent(f.id, protocol.EventSessionEnd, protocol.SessionEndPayload{Reason: "decision:" + decision})
		f.events <- ev4
		close(f.events)
	}()
	return nil
}

func (f *fakeSession) SendApproval(requestID string, decision string) error {
	if requestID != "req-fake-1" {
		return fmt.Errorf("unknown approval request %s", requestID)
	}
	f.lastDecision = decision
	f.approvals <- decision
	return nil
}

func (f *fakeSession) SendPrompt(text string) error {
	f.prompts <- text
	return nil
}

func (f *fakeSession) Alive() bool { return true }

func (f *fakeSession) Stop() error {
	f.stopCalled <- struct{}{}
	return nil
}

// TestDispatchUnknownApprovalNotifiesViewer covers the late-approval case: a
// viewer taps "approve" while offline, the request times out on the daemon,
// and the flushed approval_response arrives with an unknown requestID. The
// daemon must ack the sending viewer with an error notify carrying the
// requestID so the client can mark the card as expired instead of "已批准".
func TestDispatchUnknownApprovalNotifiesViewer(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)

	fake := &fakeSession{
		id:         "s1",
		events:     make(chan protocol.Event, 1),
		approvals:  make(chan string, 1),
		prompts:    make(chan string, 1),
		stopCalled: make(chan struct{}, 1),
	}
	sess := &session{id: "s1", adapter: fake, clients: map[*client]struct{}{}}
	key := &[32]byte{1, 2, 3}
	c := &client{
		deviceID: "d1",
		session:  sess,
		key:      key,
		send:     make(chan []byte, 1),
		done:     make(chan struct{}),
		log:      logger,
	}

	payload, _ := json.Marshal(protocol.ApprovalResponsePayload{RequestID: "hook-gone", Decision: "approve"})
	srv.dispatch(c, protocol.Event{ID: "e1", SessionID: "s1", Type: protocol.EventApprovalResp, Payload: payload})

	select {
	case data := <-c.send:
		var env protocol.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			t.Fatal(err)
		}
		plain, err := env.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		var ev protocol.Event
		if err := json.Unmarshal(plain, &ev); err != nil {
			t.Fatal(err)
		}
		if ev.Type != protocol.EventNotify {
			t.Fatalf("expected notify, got %s", ev.Type)
		}
		var n protocol.NotifyPayload
		if err := ev.DecodePayload(&n); err != nil {
			t.Fatal(err)
		}
		if n.Level != "error" || n.RequestID != "hook-gone" {
			t.Fatalf("unexpected notify payload: %+v", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no expired notify sent to the viewer")
	}
}

// TestDispatchPendingHookResolves ensures the pendingHooks path still wins
// and does not emit an expired notify.
func TestDispatchPendingHookResolves(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)

	fake := &fakeSession{
		id:         "s1",
		events:     make(chan protocol.Event, 1),
		approvals:  make(chan string, 1),
		prompts:    make(chan string, 1),
		stopCalled: make(chan struct{}, 1),
	}
	sess := &session{id: "s1", adapter: fake, clients: map[*client]struct{}{}}
	key := &[32]byte{1, 2, 3}
	c := &client{
		deviceID: "d1",
		session:  sess,
		key:      key,
		send:     make(chan []byte, 1),
		done:     make(chan struct{}),
		log:      logger,
	}

	ch := make(chan string, 1)
	srv.mu.Lock()
	srv.pendingHooks["hook-live"] = ch
	srv.mu.Unlock()

	payload, _ := json.Marshal(protocol.ApprovalResponsePayload{RequestID: "hook-live", Decision: "approve"})
	srv.dispatch(c, protocol.Event{ID: "e1", SessionID: "s1", Type: protocol.EventApprovalResp, Payload: payload})

	select {
	case d := <-ch:
		if d != "approve" {
			t.Fatalf("unexpected decision %q", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending hook not resolved")
	}
	select {
	case data := <-c.send:
		t.Fatalf("unexpected viewer notify: %s", data)
	default:
	}
}

func TestPairCreateSessionAndApprovalLoop(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	var fake *fakeSession
	factory := func(_ context.Context, req adapter.CreateRequest) (adapter.Session, error) {
		fake = &fakeSession{
			id:         req.ID,
			events:     make(chan protocol.Event, 16),
			approvals:  make(chan string, 1),
			prompts:    make(chan string, 1),
			stopCalled: make(chan struct{}, 1),
		}
		return fake, nil
	}
	srv := New(cfg, keys, dir, logger, factory)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Create pairing code and pair a P-256 web client.
	resp := authRequest(t, http.MethodPost, ts.URL+"/api/pairings", cfg.LocalToken, nil)
	var pr struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	clientID, err := protocol.GenerateKeyPair(protocol.CurveP256)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{
		"code": pr.Code, "name": "test", "curve": "p256", "publicKey": protocol.EncodeKey(clientID.PublicKey),
	})
	resp = authRequest(t, http.MethodPost, ts.URL+"/api/pair", cfg.LocalToken, strings.NewReader(string(body)))
	var pair struct {
		DeviceID        string `json:"deviceId"`
		ServerPublicKey string `json:"serverPublicKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pair); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if pair.DeviceID == "" {
		t.Fatal("pairing failed")
	}
	serverPub, err := protocol.DecodeKey(pair.ServerPublicKey)
	if err != nil {
		t.Fatal(err)
	}
	deviceSecret, err := protocol.NewDeviceSecret(clientID, serverPub)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Create a session (fake adapter).
	body, _ = json.Marshal(map[string]string{"name": "demo", "cli": "fake"})
	resp = authRequest(t, http.MethodPost, ts.URL+"/api/sessions", cfg.LocalToken, strings.NewReader(string(body)))
	var sess struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if sess.ID == "" {
		t.Fatal("session create failed")
	}
	if sess.Status != protocol.StatusWaitingInput {
		t.Fatalf("expected waiting_input for empty prompt, got %s", sess.Status)
	}

	// 3. Connect WS with an ephemeral P-256 key.
	clientEph, err := protocol.GenerateKeyPair(protocol.CurveP256)
	if err != nil {
		t.Fatal(err)
	}
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") +
		"/ws?device=" + pair.DeviceID + "&session=" + sess.ID +
		"&eph=" + protocol.EncodeKey(clientEph.PublicKey) +
		"&token=" + cfg.LocalToken
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var hello protocol.Hello
	if err := json.Unmarshal(data, &hello); err != nil {
		t.Fatal(err)
	}
	if hello.Kind != "hello" {
		t.Fatalf("expected hello, got %s", data)
	}
	serverEphPub, err := protocol.DecodeKey(hello.ServerEphPub)
	if err != nil {
		t.Fatal(err)
	}
	ephSecret, err := protocol.ECDH(clientEph, serverEphPub)
	if err != nil {
		t.Fatal(err)
	}
	key, err := protocol.DeriveSessionKey(deviceSecret, ephSecret, sess.ID)
	if err != nil {
		t.Fatal(err)
	}

	readEvent := func() protocol.Event {
		t.Helper()
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var env protocol.Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatal(err)
		}
		plain, err := env.Open(key)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		var ev protocol.Event
		if err := json.Unmarshal(plain, &ev); err != nil {
			t.Fatal(err)
		}
		return ev
	}

	ev := readEvent()
	if ev.Type != protocol.EventSessionStart {
		t.Fatalf("expected session_start, got %s", ev.Type)
	}
	ev = readEvent()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected agent_status, got %s", ev.Type)
	}
	ev = readEvent()
	if ev.Type != protocol.EventApprovalReq {
		t.Fatalf("expected approval_request, got %s", ev.Type)
	}

	// 4. Send an encrypted prompt and approval response.
	payload, _ := json.Marshal(protocol.PromptPayload{Text: "请继续"})
	msg := protocol.Event{ID: protocol.NewID(), SessionID: sess.ID, Timestamp: time.Now().UnixMilli(), Type: protocol.EventPrompt, Payload: payload}
	sendEncrypted(t, conn, sess.ID, key, msg)
	select {
	case got := <-fake.prompts:
		if got != "请继续" {
			t.Fatalf("unexpected prompt %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("prompt not delivered to adapter")
	}

	payload, _ = json.Marshal(protocol.ApprovalResponsePayload{RequestID: "req-fake-1", Decision: "approve"})
	msg = protocol.Event{ID: protocol.NewID(), SessionID: sess.ID, Timestamp: time.Now().UnixMilli(), Type: protocol.EventApprovalResp, Payload: payload}
	sendEncrypted(t, conn, sess.ID, key, msg)

	// The resolution broadcast and the adapter's own events race on the wire
	// (approval_resolved can land after session_end), so collect until BOTH
	// have arrived and assert on the set (#171).
	var gotResolved *protocol.ApprovalResolvedPayload
	gotDone := false
	gotEnd := false
	deadline := time.Now().Add(10 * time.Second)
	for gotResolved == nil || !gotEnd {
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for approval_resolved and session_end (resolved=%v end=%v)", gotResolved != nil, gotEnd)
		}
		ev = readEvent()
		switch ev.Type {
		case protocol.EventApprovalResolved:
			var arp protocol.ApprovalResolvedPayload
			if err := ev.DecodePayload(&arp); err != nil {
				t.Fatal(err)
			}
			gotResolved = &arp
		case protocol.EventAgentStatus:
			var st protocol.AgentStatusPayload
			if err := ev.DecodePayload(&st); err != nil {
				t.Fatal(err)
			}
			if st.Status == protocol.StatusDone {
				gotDone = true
			}
		case protocol.EventSessionEnd:
			gotEnd = true
		}
	}
	if gotResolved == nil {
		t.Fatal("expected approval_resolved broadcast after approval")
	}
	if gotResolved.RequestID != "req-fake-1" || gotResolved.Decision != "approve" || gotResolved.DeviceID != pair.DeviceID {
		t.Fatalf("unexpected approval_resolved: %+v", gotResolved)
	}
	if !gotDone {
		t.Fatal("expected agent_status done after approval")
	}
	if fake.lastDecision != "approve" {
		t.Fatalf("unexpected decision %q", fake.lastDecision)
	}
	// The daemon persists the final session events asynchronously (broadcast
	// reaches the viewer before the last write finishes). Wait for the meta
	// file to show the session as ended, then clean up so t.TempDir() does
	// not race an in-flight write.
	metaPath := filepath.Join(dir, "sessions", sess.ID, "meta.json")
	var ps PersistedSession
	for i := 0; i < 100; i++ {
		data, err := os.ReadFile(metaPath)
		if err == nil && json.Unmarshal(data, &ps) == nil && ps.Ended {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ps.Ended {
		t.Fatalf("session meta not finalized before cleanup")
	}
	_ = os.RemoveAll(filepath.Join(dir, "sessions", sess.ID))
}

func TestLeaseExpiryClosesSession(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)

	fake := &fakeSession{
		id:         "lease-1",
		events:     make(chan protocol.Event, 4),
		approvals:  make(chan string, 1),
		prompts:    make(chan string, 1),
		stopCalled: make(chan struct{}, 1),
	}
	sess := &session{
		id:      "lease-1",
		meta:    fake.Meta(),
		adapter: fake,
		events:  fake.events,
		status:  protocol.StatusRunning,
		lease:   true,
		lastHB:  time.Now().Add(-30 * time.Second),
		clients: map[*client]struct{}{},
	}
	srv.sessions["lease-1"] = sess

	// First sweep after expiry only marks the lease as missed (#170 grace
	// period: a post-sleep sweep can race the TUI heartbeat).
	srv.sweepOnce()

	srv.mu.Lock()
	_, stillThere := srv.sessions["lease-1"]
	srv.mu.Unlock()
	if !stillThere {
		t.Fatal("first sweep after lease expiry should grant a grace period, not close")
	}
	select {
	case <-fake.stopCalled:
		t.Fatal("adapter.Stop must not be called on the first expired sweep")
	default:
	}

	// A second consecutive sweep without heartbeat closes the session.
	srv.sweepOnce()

	srv.mu.Lock()
	_, stillThere = srv.sessions["lease-1"]
	srv.mu.Unlock()
	if stillThere {
		t.Fatal("expired-lease session should have been removed after two sweeps")
	}
	select {
	case <-fake.stopCalled:
	default:
		t.Fatal("expected adapter.Stop to be called")
	}
}

// TestLeaseGraceResetByHeartbeat covers the laptop-lid scenario (#170): the
// machine sleeps, the sweep and the heartbeat both come due, and the sweep
// runs first. The grace strike must be cleared by the heartbeat so the next
// sweep starts over instead of closing a live TUI session.
func TestLeaseGraceResetByHeartbeat(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)

	fake := &fakeSession{
		id:         "lease-2",
		events:     make(chan protocol.Event, 4),
		approvals:  make(chan string, 1),
		prompts:    make(chan string, 1),
		stopCalled: make(chan struct{}, 1),
	}
	srv.sessions["lease-2"] = &session{
		id:      "lease-2",
		meta:    fake.Meta(),
		adapter: fake,
		events:  fake.events,
		status:  protocol.StatusRunning,
		lease:   true,
		lastHB:  time.Now().Add(-30 * time.Second),
		clients: map[*client]struct{}{},
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	alive := func() bool {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		_, ok := srv.sessions["lease-2"]
		return ok
	}
	stale := func() {
		srv.mu.Lock()
		srv.sessions["lease-2"].lastHB = time.Now().Add(-30 * time.Second)
		srv.mu.Unlock()
	}

	// Sweep wins the post-sleep race: strike one, session survives.
	srv.sweepOnce()
	if !alive() {
		t.Fatal("first expired sweep must not close the session")
	}
	// The TUI heartbeat arrives right after: the strike must be cleared.
	resp := authRequest(t, http.MethodPost, ts.URL+"/api/sessions/lease-2/heartbeat", cfg.LocalToken, nil)
	resp.Body.Close()
	srv.mu.Lock()
	missed := srv.sessions["lease-2"].leaseMissed
	srv.mu.Unlock()
	if missed {
		t.Fatal("heartbeat should clear the grace strike")
	}
	// The machine sleeps again; the next expiry must again need two sweeps.
	stale()
	srv.sweepOnce()
	if !alive() {
		t.Fatal("after a heartbeat the grace period must start over")
	}
	srv.sweepOnce()
	if alive() {
		t.Fatal("two consecutive expired sweeps without heartbeat should close the session")
	}
}

// TestShutdownStopsManagedSessions covers #170: daemon shutdown must reclaim
// agent processes it spawned (they run on context.Background()), while attach
// sessions — claude running in the user's own tmux — are left alone.
func TestShutdownStopsManagedSessions(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)

	fake := &fakeSession{
		id:         "managed-1",
		events:     make(chan protocol.Event, 4),
		approvals:  make(chan string, 1),
		prompts:    make(chan string, 1),
		stopCalled: make(chan struct{}, 1),
	}
	srv.sessions["managed-1"] = &session{
		id:      "managed-1",
		meta:    fake.Meta(),
		adapter: fake,
		events:  fake.events,
		status:  protocol.StatusRunning,
		managed: true,
		clients: map[*client]struct{}{},
	}
	// An attach session and a restored session must not be touched (their
	// Stop is a no-op anyway; this pins the managed-only selection).
	srv.sessions["attach-1"] = &session{
		id:      "attach-1",
		adapter: &attachAdapter{server: srv},
		status:  protocol.StatusRunning,
		clients: map[*client]struct{}{},
	}
	srv.sessions["restored-1"] = &session{
		id:      "restored-1",
		adapter: &restoredAdapter{id: "restored-1"},
		status:  "restored",
		clients: map[*client]struct{}{},
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-fake.stopCalled:
	default:
		t.Fatal("shutdown should stop daemon-spawned sessions")
	}
}

func TestHeartbeatRenewsLease(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)

	fake := &fakeSession{
		id:         "hb-1",
		events:     make(chan protocol.Event, 4),
		approvals:  make(chan string, 1),
		prompts:    make(chan string, 1),
		stopCalled: make(chan struct{}, 1),
	}
	srv.sessions["hb-1"] = &session{
		id:      "hb-1",
		meta:    fake.Meta(),
		adapter: fake,
		events:  fake.events,
		status:  protocol.StatusRunning,
		clients: map[*client]struct{}{},
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	resp := authRequest(t, http.MethodPost, ts.URL+"/api/sessions/hb-1/heartbeat", cfg.LocalToken, nil)
	resp.Body.Close()

	srv.mu.Lock()
	s := srv.sessions["hb-1"]
	srv.mu.Unlock()
	if s == nil || !s.lease {
		t.Fatal("heartbeat should enable lease")
	}
	if time.Since(s.lastHB) > 2*time.Second {
		t.Fatal("heartbeat should refresh lastHB")
	}
}

// authRequest performs an HTTP request carrying the daemon's local API
// token, the way the riffpad CLI does.
func authRequest(t *testing.T, method, url, token string, body io.Reader) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set(LocalTokenHeader, token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func sendEncrypted(t *testing.T, conn *websocket.Conn, sid string, key *[32]byte, ev protocol.Event) {
	t.Helper()
	plain, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	env, err := protocol.WrapEnvelope(sid, plain, key)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatal(err)
	}
}

// TestLocalPairingMarkedLocal verifies that in local mode (no relay) the
// pairing response is flagged `local` and its URL targets 127.0.0.1, so the
// CLI knows to print the URL for a same-machine browser instead of a QR code.
func TestLocalPairingMarkedLocal(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := authRequest(t, http.MethodPost, ts.URL+"/api/pairings", cfg.LocalToken, nil)
	var pr struct {
		Code  string `json:"code"`
		URL   string `json:"url"`
		Local bool   `json:"local"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if pr.Code == "" {
		t.Fatal("no pairing code returned")
	}
	if !pr.Local {
		t.Fatal("local pairing response missing local flag")
	}
	want := fmt.Sprintf("http://127.0.0.1:%d/?pair=%s&token=%s", cfg.Port, pr.Code, cfg.LocalToken)
	if pr.URL != want {
		t.Fatalf("pairing url %q, want %q", pr.URL, want)
	}
}

// TestRemotePairingExpiredTokenHint: when the relay rejects /api/pairings
// with 401 (login token expired after its 30-day TTL), the daemon must answer
// with an actionable re-login hint and a stable errorCode the CLI can
// localize, instead of passing through the relay's bare "unauthorized" (#172).
func TestRemotePairingExpiredTokenHint(t *testing.T) {
	dir := t.TempDir()
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer relay.Close()

	cfg := config.Default()
	cfg.LocalToken = config.NewLocalToken()
	cfg.RelayURL = relay.URL
	cfg.RelayToken = "expired-token"
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := authRequest(t, http.MethodPost, ts.URL+"/api/pairings", cfg.LocalToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", resp.StatusCode)
	}
	var out struct {
		Error     string `json:"error"`
		ErrorCode string `json:"errorCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.ErrorCode != "relay_auth_expired" {
		t.Fatalf("errorCode %q, want relay_auth_expired", out.ErrorCode)
	}
	if !strings.Contains(out.Error, "riffpad login") {
		t.Fatalf("error should point at re-login, got %q", out.Error)
	}
}

// TestLocalPairingQueryForcesLocalCode: a relay-connected daemon normally
// mints cloud codes (createRemotePairing), which the embedded 8787 UI cannot
// claim (handlePair only looks up local pending codes). ?local=1 must bypass
// the relay and mint a local code the 8787 UI can claim.
func TestLocalPairingQueryForcesLocalCode(t *testing.T) {
	dir := t.TempDir()
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("daemon hit relay for a local-forced pairing: %s", r.URL.Path)
	}))
	defer relay.Close()

	cfg := config.Default()
	cfg.LocalToken = config.NewLocalToken()
	cfg.RelayURL = relay.URL
	cfg.RelayToken = "token"
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := authRequest(t, http.MethodPost, ts.URL+"/api/pairings?local=1", cfg.LocalToken, nil)
	defer resp.Body.Close()
	var pr struct {
		Code  string `json:"code"`
		URL   string `json:"url"`
		Local bool   `json:"local"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	if pr.Code == "" || !pr.Local {
		t.Fatalf("expected a local code, got %+v", pr)
	}
	if !strings.Contains(pr.URL, "127.0.0.1") {
		t.Fatalf("expected a 127.0.0.1 URL, got %q", pr.URL)
	}
}

// TestPairEndpointExemptFromLocalToken: /api/pair must be reachable without
// the local API token — the pairing code is the credential. A tokenless
// request must pass localAuth and reach handlePair (which rejects the bogus
// code), not be blocked by localAuth's "missing...local token" 401.
func TestPairEndpointExemptFromLocalToken(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, keys, dir, log.New(io.Discard, "", 0), nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// No Authorization header, no ?token=. localAuth must let it through.
	resp, err := http.Post(ts.URL+"/api/pair", "application/json", strings.NewReader(`{"code":"BOGUS"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "local token") {
		t.Fatalf("localAuth blocked tokenless /api/pair: %d %s", resp.StatusCode, body)
	}
}

// TestLoadDevicesHealsCorrupted: a truncated devices.json is backed up to
// devices.json.bak and the daemon starts with an empty device list instead of
// silently ignoring the file (#172).
func TestLoadDevicesHealsCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devices.json")
	if err := os.WriteFile(path, []byte(`[{"id": "d1", "name": "trun`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)
	srv.mu.Lock()
	n := len(srv.devices)
	srv.mu.Unlock()
	if n != 0 {
		t.Fatalf("expected no devices after healing, got %d", n)
	}
	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal("expected devices.json.bak backup")
	}
	if string(bak) != `[{"id": "d1", "name": "trun` {
		t.Fatalf("backup content %q", bak)
	}
}

// TestLocalPairingSweepsExpiredCodes: expired pending codes were only reaped
// when used, so repeated `riffpad pair` calls grew s.pending without bound;
// creating a new code now sweeps the stale ones (#174).
func TestLocalPairingSweepsExpiredCodes(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	srv.mu.Lock()
	srv.pending["STALE1"] = pendingPair{Code: "STALE1", Expires: time.Now().Add(-time.Minute)}
	srv.pending["FRESH1"] = pendingPair{Code: "FRESH1", Expires: time.Now().Add(time.Minute)}
	srv.mu.Unlock()

	resp := authRequest(t, http.MethodPost, ts.URL+"/api/pairings", cfg.LocalToken, nil)
	var pr struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if pr.Code == "" {
		t.Fatal("no pairing code returned")
	}

	srv.mu.Lock()
	_, stale := srv.pending["STALE1"]
	_, fresh := srv.pending["FRESH1"]
	_, newCode := srv.pending[pr.Code]
	srv.mu.Unlock()
	if stale {
		t.Fatal("expired pairing code was not swept")
	}
	if !fresh {
		t.Fatal("unexpired pairing code should be kept")
	}
	if !newCode {
		t.Fatal("new pairing code missing from pending")
	}
}

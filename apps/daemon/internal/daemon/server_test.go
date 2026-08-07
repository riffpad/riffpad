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

	ev = readEvent()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected agent_status after approval, got %s", ev.Type)
	}
	var st protocol.AgentStatusPayload
	if err := ev.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusDone {
		t.Fatalf("expected done, got %s", st.Status)
	}
	ev = readEvent()
	if ev.Type != protocol.EventSessionEnd {
		t.Fatalf("expected session_end, got %s", ev.Type)
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

	srv.sweepOnce()

	srv.mu.Lock()
	_, stillThere := srv.sessions["lease-1"]
	srv.mu.Unlock()
	if stillThere {
		t.Fatal("expired-lease session should have been removed")
	}
	select {
	case <-fake.stopCalled:
	default:
		t.Fatal("expected adapter.Stop to be called")
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

package daemon

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/packages/protocol"
)

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

func (f *fakeSession) SendApproval(_ string, decision string) error {
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
	resp, err := http.Post(ts.URL+"/api/pairings", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
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
	resp, err = http.Post(ts.URL+"/api/pair", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
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
	resp, err = http.Post(ts.URL+"/api/sessions", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
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
		"&eph=" + protocol.EncodeKey(clientEph.PublicKey)
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
	resp, err := http.Post(ts.URL+"/api/sessions/hb-1/heartbeat", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
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

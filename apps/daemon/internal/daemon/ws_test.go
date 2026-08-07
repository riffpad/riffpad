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

// shrinkWSHeartbeat shortens the heartbeat parameters for tests and restores
// them afterwards. The parameters are atomics, so this is race-safe even
// while connection goroutines are still running.
func shrinkWSHeartbeat(t *testing.T, pingPeriod, pongWait time.Duration) {
	t.Helper()
	oldWrite, oldPing, oldPong := wsWriteWait.Load(), wsPingPeriod.Load(), wsPongWait.Load()
	wsWriteWait.Store(int64(100 * time.Millisecond))
	wsPingPeriod.Store(int64(pingPeriod))
	wsPongWait.Store(int64(pongWait))
	t.Cleanup(func() {
		wsWriteWait.Store(oldWrite)
		wsPingPeriod.Store(oldPing)
		wsPongWait.Store(oldPong)
	})
}

// setupWSTest starts a daemon server with a paired device and one session.
func setupWSTest(t *testing.T) (ts *httptest.Server, deviceID, sessionID string) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	factory := func(_ context.Context, req adapter.CreateRequest) (adapter.Session, error) {
		return &fakeSession{
			id:         req.ID,
			events:     make(chan protocol.Event, 16),
			approvals:  make(chan string, 1),
			prompts:    make(chan string, 1),
			stopCalled: make(chan struct{}, 1),
		}, nil
	}
	srv := New(cfg, keys, dir, logger, factory)
	ts = httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

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
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pair); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if pair.DeviceID == "" {
		t.Fatal("pairing failed")
	}

	resp, err = http.Post(ts.URL+"/api/sessions", "application/json", strings.NewReader(`{"name":"demo","cli":"fake"}`))
	if err != nil {
		t.Fatal(err)
	}
	var sess struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if sess.ID == "" {
		t.Fatal("session create failed")
	}
	return ts, pair.DeviceID, sess.ID
}

func dialViewerWS(t *testing.T, ts *httptest.Server, deviceID, sessionID string) *websocket.Conn {
	t.Helper()
	eph, err := protocol.GenerateKeyPair(protocol.CurveP256)
	if err != nil {
		t.Fatal(err)
	}
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") +
		"/ws?device=" + deviceID + "&session=" + sessionID +
		"&eph=" + protocol.EncodeKey(eph.PublicKey)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestWSHeartbeatDropsSilentPeer(t *testing.T) {
	shrinkWSHeartbeat(t, 50*time.Millisecond, 300*time.Millisecond)
	ts, deviceID, sessionID := setupWSTest(t)
	conn := dialViewerWS(t, ts, deviceID, sessionID)

	// Half-open peer: protocol pings are swallowed, never answered. The
	// daemon must notice via the read deadline and close the connection.
	conn.SetPingHandler(func(string) error { return nil })
	start := time.Now()
	conn.SetReadDeadline(start.Add(5 * time.Second))
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("daemon did not close the half-open connection (waited %v)", elapsed)
	}
}

func TestWSHeartbeatSendsAppPing(t *testing.T) {
	shrinkWSHeartbeat(t, 50*time.Millisecond, 10*time.Second)
	ts, deviceID, sessionID := setupWSTest(t)
	conn := dialViewerWS(t, ts, deviceID, sessionID)

	// Browser clients cannot see protocol-level ping/pong, so the daemon
	// sends an application-level {"kind":"ping"} keepalive for their silence
	// watchdog. The default gorilla ping handler answers protocol pings while
	// reading, keeping the connection alive on both levels.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("no app keepalive received before timeout: %v", err)
		}
		if strings.Contains(string(data), `"kind":"ping"`) {
			break
		}
	}
}

func TestRelayClientRunOnceDetectsHalfOpen(t *testing.T) {
	shrinkWSHeartbeat(t, 50*time.Millisecond, 300*time.Millisecond)

	// Fake relay that upgrades and then goes silent: it never reads and
	// never answers protocol pings, simulating a half-open uplink.
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	quit := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetPingHandler(func(string) error { return nil })
		<-quit
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(quit) })

	c := newRelayClient("ws"+strings.TrimPrefix(srv.URL, "http"), "h1", "s1",
		log.New(io.Discard, "", 0), nil)
	done := make(chan error, 1)
	go func() { done <- c.runOnce(context.Background()) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("runOnce returned nil on a half-open connection")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runOnce did not detect the half-open connection")
	}
}

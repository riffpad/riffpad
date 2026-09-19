package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
)

var upgrader = websocket.Upgrader{
	// The localAuth middleware has already validated the token and any Origin
	// header; this is defense in depth for the WS handshake itself. Non-browser
	// clients send no Origin and are allowed through.
	CheckOrigin: func(r *http.Request) bool {
		o := r.Header.Get("Origin")
		return o == "" || loopbackOrigin(o)
	},
}

// WebSocket heartbeat parameters, shared by local viewer connections and the
// relay uplink. Pings go out every wsPingPeriod; if no pong (or any other
// frame) arrives within wsPongWait the connection is treated as half-open and
// torn down so reconnect logic can take over. Atomics (not constants) so
// tests can shrink them without data races.
var (
	wsWriteWait  atomic.Int64
	wsPingPeriod atomic.Int64
	wsPongWait   atomic.Int64
)

func init() {
	wsWriteWait.Store(int64(10 * time.Second))
	wsPingPeriod.Store(int64(30 * time.Second))
	wsPongWait.Store(int64(75 * time.Second))
}

func wsWriteDeadline() time.Time    { return time.Now().Add(time.Duration(wsWriteWait.Load())) }
func wsPingInterval() time.Duration { return time.Duration(wsPingPeriod.Load()) }
func wsPongTimeout() time.Duration  { return time.Duration(wsPongWait.Load()) }

const (
	// historyReplayLimit is how many recent events a viewer gets on connect.
	historyReplayLimit = 100
	// historyQueryLimit caps how many older events a page request may fetch.
	historyQueryLimit = 200
)

// historySlice returns the events strictly before the event with the given
// id, oldest first, at most limit entries. Events are already in
// chronological order.
func historySlice(events []protocol.Event, before string, limit int) []protocol.Event {
	if before == "" || limit <= 0 {
		return nil
	}
	idx := -1
	for i, ev := range events {
		if ev.ID == before {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return nil
	}
	start := idx - limit
	if start < 0 {
		start = 0
	}
	return events[start:idx]
}

type wsTransport struct {
	conn      *websocket.Conn
	done      chan struct{}
	closeOnce sync.Once
}

func (t *wsTransport) Send(data []byte) error {
	_ = t.conn.SetWriteDeadline(wsWriteDeadline())
	return t.conn.WriteMessage(websocket.TextMessage, data)
}

func (t *wsTransport) Recv() ([]byte, error) {
	_, data, err := t.conn.ReadMessage()
	return data, err
}

func (t *wsTransport) Close() error {
	t.closeOnce.Do(func() { close(t.done) })
	return t.conn.Close()
}

// wsPingLoop sends protocol pings until done is closed or a write fails.
func wsPingLoop(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(wsPingInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, wsWriteDeadline()); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	deviceID := q.Get("device")
	sid := q.Get("session")
	ephRaw := q.Get("eph")
	if deviceID == "" || sid == "" || ephRaw == "" {
		writeError(w, http.StatusBadRequest, "device, session and eph are required")
		return
	}
	s.mu.Lock()
	dev, devOK := s.devices[deviceID]
	s.mu.Unlock()
	if !devOK {
		writeError(w, http.StatusUnauthorized, "device not paired")
		return
	}
	ephPub, err := protocol.DecodeKey(ephRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ephemeral key")
		return
	}
	devPub, err := dev.PublicKeyBytes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "device key unavailable")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	})
	tr := &wsTransport{conn: conn, done: make(chan struct{})}
	if err := s.attachViewer(tr, deviceID, sid, ephPub, dev.Curve, devPub); err != nil {
		_ = tr.Close()
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	go wsPingLoop(conn, tr.done)
}

// attachViewer performs the E2EE handshake for a viewer (local or relay) and
// starts event streaming. devicePubKey is the viewer's paired identity key.
func (s *Server) attachViewer(tr viewerTransport, deviceID, sid string, ephPub []byte, curve protocol.Curve, devicePubKey []byte) error {
	s.mu.Lock()
	sess, sessOK := s.sessions[sid]
	s.mu.Unlock()
	if !sessOK {
		return fmt.Errorf("session not found")
	}
	identity, err := s.keys.Identity(curve)
	if err != nil {
		return fmt.Errorf("server identity unavailable: %w", err)
	}
	deviceSecret, err := protocol.NewDeviceSecret(identity, devicePubKey)
	if err != nil {
		return fmt.Errorf("device secret failed: %w", err)
	}
	serverEph, err := protocol.GenerateKeyPair(curve)
	if err != nil {
		return fmt.Errorf("ephemeral key failed: %w", err)
	}
	ephSecret, err := protocol.ECDH(serverEph, ephPub)
	if err != nil {
		return fmt.Errorf("ephemeral exchange failed: %w", err)
	}
	key, err := protocol.DeriveSessionKey(deviceSecret, ephSecret, sid)
	if err != nil {
		return fmt.Errorf("session key derivation failed: %w", err)
	}
	c := &client{
		deviceID:  deviceID,
		session:   sess,
		key:       key,
		transport: tr,
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		log:       s.log,
	}
	s.log.Printf("viewer connect device=%s session=%s curve=%s", deviceID, sid, curve)
	sess.addClient(c)
	hello := protocol.Hello{
		V:            1,
		Kind:         "hello",
		SessionID:    sid,
		ServerEphPub: protocol.EncodeKey(serverEph.PublicKey),
	}
	helloData, _ := json.Marshal(hello)
	c.sendRaw(helloData)
	replay := sess.snapshotLast(historyReplayLimit)
	s.log.Printf("hello queued device=%s session=%s replay=%d", deviceID, sid, len(replay))
	for _, ev := range replay {
		c.sendEvent(ev)
	}
	go c.writeLoop()
	go c.readLoop(s)
	return nil
}

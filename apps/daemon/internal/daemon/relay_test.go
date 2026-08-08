package daemon

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
)

// When the relay reports that the same host credentials connected elsewhere,
// the daemon must stop auto-reconnecting instead of kick-looping with the
// other daemon (#169).
func TestRelayClientStopsReconnectingOnSuperseded(t *testing.T) {
	var dials atomic.Int32
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		dials.Add(1)
		data, _ := json.Marshal(relayFrame{Kind: protocol.RelayFrameSuperseded})
		_ = conn.WriteMessage(websocket.TextMessage, data)
		time.Sleep(100 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)

	c := newRelayClient("ws"+strings.TrimPrefix(srv.URL, "http"), "h1", "s1",
		log.New(io.Discard, "", 0), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { c.run(ctx); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("run did not return after superseded")
	}
	// The reconnect interval is 3s; wait past it and assert no second dial.
	time.Sleep(3500 * time.Millisecond)
	if n := dials.Load(); n != 1 {
		t.Fatalf("expected no reconnect after superseded, got %d dials", n)
	}
}

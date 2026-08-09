//go:build !windows

package daemon

import (
	"context"
	"encoding/base64"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/packages/protocol"
	"golang.org/x/sys/unix"
)

type ptyTestTerm struct {
	m        *os.File
	lastCols uint16
	lastRows uint16
}

func (t *ptyTestTerm) Read(p []byte) (int, error)  { return t.m.Read(p) }
func (t *ptyTestTerm) Write(p []byte) (int, error) { return t.m.Write(p) }
func (t *ptyTestTerm) Close() error                { return t.m.Close() }
func (t *ptyTestTerm) Resize(c, r uint16) error    { t.lastCols, t.lastRows = c, r; return nil }

type ptyTestAdapter struct {
	id     string
	events chan protocol.Event
	master *os.File
	last   *ptyTestTerm
}

func (f *ptyTestAdapter) ID() string                    { return f.id }
func (f *ptyTestAdapter) Events() <-chan protocol.Event { return f.events }
func (f *ptyTestAdapter) Meta() protocol.SessionStartPayload {
	return protocol.SessionStartPayload{Name: "pty", CLI: "claude", Cwd: "/tmp"}
}
func (f *ptyTestAdapter) Start(_ context.Context) error  { return nil }
func (f *ptyTestAdapter) SendApproval(_, _ string) error { return nil }
func (f *ptyTestAdapter) SendPrompt(_ string) error      { return nil }
func (f *ptyTestAdapter) Alive() bool                    { return true }
func (f *ptyTestAdapter) Stop() error                    { return nil }
func (f *ptyTestAdapter) AttachPTY() (adapter.Terminal, error) {
	f.last = &ptyTestTerm{m: f.master}
	return f.last, nil
}

func TestPTYConsoleRoundTrip(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()

	// Disable echo/canonical so input written to the master does not bounce
	// back as an extra output frame.
	tio, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	tio.Lflag &^= unix.ECHO | unix.ICANON
	if err := unix.IoctlSetTermios(int(slave.Fd()), unix.TCSETS, tio); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, keys, dir, log.New(io.Discard, "", 0), nil)
	ad := &ptyTestAdapter{id: "pty-1", events: make(chan protocol.Event, 8), master: master}
	srv.sessions["pty-1"] = &session{
		id:      "pty-1",
		meta:    ad.Meta(),
		adapter: ad,
		events:  ad.events,
		status:  protocol.StatusRunning,
		clients: map[*client]struct{}{},
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") +
		"/api/sessions/pty-1/pty?token=" + cfg.LocalToken
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Client → PTY: bytes arrive on the slave.
	if err := conn.WriteJSON(map[string]any{"in": base64.StdEncoding.EncodeToString([]byte("hello"))}); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 16)
	if n, err := slave.Read(buf); err != nil || string(buf[:n]) != "hello" {
		t.Fatalf("slave got %q err=%v", buf[:n], err)
	}

	// PTY → client: writing to the slave produces an {"out"} frame.
	if _, err := slave.Write([]byte("world")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var msg struct {
			Out string `json:"out"`
		}
		if err := conn.SetReadDeadline(deadline); err != nil {
			t.Fatal(err)
		}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatal(err)
		}
		if msg.Out != "" {
			raw, err := base64.StdEncoding.DecodeString(msg.Out)
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != "world" {
				t.Fatalf("unexpected out: %q", raw)
			}
			break
		}
	}

	// Resize propagates to the terminal implementation.
	if err := conn.WriteJSON(map[string]any{"resize": map[string]uint16{"cols": 120, "rows": 40}}); err != nil {
		t.Fatal(err)
	}
	if got := srv.ptys["pty-1"]; got == nil {
		t.Fatal("console not registered")
	}
	deadline = time.Now().Add(5 * time.Second)
	for ad.last == nil || ad.last.lastCols != 120 || ad.last.lastRows != 40 {
		if time.Now().After(deadline) {
			t.Fatalf("resize not applied: cols=%d rows=%d", ad.last.lastCols, ad.last.lastRows)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

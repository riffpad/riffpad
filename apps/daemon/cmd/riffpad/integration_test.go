package main

// In-memory daemon integration smoke test (issue #282 testing strategy #5):
// spins a real daemon Server on a random port and drives the CLI command
// functions against its real handlers, so future refactors have stronger
// guardrails than httptest stubs.

import (
	"log"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
)

// newInMemDaemon spins a real daemon Server (no relay) on a random port and
// points the CLI helpers at its local token, so command functions exercise
// the real pairing/status/kill/session handlers end to end.
func newInMemDaemon(t *testing.T) (base string) {
	t.Helper()
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(os.Stderr, "[daemon] ", log.LstdFlags)
	srv := daemon.New(cfg, keys, dir, logger, daemon.DefaultFactory())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	httpSrv := &http.Server{Handler: srv.Handler()}
	t.Cleanup(func() { _ = httpSrv.Close() })
	go func() { _ = httpSrv.Serve(ln) }()
	withCliToken(t, cfg.LocalToken)
	return "http://" + ln.Addr().String()
}

func TestSessionsCmdAgainstInMemDaemon(t *testing.T) {
	base := newInMemDaemon(t)
	if err := sessionsCmd(base); err != nil {
		t.Fatalf("sessions failed: %v", err)
	}
}

func TestStatusCmdAgainstInMemDaemon(t *testing.T) {
	base := newInMemDaemon(t)
	if err := statusCmd(base); err != nil {
		t.Fatalf("status failed: %v", err)
	}
}

func TestKillCmdAgainstInMemDaemon(t *testing.T) {
	base := newInMemDaemon(t)
	if err := killCmd(base); err != nil {
		t.Fatalf("kill failed: %v", err)
	}
}

// TestPairCmdLocalAgainstInMemDaemon covers the full mint → decode → print
// path against the real pairing handler.
func TestPairCmdLocalAgainstInMemDaemon(t *testing.T) {
	base := newInMemDaemon(t)
	if err := pairCmd(base, []string{"--local"}); err != nil {
		t.Fatalf("pair --local failed: %v", err)
	}
}

// TestRunCmdAgainstInMemDaemon creates a demo session (no real CLI spawned)
// through the full createSession handler; runCmd prints the session URL and
// returns without attaching a TUI.
func TestRunCmdAgainstInMemDaemon(t *testing.T) {
	base := newInMemDaemon(t)
	if err := runCmd([]string{"--cli", "demo"}, base); err != nil {
		t.Fatalf("run demo failed: %v", err)
	}
}

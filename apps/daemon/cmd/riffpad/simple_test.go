package main

// Characterization tests for statusCmd / killCmd / logsCmd / authCmd, added
// before the #282 file split so the refactor cannot silently change them.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
)

// --- statusCmd ---

func TestStatusCmdPrintsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/status" {
			_, _ = w.Write([]byte(`{"version":"dev","port":8787,"sessions":0}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	if err := statusCmd(srv.URL); err != nil {
		t.Fatalf("status failed: %v", err)
	}
}

func TestStatusCmdUnreachable(t *testing.T) {
	err := statusCmd("http://127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "daemon not reachable") {
		t.Fatalf("expected not-reachable error, got %v", err)
	}
}

// --- killCmd ---

func TestKillCmdDecodesSessions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/killswitch" && r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"killed":true,"sessions":2}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	if err := killCmd(srv.URL); err != nil {
		t.Fatalf("kill failed: %v", err)
	}
}

func TestKillCmdUnreachable(t *testing.T) {
	err := killCmd("http://127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "daemon not reachable") {
		t.Fatalf("expected not-reachable error, got %v", err)
	}
}

// --- logsCmd ---

func TestLogsCmdTailsDaemonLog(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logs", "daemon.log"), []byte("line1\nline2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := logsCmd(dir); err != nil {
		t.Fatalf("logs failed: %v", err)
	}
}

func TestLogsCmdMissingLog(t *testing.T) {
	err := logsCmd(t.TempDir())
	if err == nil {
		t.Fatal("expected error when daemon.log is missing")
	}
}

// --- authCmd ---

func TestAuthCmdNotLoggedIn(t *testing.T) {
	if err := authCmd(t.TempDir()); err != nil {
		t.Fatalf("auth with empty config: %v", err)
	}
}

func TestAuthCmdInvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	if err := config.Update(dir, func(c *config.Config) {
		c.RelayToken = "stale"
		c.RelayUser = "alice"
		c.RelayURL = "ws://" + strings.TrimPrefix(srv.URL, "http://")
	}); err != nil {
		t.Fatal(err)
	}
	if err := authCmd(dir); err != nil {
		t.Fatalf("auth invalid token: %v", err)
	}
}

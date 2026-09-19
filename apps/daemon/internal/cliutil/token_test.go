package cliutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
)

// stubToken stubs the CLI's local token for the duration of a test.
func stubToken(t *testing.T, token string) {
	t.Helper()
	old := cliToken
	SetToken(token)
	t.Cleanup(func() { SetToken(old) })
}

func TestDaemonDoAttachesLocalToken(t *testing.T) {
	stubToken(t, "test-token")
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(daemon.LocalTokenHeader)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	resp, err := DaemonDo(nil, http.MethodGet, srv.URL+"/api/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got != "test-token" {
		t.Fatalf("expected %s header %q, got %q", daemon.LocalTokenHeader, "test-token", got)
	}
}

// The CLI health check must authenticate too: a daemon that requires the
// local token is "reachable" only when the CLI presents it.
func TestReachableWithTokenRequiringDaemon(t *testing.T) {
	stubToken(t, "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(daemon.LocalTokenHeader) != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	if !Reachable(srv.URL) {
		t.Fatal("reachable should succeed with the local token")
	}

	stubToken(t, "wrong-token")
	if Reachable(srv.URL) {
		t.Fatal("reachable should fail with the wrong token")
	}
}

// TestDaemonBaseFollowsConfiguredPort pins #316: the CLI must talk to the port
// the daemon was actually started on, not a hardcoded 8787. RIFFPAD_URL stays
// the explicit override.
func TestDaemonBaseFollowsConfiguredPort(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"port":18787,"localToken":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	oldDir := cliDataDir
	SetDataDir(dir)
	t.Cleanup(func() { cliDataDir = oldDir })

	t.Setenv("RIFFPAD_URL", "")
	if got := DaemonBase(); got != "http://127.0.0.1:18787" {
		t.Errorf("DaemonBase() = %q, want the configured port", got)
	}

	t.Setenv("RIFFPAD_URL", "http://127.0.0.1:9999/")
	if got := DaemonBase(); got != "http://127.0.0.1:9999" {
		t.Errorf("DaemonBase() with RIFFPAD_URL = %q, want the override (trailing slash trimmed)", got)
	}
}

// TestDaemonBaseDefaultsWithoutConfig covers the untouched-machine case.
func TestDaemonBaseDefaultsWithoutConfig(t *testing.T) {
	oldDir := cliDataDir
	SetDataDir(t.TempDir())
	t.Cleanup(func() { cliDataDir = oldDir })
	t.Setenv("RIFFPAD_URL", "")
	if got := DaemonBase(); got != "http://127.0.0.1:8787" {
		t.Errorf("DaemonBase() = %q, want the default port", got)
	}
}

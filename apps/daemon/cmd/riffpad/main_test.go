package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
)

func TestOAuthDeviceLoginRetriesTransientError(t *testing.T) {
	oldOpen, oldRestart := openBrowserFn, restartDaemonFn
	openBrowserFn = func(string) {}
	restartDaemonFn = func(string) {}
	t.Cleanup(func() { openBrowserFn, restartDaemonFn = oldOpen, oldRestart })

	var polls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/oauth/device":
			_, _ = w.Write([]byte(`{"userCode":"ABC123","verificationURL":"http://x/device?code=ABC123","expiresIn":60,"interval":1}`))
		case "/api/auth/oauth/device/poll":
			n := atomic.AddInt32(&polls, 1)
			if n == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if n == 2 {
				_, _ = w.Write([]byte(`{"pending":true}`))
				return
			}
			_, _ = w.Write([]byte(`{"pending":false,"token":"tok","username":"u1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	if err := oauthDeviceLogin(srv.URL, "wss://relay.test", dir); err != nil {
		t.Fatalf("device login failed: %v", err)
	}
	if atomic.LoadInt32(&polls) < 3 {
		t.Fatalf("expected retries on transient error, got %d polls", atomic.LoadInt32(&polls))
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RelayToken != "tok" || cfg.RelayUser != "u1" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestSyncHostCredsAccountSwitchClearsLocally(t *testing.T) {
	cfg := &config.Config{HostID: "host1", HostSecret: "secret1", RelayUser: "alice"}
	// The relay is deliberately unreachable; an account switch must still
	// clear stale host credentials so a successful authorization is never
	// thrown away by a transient network failure.
	if err := syncHostCreds("http://127.0.0.1:1", "tok", "bob", cfg); err != nil {
		t.Fatalf("syncHostCreds failed: %v", err)
	}
	if cfg.HostID != "" || cfg.HostSecret != "" {
		t.Fatalf("stale host credentials not cleared: %+v", cfg)
	}
}

func TestSyncHostCredsSameUserKeepsHost(t *testing.T) {
	cfg := &config.Config{HostID: "host1", HostSecret: "secret1", RelayUser: "alice"}
	// Same-account re-login while the relay is unreachable keeps the host
	// credentials; the ownership check is best-effort.
	if err := syncHostCreds("http://127.0.0.1:1", "tok", "alice", cfg); err != nil {
		t.Fatalf("syncHostCreds failed: %v", err)
	}
	if cfg.HostID != "host1" || cfg.HostSecret != "secret1" {
		t.Fatalf("host credentials should be kept: %+v", cfg)
	}
}

// withCliToken stubs the CLI's local token for the duration of a test.
func withCliToken(t *testing.T, token string) {
	t.Helper()
	cliutil.SetToken(token)
	t.Cleanup(func() { cliutil.SetToken("") })
}

// The CLI health check must authenticate too: a daemon that requires the
// local token is "reachable" only when the CLI presents it.

package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
)

func fakeDaemon(t *testing.T, ready *atomic.Bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			http.NotFound(w, r)
			return
		}
		if ready.Load() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	return srv
}

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

func TestEnsureDaemonSkipsWhenReachable(t *testing.T) {
	ready := &atomic.Bool{}
	ready.Store(true)
	srv := fakeDaemon(t, ready)
	calls := 0
	startDaemonFn = func(string, string) error { calls++; return nil }
	defer func() { startDaemonFn = daemonStart }()

	if err := ensureDaemon(srv.URL, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("expected no daemon start when reachable, got %d", calls)
	}
}

func TestEnsureDaemonStartsLazily(t *testing.T) {
	ready := &atomic.Bool{}
	srv := fakeDaemon(t, ready)
	calls := 0
	startDaemonFn = func(base, dataDir string) error {
		calls++
		if base != srv.URL {
			t.Fatalf("unexpected base %q", base)
		}
		ready.Store(true)
		return nil
	}
	defer func() { startDaemonFn = daemonStart }()

	if err := ensureDaemon(srv.URL, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected one daemon start, got %d", calls)
	}
}

func TestEnsureDaemonConcurrentSingleStart(t *testing.T) {
	ready := &atomic.Bool{}
	srv := fakeDaemon(t, ready)
	var mu sync.Mutex
	calls := 0
	startDaemonFn = func(string, string) error {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(150 * time.Millisecond) // simulate slow startup
		ready.Store(true)
		return nil
	}
	defer func() { startDaemonFn = daemonStart }()

	dataDir := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := ensureDaemon(srv.URL, dataDir); err != nil {
				t.Errorf("ensureDaemon: %v", err)
			}
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("expected exactly one daemon start under concurrency, got %d", calls)
	}
}

// withCliToken stubs the CLI's local token for the duration of a test.
func withCliToken(t *testing.T, token string) {
	t.Helper()
	oldToken, oldDir, oldOnce := cliToken, cliDataDir, tokenOnce
	cliToken, cliDataDir, tokenOnce = token, "", sync.Once{}
	t.Cleanup(func() { cliToken, cliDataDir, tokenOnce = oldToken, oldDir, oldOnce })
}

func TestDaemonDoAttachesLocalToken(t *testing.T) {
	withCliToken(t, "test-token")
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(daemon.LocalTokenHeader)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	resp, err := daemonDo(nil, http.MethodGet, srv.URL+"/api/status", nil)
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
	withCliToken(t, "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(daemon.LocalTokenHeader) != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	if !reachable(srv.URL) {
		t.Fatal("reachable should succeed with the local token")
	}

	withCliToken(t, "wrong-token")
	if reachable(srv.URL) {
		t.Fatal("reachable should fail with the wrong token")
	}
}

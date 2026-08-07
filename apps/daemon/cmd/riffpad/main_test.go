package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
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

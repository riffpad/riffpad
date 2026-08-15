package commands

// Daemon lifecycle tests (lazy start, single starter under the file lock,
// restart routing via systemd/Windows task, force-kill safety), moved from
// cmd/riffpad during the #282 split.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
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

func TestEnsureDaemonSkipsWhenReachable(t *testing.T) {
	ready := &atomic.Bool{}
	ready.Store(true)
	srv := fakeDaemon(t, ready)
	calls := 0
	startDaemonFn = func(string, string) error { calls++; return nil }
	defer func() { startDaemonFn = daemonStart }()

	if err := EnsureDaemon(srv.URL, t.TempDir()); err != nil {
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

	if err := EnsureDaemon(srv.URL, t.TempDir()); err != nil {
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
			if err := EnsureDaemon(srv.URL, dataDir); err != nil {
				t.Errorf("EnsureDaemon: %v", err)
			}
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("expected exactly one daemon start under concurrency, got %d", calls)
	}
}

func TestDaemonRestartPrefersSystemd(t *testing.T) {
	oldActive, oldRestart := systemdActiveFn, systemdRestartFn
	systemdActiveFn = func() (bool, error) { return true, nil }
	var calls int
	systemdRestartFn = func() error { calls++; return nil }
	t.Cleanup(func() { systemdActiveFn, systemdRestartFn = oldActive, oldRestart })

	if err := daemonRestart("http://127.0.0.1:1", t.TempDir()); err != nil {
		t.Fatalf("restart failed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one systemctl restart, got %d", calls)
	}
}

func TestDaemonRestartRequiresRunningDaemon(t *testing.T) {
	oldActive, oldRestart := systemdActiveFn, systemdRestartFn
	systemdActiveFn = func() (bool, error) { return false, nil }
	systemdRestartFn = func() error {
		t.Fatal("must not call systemctl when service is inactive")
		return nil
	}
	t.Cleanup(func() { systemdActiveFn, systemdRestartFn = oldActive, oldRestart })

	err := daemonRestart("http://127.0.0.1:1", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "daemon is not running") {
		t.Fatalf("expected not-running error, got %v", err)
	}
}

func TestRestartViaWindowsTaskStopsThenRuns(t *testing.T) {
	oldRun := windowsTaskRunFn
	var runs, shutdowns int
	down := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/status":
			if down {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		case "/api/shutdown":
			shutdowns++
			down = true
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	windowsTaskRunFn = func() error {
		runs++
		down = false // the scheduled task spawns the daemon
		return nil
	}
	t.Cleanup(func() { windowsTaskRunFn = oldRun })

	if err := restartViaWindowsTask(srv.URL, t.TempDir()); err != nil {
		t.Fatalf("restart failed: %v", err)
	}
	if runs != 1 || shutdowns != 1 {
		t.Fatalf("runs=%d shutdowns=%d, want 1/1", runs, shutdowns)
	}
}

func TestRestartViaWindowsTaskSkipsStopWhenNotRunning(t *testing.T) {
	oldRun := windowsTaskRunFn
	var runs, shutdowns int
	down := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/status":
			if down {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		case "/api/shutdown":
			shutdowns++
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	windowsTaskRunFn = func() error {
		runs++
		down = false
		return nil
	}
	t.Cleanup(func() { windowsTaskRunFn = oldRun })

	if err := restartViaWindowsTask(srv.URL, t.TempDir()); err != nil {
		t.Fatalf("restart failed: %v", err)
	}
	if runs != 1 || shutdowns != 0 {
		t.Fatalf("runs=%d shutdowns=%d, want 1/0", runs, shutdowns)
	}
}

// --- daemon stop force-kill safety (issue #174 #9) ---

func TestForceKillDaemonMissingPidFile(t *testing.T) {
	if err := forceKillDaemon(t.TempDir()); err == nil {
		t.Fatal("expected error when daemon.pid is missing")
	}
}

func TestForceKillDaemonRefusesForeignProcess(t *testing.T) {
	// pid 1 is never a riffpad daemon; a stale/recycled pid file must not get
	// an unrelated process killed (#174).
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "daemon.pid"), []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := forceKillDaemon(dir); err == nil || !strings.Contains(err.Error(), "not a riffpad daemon") {
		t.Fatalf("expected refusal to kill pid 1, got %v", err)
	}
}

// cliutil.Reachable must present the local token (regression guard for the
// lazy-start path, which decides "already running" from it).
func TestEnsureDaemonUsesAuthenticatedHealthCheck(t *testing.T) {
	// Token-requiring daemon: without the right token it is unreachable and
	// EnsureDaemon would try to start another instance.
	cliutil.SetToken("right-token")
	t.Cleanup(func() { cliutil.SetToken("") })
	ready := &atomic.Bool{}
	ready.Store(true)
	srv := fakeDaemon(t, ready)
	calls := 0
	startDaemonFn = func(string, string) error { calls++; return nil }
	defer func() { startDaemonFn = daemonStart }()

	if err := EnsureDaemon(srv.URL, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("authenticated daemon must be detected as running, got %d starts", calls)
	}
}

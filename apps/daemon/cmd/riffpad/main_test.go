package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestPairWithRetryRecoversFromHostOffline(t *testing.T) {
	oldDelay := pairRetryDelay
	pairRetryDelay = time.Millisecond
	t.Cleanup(func() { pairRetryDelay = oldDelay })

	var calls int32
	res, err := pairWithRetry(func() (pairingResult, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return pairingResult{Error: "host offline"}, nil
		}
		return pairingResult{Code: "ABCDEF", URL: "https://app.riffpad.ai/pair?code=ABCDEF"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != "ABCDEF" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 pairing attempts, got %d", got)
	}
}

func TestPairWithRetryFailsFastOnNonTransientError(t *testing.T) {
	oldDelay := pairRetryDelay
	pairRetryDelay = time.Millisecond
	t.Cleanup(func() { pairRetryDelay = oldDelay })

	var calls int32
	_, err := pairWithRetry(func() (pairingResult, error) {
		atomic.AddInt32(&calls, 1)
		return pairingResult{Error: "host not found"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 attempt for host not found, got %d", got)
	}

	calls = 0
	_, _ = pairWithRetry(func() (pairingResult, error) {
		atomic.AddInt32(&calls, 1)
		return pairingResult{Error: "登录已过期", ErrorCode: "relay_auth_expired"}, nil
	})
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 attempt for auth expiry, got %d", got)
	}
}

func TestPairWithRetryStopsAtDeadline(t *testing.T) {
	oldDelay, oldWait := pairRetryDelay, pairRetryMaxWait
	pairRetryDelay = time.Millisecond
	pairRetryMaxWait = 15 * time.Millisecond
	t.Cleanup(func() { pairRetryDelay, pairRetryMaxWait = oldDelay, oldWait })

	var calls int32
	res, err := pairWithRetry(func() (pairingResult, error) {
		atomic.AddInt32(&calls, 1)
		return pairingResult{Error: "host offline"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Error != "host offline" {
		t.Fatalf("expected final host offline error, got %+v", res)
	}
	if got := atomic.LoadInt32(&calls); got < 5 {
		t.Fatalf("expected several retries before deadline, got %d", got)
	}
}

func TestRequestPairingDecodesErrorAndStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"host offline"}`))
	}))
	t.Cleanup(srv.Close)

	res, err := requestPairing(srv.URL + "/api/pairings")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusBadGateway || res.Error != "host offline" {
		t.Fatalf("unexpected pairing result: %+v", res)
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

// --- attach/detach hook merging (issue #168) ---

// setupAttachEnv points HOME at a temp dir and stubs the local token so
// attachCmd writes to a throwaway ~/.claude/settings.json.
func setupAttachEnv(t *testing.T) (home string, base string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	withCliToken(t, "")
	ready := &atomic.Bool{}
	ready.Store(true)
	return home, fakeDaemon(t, ready).URL
}

func userHookEntry(command string) map[string]any {
	return map[string]any{
		"matcher": "Bash",
		"hooks":   []any{map[string]any{"type": "command", "command": command}},
	}
}

func riffpadHookEntry(path string) map[string]any {
	return map[string]any{
		"matcher": "",
		"hooks": []any{map[string]any{
			"type": "http",
			"url":  "http://127.0.0.1:8787/hooks/claude/" + path,
		}},
	}
}

func writeClaudeSettings(t *testing.T, home string, settings map[string]any) {
	t.Helper()
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readClaudeSettings(t *testing.T, home string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	settings := map[string]any{}
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatal(err)
	}
	return settings
}

// countHooks classifies every entry under settings["hooks"] as riffpad-owned
// (URL path under /hooks/claude/) or user-owned.
func countHooks(t *testing.T, settings map[string]any) (riffpad, user int) {
	t.Helper()
	hooks, _ := settings["hooks"].(map[string]any)
	for _, list := range hooks {
		entries, _ := list.([]any)
		for _, e := range entries {
			entry, _ := e.(map[string]any)
			nested, _ := entry["hooks"].([]any)
			isRiffpad := false
			for _, h := range nested {
				hm, _ := h.(map[string]any)
				if raw, _ := hm["url"].(string); strings.Contains(raw, "/hooks/claude/") {
					isRiffpad = true
				}
			}
			if isRiffpad {
				riffpad++
			} else {
				user++
			}
		}
	}
	return riffpad, user
}

func TestAttachPreservesUserHooks(t *testing.T) {
	home, base := setupAttachEnv(t)
	writeClaudeSettings(t, home, map[string]any{
		"model": "opus",
		"hooks": map[string]any{
			"PreToolUse":   []any{userHookEntry("my-formatter")},
			"Notification": []any{userHookEntry("my-notifier")},
		},
	})

	if err := attachCmd(base); err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("expected deprecated error, got %v", err)
	}
	settings := readClaudeSettings(t, home)
	if settings["model"] != "opus" {
		t.Fatalf("unrelated setting lost: %+v", settings)
	}
	riffpad, user := countHooks(t, settings)
	if riffpad != 0 || user != 2 {
		t.Fatalf("disabled attach must not touch settings: riffpad=%d user=%d", riffpad, user)
	}
}

func TestDetachRemovesOnlyRiffpadHooks(t *testing.T) {
	home, _ := setupAttachEnv(t)
	legacyHooks := map[string]any{
		"MessageDisplay":    []any{riffpadHookEntry("message-display")},
		"Notification":      []any{riffpadHookEntry("notification")},
		"PermissionRequest": []any{riffpadHookEntry("permission")},
		"PostToolUse":       []any{riffpadHookEntry("post-tool-use")},
		"PreToolUse":        []any{riffpadHookEntry("pre-tool-use")},
		"SessionEnd":        []any{riffpadHookEntry("session-end")},
		"SessionStart":      []any{riffpadHookEntry("session-start")},
		"UserPromptSubmit":  []any{riffpadHookEntry("user-prompt-submit")},
	}
	writeClaudeSettings(t, home, map[string]any{
		"model": "opus",
		"hooks": map[string]any{
			"PreToolUse":   []any{userHookEntry("my-formatter"), riffpadHookEntry("pre-tool-use")},
			"Notification": []any{userHookEntry("my-notifier")},
			"PostToolUse":  legacyHooks["PostToolUse"],
			"SessionStart": legacyHooks["SessionStart"],
		},
	})

	if err := detachCmd(); err != nil {
		t.Fatal(err)
	}
	settings := readClaudeSettings(t, home)
	if settings["model"] != "opus" {
		t.Fatalf("unrelated setting lost: %+v", settings)
	}
	riffpad, user := countHooks(t, settings)
	if riffpad != 0 {
		t.Fatalf("expected no riffpad entries after detach, got %d", riffpad)
	}
	if user != 2 {
		t.Fatalf("expected both user hook entries kept, got %d", user)
	}
}

// --- run command error propagation (issue #174 #1) ---

func TestRunCmdPropagatesDaemonError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": `unsupported cli "claud"`})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	err := runCmd([]string{"--cli", "claud"}, srv.URL)
	if err == nil || !strings.Contains(err.Error(), `unsupported cli "claud"`) {
		t.Fatalf("expected unsupported cli error, got %v", err)
	}
}

func TestRunCmdFailsOnErrorStatusWithoutMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	err := runCmd(nil, srv.URL)
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected HTTP status error, got %v", err)
	}
}

func TestRunCmdSuccess(t *testing.T) {
	var gotBinary string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Binary string `json:"binary"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotBinary = req.Binary
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "s1", "url": "http://x/s1", "cli": "sh"})
	}))
	t.Cleanup(srv.Close)

	if err := runCmd([]string{"--cli", "sh"}, srv.URL); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	want, err := exec.LookPath("sh")
	if err != nil {
		t.Fatal(err)
	}
	if gotBinary != want {
		t.Fatalf("expected resolved binary %q in request, got %q", want, gotBinary)
	}
}

func TestRunCmdAcceptsPositionalCLI(t *testing.T) {
	var gotCLI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CLI string `json:"cli"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotCLI = req.CLI
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "s1", "url": "http://x/s1", "cli": req.CLI})
	}))
	t.Cleanup(srv.Close)

	if err := runCmd([]string{"sh"}, srv.URL); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if gotCLI != "sh" {
		t.Fatalf("expected positional cli sh, got %q", gotCLI)
	}
}

func TestRunCmdRejectsMultiplePositionals(t *testing.T) {
	err := runCmd([]string{"codex", "claude"}, "http://127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("expected unexpected-arguments error, got %v", err)
	}
}

func TestAttachCmdDeprecated(t *testing.T) {
	err := attachCmd("http://127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("expected deprecated error, got %v", err)
	}
}

func TestPairCmdRejectsLocalWithoutFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "ABC123", "url": "http://127.0.0.1:8787/?pair=ABC123", "local": true,
		})
	}))
	t.Cleanup(srv.Close)

	err := pairCmd(srv.URL, nil)
	if err == nil || !strings.Contains(err.Error(), "pair --local") {
		t.Fatalf("expected login-required error mentioning --local, got %v", err)
	}
}

func TestPairCmdAllowsLocalWithFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "ABC123", "url": "http://127.0.0.1:8787/?pair=ABC123", "local": true,
		})
	}))
	t.Cleanup(srv.Close)

	if err := pairCmd(srv.URL, []string{"--local"}); err != nil {
		t.Fatalf("pair --local failed: %v", err)
	}
}

// TestPairCmdLocalSendsQuery: `riffpad pair --local` must ask the daemon for a
// local-only code by passing ?local=1, so a relay-connected daemon mints a code
// the embedded 8787 UI can claim instead of a cloud code.
func TestPairCmdLocalSendsQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "ABC123", "url": "http://127.0.0.1:8787/?pair=ABC123", "local": true,
		})
	}))
	t.Cleanup(srv.Close)

	if err := pairCmd(srv.URL, []string{"--local"}); err != nil {
		t.Fatalf("pair --local failed: %v", err)
	}
	if !strings.Contains(gotQuery, "local=1") {
		t.Fatalf("expected local=1 in pairing request query, got %q", gotQuery)
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

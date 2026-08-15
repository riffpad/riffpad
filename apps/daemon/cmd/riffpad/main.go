// Command riffpad is the single Riffpad binary: user-facing CLI plus the
// hidden `_daemon` subcommand that runs the background daemon.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	"golang.org/x/term"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
	"github.com/riffpad/riffpad/apps/daemon/internal/commands"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/console"
	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
	"github.com/riffpad/riffpad/apps/daemon/internal/i18n"
	"github.com/riffpad/riffpad/apps/daemon/internal/logging"
	"github.com/riffpad/riffpad/apps/daemon/internal/version"
)

const updateRepo = "riffpad/riffpad"

// updateAPIBase and updateDownloadBase point the updater at GitHub; they are
// vars so tests can serve a fake release server.
var (
	updateAPIBase      = "https://api.github.com/repos"
	updateDownloadBase = "https://github.com/" + updateRepo + "/releases/latest/download/"
)

// osExecutableFn and systemctlFn are indirections so update/setup tests can
// run against a throwaway binary and without touching the real systemd user
// session.
var (
	osExecutableFn = os.Executable
	systemctlFn    = func(args ...string) ([]byte, error) {
		return exec.Command("systemctl", append([]string{"--user"}, args...)...).CombinedOutput()
	}
)

// pairRetryDelay and pairRetryMaxWait control the transient "host offline"
// retry in mintPairingCode: right after the daemon starts, its relay
// WebSocket registration can lag the local HTTP endpoint by a few seconds.
var (
	pairRetryDelay   = time.Second
	pairRetryMaxWait = 10 * time.Second
)

// t is the active language bundle, initialized in main from --lang.
var t = i18n.New(i18n.DefaultLang)

// localToken and daemonDo delegate to internal/cliutil, which owns the
// daemon's local API token resolution shared by all CLI commands.
func localToken() string { return cliutil.LocalToken() }
func daemonDo(client *http.Client, method, url string, body io.Reader) (*http.Response, error) {
	return cliutil.DaemonDo(client, method, url, body)
}

func main() {
	langFlag, args := extractLangFlag(os.Args[1:])
	t = i18n.New(i18n.Detect(langFlag))
	commands.SetBundle(t)
	console.SetBundle(t)
	os.Args = append([]string{os.Args[0]}, args...)
	// Corrupted state files are backed up and rebuilt automatically (#172);
	// warn instead of dying at startup. runDaemon overrides this to also log.
	config.OnHeal = func(h config.Heal) {
		fmt.Fprintln(os.Stderr, t.T("config_file_healed", h.Path, h.Backup))
		if h.Kind == "keys" {
			fmt.Fprintln(os.Stderr, t.T("keys_regenerated_warn"))
		}
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if os.Args[1] == "_daemon" {
		os.Exit(runDaemon(os.Args[2:]))
	}
	base := os.Getenv("RIFFPAD_URL")
	if base == "" {
		base = "http://127.0.0.1:8787"
	}
	dataDir := os.Getenv("RIFFPAD_DIR")
	if dataDir == "" {
		d, err := config.DefaultDataDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, t.T("resolve_data_dir", err))
			os.Exit(1)
		}
		dataDir = d
	}
	cliutil.SetDataDir(dataDir)

	var err error
	switch os.Args[1] {
	case "daemon":
		err = daemonCmd(os.Args[2:], base, dataDir)
	case "status":
		err = statusCmd(base)
	case "pair":
		err = withDaemon(func() error { return pairCmd(base, os.Args[2:]) }, base, dataDir)
	case "sessions":
		err = withDaemon(func() error { return sessionsCmd(base) }, base, dataDir)
	case "run":
		err = withDaemon(func() error { return runCmd(os.Args[2:], base) }, base, dataDir)
	case "logs":
		err = logsCmd(dataDir)
	case "attach":
		err = withDaemon(func() error { return attachCmd(base) }, base, dataDir)
	case "detach":
		err = detachCmd()
	case "auth":
		err = authCmd(dataDir)
	case "login":
		err = loginCmd(os.Args[2:], dataDir)
	case "logout":
		err = logoutCmd(dataDir)
	case "relay":
		err = relayCmd(os.Args[2:], dataDir)
	case "setup":
		err = setupCmd(os.Args[2:], dataDir)
	case "kill":
		err = killCmd(base)
	case "update":
		err = updateCmd(os.Args[2:], dataDir)
	case "version":
		fmt.Println("riffpad", version.Version)
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "riffpad:", err)
		os.Exit(1)
	}
}

// runDaemon is the hidden `riffpad _daemon` entry point used by daemon start
// and systemd. Keeping the daemon inside the same binary makes installation a
// single-file affair.
func runDaemon(args []string) int {
	fs := flag.NewFlagSet("_daemon", flag.ExitOnError)
	dataDir := fs.String("data-dir", "", "daemon data directory (default ~/.config/riffpad)")
	_ = fs.Parse(args)
	enrichDaemonPath()
	dir := *dataDir
	if dir == "" {
		var err error
		dir, err = config.DefaultDataDir()
		if err != nil {
			log.Printf("resolve data dir: %v", err)
			return 1
		}
	}
	logger, closer, err := logging.New(dir)
	if err != nil {
		log.Printf("init logging: %v", err)
		return 1
	}
	defer closer.Close()
	// Route self-heal warnings (#172) through the daemon log as well.
	config.OnHeal = func(h config.Heal) {
		logger.Printf("%s", t.T("config_file_healed", h.Path, h.Backup))
		if h.Kind == "keys" {
			logger.Printf("%s", t.T("keys_regenerated_warn"))
		}
	}

	cfg, err := config.Load(dir)
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		logger.Fatalf("load keys: %v", err)
	}
	srv := daemon.New(cfg, keys, dir, logger, daemon.DefaultFactory())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("server: %v", err)
		}
		stop()
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown: %v", err)
	}
	logger.Printf("daemon stopped")
	return 0
}

// enrichDaemonPath appends common user bin directories to PATH so the daemon
// can spawn coding CLIs (codex/kimi/claude) even when it was started by
// systemd with a minimal PATH. Only existing directories are added; the
// authoritative PATH captured by `riffpad setup` and per-command binary paths
// passed by `riffpad run` take precedence because they are earlier in PATH.
func enrichDaemonPath() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	extra := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".codex", "bin"),
		filepath.Join(home, ".kimi-code", "bin"),
		filepath.Join(home, ".opencode", "bin"),
		filepath.Join(home, ".cargo", "bin"),
		filepath.Join(home, "go", "bin"),
		filepath.Join(home, ".bun", "bin"),
		filepath.Join(home, ".npm-global", "bin"),
	}
	seen := map[string]bool{}
	parts := make([]string, 0, 16)
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p != "" && !seen[p] {
			seen[p] = true
			parts = append(parts, p)
		}
	}
	for _, d := range extra {
		if seen[d] {
			continue
		}
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			parts = append(parts, d)
			seen[d] = true
		}
	}
	os.Setenv("PATH", strings.Join(parts, string(os.PathListSeparator)))
}

func usage() {
	fmt.Fprintln(os.Stderr, t.T("usage"))
}

// extractLangFlag pulls a global --lang/-lang value out of args so it works
// before any subcommand, regardless of position.
func extractLangFlag(args []string) (lang string, rest []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--lang" || a == "-lang":
			if i+1 < len(args) {
				lang = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--lang="):
			lang = strings.TrimPrefix(a, "--lang=")
		case strings.HasPrefix(a, "-lang="):
			lang = strings.TrimPrefix(a, "-lang=")
		default:
			rest = append(rest, a)
		}
	}
	return lang, rest
}

func daemonCmd(args []string, base, dataDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s", t.T("usage_daemon"))
	}
	switch args[0] {
	case "start":
		return daemonStart(base, dataDir)
	case "stop":
		return daemonStop(base, dataDir)
	case "restart":
		return daemonRestart(base, dataDir)
	default:
		return fmt.Errorf("%s", t.T("usage_daemon"))
	}
}

// systemdActiveFn reports whether the riffpad systemd user service is active.
// Indirection lets tests simulate both managed and unmanaged daemons.
var systemdActiveFn = func() (bool, error) {
	out, err := exec.Command("systemctl", "--user", "is-active", "riffpad.service").CombinedOutput()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "active", nil
}

// systemdRestartFn restarts the riffpad systemd user service.
var systemdRestartFn = func() error {
	return exec.Command("systemctl", "--user", "restart", "riffpad.service").Run()
}

// windowsTaskExistsFn reports whether the RiffpadDaemon scheduled task
// (created by `riffpad setup` / install.ps1) exists.
var windowsTaskExistsFn = func() (bool, error) {
	if err := exec.Command("schtasks", "/Query", "/TN", "RiffpadDaemon").Run(); err != nil {
		return false, err
	}
	return true, nil
}

// windowsTaskRunFn starts the RiffpadDaemon scheduled task so Task Scheduler
// stays the bookkeeper for the daemon process.
var windowsTaskRunFn = func() error {
	return exec.Command("schtasks", "/Run", "/TN", "RiffpadDaemon").Run()
}

// daemonRestart restarts the daemon. When the systemd user service is active,
// it goes through systemctl so systemd stays the single source of truth;
// on Windows, an existing RiffpadDaemon scheduled task is restarted through
// Task Scheduler; otherwise it stops and starts the background process. A
// daemon that is not running is an error.
func daemonRestart(base, dataDir string) error {
	switch runtime.GOOS {
	case "linux":
		if active, err := systemdActiveFn(); err == nil && active {
			return restartViaSystemd()
		}
	case "windows":
		if ok, err := windowsTaskExistsFn(); err == nil && ok {
			return restartViaWindowsTask(base, dataDir)
		}
	}
	if !reachable(base) {
		return fmt.Errorf("%s", t.T("daemon_not_running"))
	}
	if err := daemonStop(base, dataDir); err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_restart_failed"), err)
	}
	if err := daemonStart(base, dataDir); err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_restart_failed"), err)
	}
	fmt.Println(t.T("daemon_restarted"))
	return nil
}

func restartViaSystemd() error {
	if err := systemdRestartFn(); err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_restart_failed"), err)
	}
	fmt.Println(t.T("daemon_restarted"))
	return nil
}

func restartViaWindowsTask(base, dataDir string) error {
	// Stop the current daemon first (if any) so the task-started process can
	// bind the port; Task Scheduler then owns the new instance.
	if reachable(base) {
		if err := daemonStop(base, dataDir); err != nil {
			return fmt.Errorf("%s: %w", t.T("daemon_restart_failed"), err)
		}
	}
	if err := windowsTaskRunFn(); err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_restart_failed"), err)
	}
	if !waitReachable(base, 2*time.Second) {
		return fmt.Errorf("%s", t.T("daemon_restart_wait_failed"))
	}
	fmt.Println(t.T("daemon_restarted"))
	return nil
}

func waitReachable(base string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		if reachable(base) {
			return true
		}
	}
	return false
}

func daemonStart(base, dataDir string) error {
	if reachable(base) {
		return fmt.Errorf("%s", t.T("daemon_already_running", base))
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(filepath.Join(logDir, "daemon.out.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	cmd := exec.Command(exe, "_daemon", "--data-dir", dataDir)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.SysProcAttr = daemonProcAttr()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	_ = os.WriteFile(filepath.Join(dataDir, "daemon.pid"), []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o600)
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if reachable(base) {
			fmt.Println(t.T("daemon_started", base))
			return nil
		}
	}
	return fmt.Errorf("%s", t.T("daemon_start_wait_failed", filepath.Join(logDir, "daemon.out.log")))
}

// startDaemonFn is indirection so tests can observe/emulate lazy starts.
var startDaemonFn = daemonStart

// withDaemon ensures the daemon is running before executing an operation that
// needs it. Only reachable checks and daemon start are guarded by a file lock,
// so concurrent riffpad invocations start at most one daemon.
func withDaemon(fn func() error, base, dataDir string) error {
	if err := ensureDaemon(base, dataDir); err != nil {
		return err
	}
	return fn()
}

func ensureDaemon(base, dataDir string) error {
	if reachable(base) {
		return nil
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	lockPath := filepath.Join(dataDir, "daemon.lock")
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open daemon lock: %w", err)
	}
	defer lf.Close()
	if err := lockFile(lf); err != nil {
		return fmt.Errorf("lock daemon start: %w", err)
	}
	defer func() { _ = unlockFile(lf) }()
	// Another process may have started the daemon while we waited for the lock.
	if reachable(base) {
		return nil
	}
	if err := startDaemonFn(base, dataDir); err != nil {
		return err
	}
	return nil
}

// setupCmd installs (or removes) a Linux systemd user service so the daemon
// starts at login and restarts after crashes.
func setupCmd(args []string, dataDir string) error {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	remove := fs.Bool("remove", false, "stop and remove the autostart entry")
	_ = fs.Parse(args)
	if runtime.GOOS == "windows" {
		return setupWindowsTask(*remove, dataDir)
	}
	if runtime.GOOS != "linux" {
		return fmt.Errorf("setup currently supports Linux systemd / Windows Task Scheduler only")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	unitPath := filepath.Join(unitDir, "riffpad.service")
	if *remove {
		_, _ = systemctlFn("disable", "--now", "riffpad.service")
		if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Println(t.T("setup_removed"))
		return nil
	}
	exe, err := osExecutableFn()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(unitDir, 0o700); err != nil {
		return err
	}
	execStart := exe
	if strings.Contains(exe, " ") {
		execStart = `"` + exe + `"`
	}
	// The systemd user manager runs with a minimal PATH, which would hide
	// coding CLIs installed in user dirs (~/.local/bin etc.). Capture the
	// PATH of the shell that ran `riffpad setup` so the daemon can spawn
	// codex/kimi/claude. "%" is a systemd specifier, so escape it.
	path := strings.ReplaceAll(os.Getenv("PATH"), "%", "%%")
	unit := fmt.Sprintf(`[Unit]
Description=Riffpad daemon (AI agent remote control)
After=network-online.target

[Service]
Environment=PATH=%s
ExecStart=%s _daemon --data-dir %s
Restart=on-failure
RestartSec=2

[Install]
WantedBy=default.target
`, path, execStart, dataDir)
	if err := os.WriteFile(unitPath, []byte(unit), 0o600); err != nil {
		return err
	}
	if out, err := systemctlFn("daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %v\n%s", err, out)
	}
	if out, err := systemctlFn("enable", "--now", "riffpad.service"); err != nil {
		return fmt.Errorf("systemctl enable riffpad: %v\n%s", err, out)
	}
	fmt.Println(t.T("setup_installed", unitPath))
	fmt.Println(t.T("setup_done"))
	return nil
}

// setupWindowsTask registers (or removes) a scheduled task that starts the
// daemon at logon.
func setupWindowsTask(remove bool, dataDir string) error {
	if remove {
		out, err := exec.Command("schtasks", "/Delete", "/TN", "RiffpadDaemon", "/F").CombinedOutput()
		if err != nil {
			return fmt.Errorf("schtasks delete: %v\n%s", err, out)
		}
		fmt.Println(t.T("setup_removed"))
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	tr := fmt.Sprintf(`"%s" _daemon --data-dir "%s"`, exe, dataDir)
	out, err := exec.Command("schtasks", "/Create", "/TN", "RiffpadDaemon", "/TR", tr, "/SC", "ONLOGON", "/RL", "LIMITED", "/F").CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks create: %v\n%s", err, out)
	}
	fmt.Println(t.T("setup_installed", "RiffpadDaemon"))
	fmt.Println(t.T("setup_done"))
	return nil
}

func daemonStop(base, dataDir string) error {
	if !reachable(base) {
		return fmt.Errorf("%s", t.T("daemon_not_running"))
	}
	// Bound the shutdown request itself: a wedged daemon may accept the
	// connection yet never answer, and http.DefaultClient has no timeout.
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := daemonDo(client, http.MethodPost, base+"/api/shutdown", nil)
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if !reachable(base) {
			fmt.Println(t.T("daemon_stopped"))
			return nil
		}
	}
	// HTTP shutdown failed or the daemon ignored it; fall back to the pid
	// file written by daemonStart and kill the process (#174).
	if err := forceKillDaemon(dataDir); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if !reachable(base) {
			fmt.Println(t.T("daemon_stopped"))
			return nil
		}
	}
	return fmt.Errorf("%s", t.T("daemon_did_not_stop"))
}

// forceKillDaemon reads the pid file written by daemonStart and terminates
// the process after verifying it really is a riffpad daemon, so a stale or
// recycled pid can never take down an unrelated process (#174).
func forceKillDaemon(dataDir string) error {
	raw, err := os.ReadFile(filepath.Join(dataDir, "daemon.pid"))
	if err != nil {
		return fmt.Errorf("%s", t.T("daemon_stop_no_pid"))
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return fmt.Errorf("%s", t.T("daemon_stop_no_pid"))
	}
	if !isRiffpadDaemonProcess(pid) {
		return fmt.Errorf("%s", t.T("daemon_stop_pid_mismatch", pid))
	}
	if err := terminateProcess(pid); err != nil {
		return fmt.Errorf("%s", t.T("daemon_stop_kill_failed", err))
	}
	return nil
}

func statusCmd(base string) error { return commands.StatusCmd(base) }

func pairCmd(base string, args []string) error {
	fs := flag.NewFlagSet("pair", flag.ExitOnError)
	local := fs.Bool("local", false, "mint a local-only code for the embedded UI at 8787, even when connected to a relay")
	_ = fs.Parse(args)
	pairURL := base + "/api/pairings"
	if *local {
		pairURL += "?local=1"
	}
	data, err := pairWithRetry(func() (pairingResult, error) { return requestPairing(pairURL) })
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
	}
	if data.Code == "" {
		if data.ErrorCode == "relay_auth_expired" {
			return fmt.Errorf("%s", t.T("pair_login_expired"))
		}
		if data.Error != "" {
			return fmt.Errorf("%s", data.Error)
		}
		return fmt.Errorf("%s", t.T("pair_failed_status", data.Status))
	}
	if data.Local {
		if !*local {
			return fmt.Errorf("%s", t.T("pair_requires_login"))
		}
		// Local mode: the URL points at 127.0.0.1 and is only meaningful in a
		// browser on this machine, so a QR code (which implies scanning with
		// another device) would be misleading — print the URL instead.
		fmt.Println(t.T("pair_local", data.Code, data.URL))
		return nil
	}
	fmt.Println(t.T("pair_code", data.Code))
	qrterminal.GenerateWithConfig(data.URL, qrterminal.Config{
		Level:     qrterminal.L,
		Writer:    os.Stdout,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
		QuietZone: 1,
	})
	return nil
}

type pairingResult struct {
	Status    int    `json:"-"`
	Code      string `json:"code"`
	URL       string `json:"url"`
	Local     bool   `json:"local"`
	Error     string `json:"error"`
	ErrorCode string `json:"errorCode"`
}

func requestPairing(pairURL string) (pairingResult, error) {
	var data struct {
		Code      string `json:"code"`
		URL       string `json:"url"`
		Local     bool   `json:"local"`
		Error     string `json:"error"`
		ErrorCode string `json:"errorCode"`
	}
	resp, err := daemonDo(nil, http.MethodPost, pairURL, nil)
	if err != nil {
		return pairingResult{}, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return pairingResult{}, err
	}
	return pairingResult{
		Status:    resp.StatusCode,
		Code:      data.Code,
		URL:       data.URL,
		Local:     data.Local,
		Error:     data.Error,
		ErrorCode: data.ErrorCode,
	}, nil
}

// pairWithRetry treats "host offline" as transient and keeps retrying until
// the daemon registers with the relay or the deadline passes. All other
// responses (success, auth expiry, host not found, etc.) return immediately.
func pairWithRetry(post func() (pairingResult, error)) (pairingResult, error) {
	deadline := time.Now().Add(pairRetryMaxWait)
	waited := false
	for {
		data, err := post()
		if err != nil {
			return data, err
		}
		if data.Code != "" || data.ErrorCode != "" || data.Error != "host offline" || !time.Now().Before(deadline) {
			return data, nil
		}
		if !waited {
			fmt.Fprintln(os.Stderr, t.T("pair_waiting_host"))
			waited = true
		}
		time.Sleep(pairRetryDelay)
	}
}

func sessionsCmd(base string) error { return commands.SessionsCmd(base) }

func runCmd(args []string, base string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	name := fs.String("name", "", "session name")
	prompt := fs.String("prompt", "", "initial prompt")
	cwd := fs.String("cwd", "", "working directory")
	cli := fs.String("cli", "", "agent CLI (claude|kimi|codex)")
	_ = fs.Parse(args)
	rest := fs.Args()
	if len(rest) > 1 {
		return fmt.Errorf("unexpected arguments: %v", rest)
	}
	if len(rest) == 1 {
		*cli = rest[0]
	}
	if *cli == "" {
		*cli = "claude"
	}
	if *cwd == "" {
		// Default to the directory where the user ran the command, not the
		// daemon's own cwd (the daemon may have been started elsewhere).
		if wd, err := os.Getwd(); err == nil {
			*cwd = wd
		}
	}
	// Resolve the CLI binary from the user's interactive PATH so sessions
	// work even when the daemon runs under systemd with a minimal PATH.
	// Empty is fine: the daemon falls back to its own PATH lookup.
	binary := ""
	if p, err := exec.LookPath(*cli); err == nil {
		binary = p
	}
	body, _ := json.Marshal(map[string]string{
		"name": *name, "prompt": *prompt, "cwd": *cwd, "cli": *cli, "binary": binary,
	})
	resp, err := daemonDo(nil, http.MethodPost, base+"/api/sessions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
	}
	defer resp.Body.Close()
	var data struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		CLI   string `json:"cli"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}
	if data.ID == "" {
		if data.Error != "" {
			return fmt.Errorf("%s", data.Error)
		}
		return fmt.Errorf("%s", t.T("run_failed_status", resp.StatusCode))
	}
	if data.CLI == "codex" {
		return console.AttachCodexTUI(base, data.ID)
	}
	if data.CLI == "demo" {
		fmt.Println(t.T("session_url", data.ID, data.URL))
		return nil
	}
	if data.CLI == "claude" || data.CLI == "kimi" {
		if runtime.GOOS == "windows" {
			fmt.Println(t.T("session_url", data.ID, data.URL))
			return nil
		}
		return console.AttachConsoleTUI(base, data.ID, data.CLI)
	}
	fmt.Println(t.T("session_url", data.ID, data.URL))
	return nil
}

// attachCodexTUI waits for the daemon's Codex app-server thread and then runs
// `codex resume --remote` in the foreground so the TUI stays in the user's
// terminal (no-silent hosting). Ctrl-C exits the TUI; the daemon session
// remains available from the phone.
func attachCodexTUI(base, sessionID string) error {
	fmt.Println(t.T("codex_tui_starting"))
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := daemonDo(client, http.MethodGet, base+"/api/sessions/"+sessionID+"/connect", nil)
	if err != nil {
		return fmt.Errorf("%s", t.T("codex_tui_wait_failed", err))
	}
	defer resp.Body.Close()
	var info struct {
		Socket   string `json:"socket"`
		ThreadID string `json:"threadId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK || info.Socket == "" || info.ThreadID == "" {
		return fmt.Errorf("%s", t.T("codex_tui_not_ready", resp.StatusCode))
	}
	codexBin, err := exec.LookPath("codex")
	if err != nil {
		return fmt.Errorf("%s", t.T("codex_not_found", err))
	}
	cmd := exec.Command(codexBin, "resume", "--remote", "unix://"+info.Socket, info.ThreadID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Ctrl+C (SIGINT) is delivered to the whole foreground process group,
	// i.e. both codex and this CLI. Without handling it, Go kills the CLI
	// immediately and the session cleanup below never runs — the daemon
	// session would stay alive and stay remote-controllable. So capture the
	// signal, let codex exit normally (or kill it after a timeout), then
	// close the session.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	exited := make(chan struct{})
	go func() {
		select {
		case <-sigCh:
			select {
			case <-exited:
			case <-time.After(5 * time.Second):
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			}
		case <-exited:
		}
	}()
	// Lease heartbeat: renew every 5s so the daemon can close the session if
	// this CLI dies without running cleanup (SIGKILL, panic, network loss).
	hbStop := make(chan struct{})
	defer close(hbStop)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if resp, err := daemonDo(nil, http.MethodPost, base+"/api/sessions/"+sessionID+"/heartbeat", nil); err == nil {
					_ = resp.Body.Close()
				}
			case <-hbStop:
				return
			}
		}
	}()
	runErr := cmd.Run()
	close(exited)
	if runErr != nil {
		fmt.Fprintln(os.Stderr, t.T("codex_tui_exit_error"), runErr)
	}
	// The user exited the TUI — per the no-silent-hosting convention, exiting
	// means exiting. Close the daemon session so it disappears from the client
	// and cannot be remote-controlled anymore. Users who want a persistent
	// session should run riffpad inside tmux themselves.
	if resp, err := daemonDo(nil, http.MethodPost, base+"/api/sessions/"+sessionID+"/stop", nil); err == nil {
		_ = resp.Body.Close()
	}
	fmt.Printf("%s\n", t.T("codex_tui_exited", sessionID))
	return nil
}

func logsCmd(dataDir string) error { return commands.LogsCmd(dataDir) }

// attachCmd injects Claude Code hooks pointing at the local daemon, so a
// normal interactive `claude` session is captured and approvals can be made
// from the web UI / mobile. Existing user hooks are preserved: only riffpad's
// own entries are replaced, making repeated attaches idempotent.
func attachCmd(base string) error {
	// Attach mode is deprecated: it permanently edits the user's global
	// Claude settings, which breaks normal claude startup when the daemon is
	// off or unpaired. The implementation is kept below for reference; only
	// the entry point is disabled (#214).
	return fmt.Errorf("%s", t.T("attach_deprecated"))
	if !reachable(base) {
		return fmt.Errorf("%s", t.T("daemon_start_hint", base))
	}
	port := defaultDaemonPort(base)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		return err
	}
	settings := map[string]any{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	}
	backup := settingsPath + ".riffpad.bak"
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		raw, _ := json.MarshalIndent(settings, "", "  ")
		if err := os.WriteFile(backup, raw, 0o600); err != nil {
			return err
		}
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	httpHook := func(path string, timeout int) map[string]any {
		hookURL := baseURL + path
		if tok := localToken(); tok != "" {
			// The hook process cannot set headers; the daemon also accepts the
			// local API token as a query parameter.
			hookURL += "?token=" + url.QueryEscape(tok)
		}
		return map[string]any{
			"matcher": "",
			"hooks": []any{
				map[string]any{"type": "http", "url": hookURL, "timeout": timeout},
			},
		}
	}
	// Merge, don't overwrite: drop any previously injected riffpad entries
	// (idempotent re-attach) while keeping the user's own hooks intact.
	if stripRiffpadHooks(settings) {
		fmt.Println(t.T("attach_keep_existing"))
	}
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = map[string]any{}
		settings["hooks"] = hooks
	}
	add := func(event string, entry map[string]any) {
		list, _ := hooks[event].([]any)
		hooks[event] = append(list, entry)
	}
	add("SessionStart", httpHook("/hooks/claude/session-start", 10))
	add("SessionEnd", httpHook("/hooks/claude/session-end", 10))
	add("UserPromptSubmit", httpHook("/hooks/claude/user-prompt-submit", 30))
	add("MessageDisplay", httpHook("/hooks/claude/message-display", 10))
	add("PreToolUse", httpHook("/hooks/claude/pre-tool-use", 10))
	add("PostToolUse", httpHook("/hooks/claude/post-tool-use", 10))
	add("PermissionRequest", httpHook("/hooks/claude/permission", 600))
	add("Notification", httpHook("/hooks/claude/notification", 10))
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, raw, 0o600); err != nil {
		return err
	}
	fmt.Println(t.T("attach_injected"))
	fmt.Println(t.T("attach_next"))
	fmt.Println(t.T("attach_verify"))
	return nil
}

// riffpadHookPathPrefix namespaces every hook URL the daemon injects, so
// attach/detach can tell riffpad's own entries apart from the user's — query
// parameters (e.g. the ?token= added for local auth) are ignored by matching
// on the parsed URL path only.
const riffpadHookPathPrefix = "/hooks/claude/"

func isRiffpadHook(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && strings.HasPrefix(u.Path, riffpadHookPathPrefix)
}

// stripRiffpadHooks removes riffpad-injected hook entries from
// settings["hooks"] in place, keeping everything the user added. It reports
// whether any user-defined (non-riffpad) hooks remain.
func stripRiffpadHooks(settings map[string]any) bool {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return false
	}
	userHooks := false
	for event, list := range hooks {
		entries, ok := list.([]any)
		if !ok {
			userHooks = true
			continue
		}
		kept := make([]any, 0, len(entries))
		for _, e := range entries {
			entry, ok := e.(map[string]any)
			if !ok {
				kept = append(kept, e)
				continue
			}
			nested, ok := entry["hooks"].([]any)
			if !ok {
				kept = append(kept, e)
				continue
			}
			keptNested := make([]any, 0, len(nested))
			for _, h := range nested {
				if hm, ok := h.(map[string]any); ok {
					if raw, _ := hm["url"].(string); raw != "" && isRiffpadHook(raw) {
						continue
					}
				}
				keptNested = append(keptNested, h)
			}
			if len(keptNested) == 0 {
				continue
			}
			entry["hooks"] = keptNested
			kept = append(kept, entry)
		}
		if len(kept) == 0 {
			delete(hooks, event)
			continue
		}
		hooks[event] = kept
		userHooks = true
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	}
	return userHooks
}

// detachCmd removes only the riffpad-injected hook entries, leaving any user
// configuration (including hooks added after attach) untouched. The
// settings.json.riffpad.bak snapshot from the first attach is no longer used
// for restoration; it is kept on disk for manual reference.
func detachCmd() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("no settings found (nothing to detach)")
	}
	settings := map[string]any{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parse %s: %w", settingsPath, err)
	}
	stripRiffpadHooks(settings)
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, raw, 0o600); err != nil {
		return err
	}
	fmt.Println(t.T("detach_restored"))
	return nil
}

func defaultDaemonPort(base string) int {
	if i := strings.LastIndex(base, ":"); i >= 0 {
		if p, err := strconv.Atoi(base[i+1:]); err == nil {
			return p
		}
	}
	return 8787
}

// loginCmd logs into the Riffpad relay and stores the token in the daemon
// config. `riffpad relay login` is kept as an alias.
func loginCmd(args []string, dataDir string) error {
	return doLogin(args, dataDir)
}

// Injectable indirections so tests can stub browser opening and daemon
// restarts without touching a real daemon.
var (
	openBrowserFn   = openBrowser
	restartDaemonFn = restartDaemon
)

// authCmd prints which relay account the daemon is logged in as, verifying
// the saved token against the relay when possible.
func authCmd(dataDir string) error {
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	if cfg.RelayToken == "" {
		fmt.Println(t.T("auth_not_logged_in"))
		return nil
	}
	user := cfg.RelayUser
	relayURL := cfg.RelayURL
	if relayURL == "" {
		relayURL = "wss://api.riffpad.ai"
	}
	httpURL := strings.TrimSuffix(relayURL, "/")
	httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
	httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")
	req, err := http.NewRequest(http.MethodGet, httpURL+"/api/auth/me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.RelayToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(t.T("auth_logged_in_cached", user, relayURL))
		return nil
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var out struct {
			User struct {
				Username string `json:"username"`
			} `json:"user"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err == nil && out.User.Username != "" {
			user = out.User.Username
		}
		fmt.Println(t.T("auth_logged_in", user, relayURL))
	case http.StatusUnauthorized:
		fmt.Println(t.T("auth_token_invalid", user, relayURL))
	default:
		fmt.Println(t.T("auth_relay_error", resp.Status))
	}
	return nil
}

// logoutCmd clears the stored relay token. `riffpad relay logout` is kept as
// an alias. It also revokes the token on the relay (best effort) and forgets
// the saved username.
func logoutCmd(dataDir string) error {
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	if cfg.RelayToken != "" && cfg.RelayURL != "" {
		httpURL := strings.TrimSuffix(cfg.RelayURL, "/")
		httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
		httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")
		req, err := http.NewRequest(http.MethodPost, httpURL+"/api/auth/logout", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+cfg.RelayToken)
			client := &http.Client{Timeout: 5 * time.Second}
			if resp, err := client.Do(req); err == nil {
				resp.Body.Close()
			}
		}
	}
	cfg.RelayToken = ""
	cfg.RelayUser = ""
	// Merge under the config lock so a concurrently running daemon's writes
	// (host creds, fresh relay token) survive (#172).
	if err := config.Update(dataDir, func(c *config.Config) {
		c.RelayToken = cfg.RelayToken
		c.RelayUser = cfg.RelayUser
	}); err != nil {
		return err
	}
	fmt.Println(t.T("logout_done"))
	return nil
}

func relayCmd(args []string, dataDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: riffpad relay login|logout")
	}
	switch args[0] {
	case "login":
		return doLogin(args[1:], dataDir)
	case "logout":
		return logoutCmd(dataDir)
	default:
		return fmt.Errorf("usage: riffpad relay login|logout")
	}
}

// doLogin implements `riffpad login` / `riffpad relay login`. With no
// --username it starts the GitHub device flow against the default relay
// (flag > env > config > wss://api.riffpad.ai).
func doLogin(args []string, dataDir string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	url := fs.String("url", "", "relay URL (wss:// or ws://)")
	username := fs.String("username", os.Getenv("RIFFPAD_RELAY_USER"), "relay username (password login)")
	github := fs.Bool("github", false, "log in with GitHub OAuth (opens browser)")
	_ = fs.Parse(args)

	relayURL := *url
	if relayURL == "" {
		relayURL = os.Getenv("RIFFPAD_RELAY_URL")
	}
	if relayURL == "" {
		if cfg, err := config.Load(dataDir); err == nil {
			relayURL = cfg.RelayURL
		}
	}
	if relayURL == "" {
		relayURL = "wss://api.riffpad.ai"
	}
	httpURL := strings.TrimSuffix(relayURL, "/")
	httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
	httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")

	if *github || *username == "" {
		return oauthDeviceLogin(httpURL, relayURL, dataDir)
	}
	password := os.Getenv("RIFFPAD_RELAY_PASSWORD")
	if password == "" {
		fmt.Print(t.T("login_password"))
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		fmt.Println()
		password = string(b)
	}
	body, _ := json.Marshal(map[string]string{"username": *username, "password": password})
	resp, err := http.Post(httpURL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("login_failed"), err)
	}
	defer resp.Body.Close()
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if out.Token == "" {
		return fmt.Errorf("%s", t.T("login_failed_status", resp.StatusCode))
	}
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	cfg.RelayURL = relayURL
	cfg.RelayToken = out.Token
	if err := syncHostCreds(httpURL, out.Token, *username, cfg); err != nil {
		return err
	}
	cfg.RelayUser = *username
	// Persist under the config lock, merging onto the current on-disk state:
	// the running daemon may have written host credentials since our Load,
	// and a blind Save would clobber them (last-write-wins, #172).
	if err := config.Update(dataDir, func(c *config.Config) {
		c.RelayURL = cfg.RelayURL
		c.RelayToken = cfg.RelayToken
		c.RelayUser = cfg.RelayUser
		c.HostID = cfg.HostID
		c.HostSecret = cfg.HostSecret
	}); err != nil {
		return err
	}
	fmt.Println(t.T("login_success", *username))
	restartDaemonFn(dataDir)
	return nil
}

// oauthDeviceLogin uses the relay's GitHub device flow so passwordless
// accounts can log in from the CLI. The CLI polls until the user authorizes
// in the browser (https://app.riffpad.ai/device?code=…).
func oauthDeviceLogin(httpURL, relayURL, dataDir string) error {
	// A wedged relay connection must not hang the login forever; every
	// request in the device flow gets a hard timeout.
	oauthClient := &http.Client{Timeout: 10 * time.Second}
	body, _ := json.Marshal(map[string]string{})
	resp, err := oauthClient.Post(httpURL+"/api/auth/oauth/device", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("login_oauth_failed"), err)
	}
	defer resp.Body.Close()
	var dev struct {
		UserCode        string `json:"userCode"`
		VerificationURL string `json:"verificationURL"`
		ExpiresIn       int    `json:"expiresIn"`
		Interval        int    `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dev); err != nil {
		return err
	}
	if dev.UserCode == "" {
		return fmt.Errorf("%s", t.T("login_oauth_failed_status", resp.StatusCode))
	}
	if dev.VerificationURL == "" {
		dev.VerificationURL = strings.TrimSuffix(httpURL, "/") + "/device?code=" + url.QueryEscape(dev.UserCode)
	}
	fmt.Println(t.T("login_oauth_open", dev.VerificationURL, dev.UserCode))
	openBrowserFn(dev.VerificationURL)
	if dev.Interval <= 0 {
		dev.Interval = 3
	}
	if dev.ExpiresIn <= 0 {
		dev.ExpiresIn = 600
	}
	deadline := time.Now().Add(time.Duration(dev.ExpiresIn) * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(dev.Interval) * time.Second)
		payload, _ := json.Marshal(map[string]string{"code": dev.UserCode})
		presp, err := oauthClient.Post(httpURL+"/api/auth/oauth/device/poll", "application/json", bytes.NewReader(payload))
		if err != nil {
			// Transient network failure: keep polling until the deadline.
			lastErr = err
			continue
		}
		if presp.StatusCode == http.StatusUnauthorized {
			var relayErr struct {
				Error string `json:"error"`
			}
			_ = json.NewDecoder(presp.Body).Decode(&relayErr)
			presp.Body.Close()
			if relayErr.Error != "" {
				return fmt.Errorf("%s", relayErr.Error)
			}
			return fmt.Errorf("%s", t.T("login_oauth_failed_status", presp.StatusCode))
		}
		if presp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d", presp.StatusCode)
			presp.Body.Close()
			continue
		}
		var out struct {
			Pending  bool   `json:"pending"`
			Token    string `json:"token"`
			Username string `json:"username"`
		}
		decodeErr := json.NewDecoder(presp.Body).Decode(&out)
		presp.Body.Close()
		if decodeErr != nil {
			lastErr = decodeErr
			continue
		}
		if out.Pending {
			continue
		}
		if out.Token == "" || out.Username == "" {
			return fmt.Errorf("%s", t.T("login_oauth_failed_status", presp.StatusCode))
		}
		cfg, err := config.Load(dataDir)
		if err != nil {
			return err
		}
		cfg.RelayURL = relayURL
		cfg.RelayToken = out.Token
		if err := syncHostCreds(httpURL, out.Token, out.Username, cfg); err != nil {
			return err
		}
		cfg.RelayUser = out.Username
		// Persist under the config lock, merging onto the current on-disk
		// state so concurrent daemon writes are not clobbered (#172).
		if err := config.Update(dataDir, func(c *config.Config) {
			c.RelayURL = cfg.RelayURL
			c.RelayToken = cfg.RelayToken
			c.RelayUser = cfg.RelayUser
			c.HostID = cfg.HostID
			c.HostSecret = cfg.HostSecret
		}); err != nil {
			return err
		}
		fmt.Println(t.T("login_success", out.Username))
		restartDaemonFn(dataDir)
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("%s（最后错误：%v）", t.T("login_oauth_timeout"), lastErr)
	}
	return fmt.Errorf("%s", t.T("login_oauth_timeout"))
}

// syncHostCreds clears stale host credentials when the stored host belongs to
// a different relay account than the one just logged into.
//
// An account switch is handled locally (no network needed) so a successful
// authorization is never thrown away because the relay is briefly unreachable.
// Same-account re-login keeps the credentials; the relay ownership check is
// best-effort and only clears credentials when it can confirm the host no
// longer belongs to the user.
func syncHostCreds(httpURL, token, username string, cfg *config.Config) error {
	if cfg.HostID == "" || cfg.HostSecret == "" {
		return nil
	}
	if cfg.RelayUser != "" && cfg.RelayUser != username {
		cfg.HostID = ""
		cfg.HostSecret = ""
		return nil
	}
	req, err := http.NewRequest(http.MethodGet, httpURL+"/api/hosts", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, t.T("login_host_check_warn", err))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var out struct {
		Hosts []struct {
			ID string `json:"id"`
		} `json:"hosts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}
	owned := false
	for _, h := range out.Hosts {
		if h.ID == cfg.HostID {
			owned = true
			break
		}
	}
	if !owned {
		cfg.HostID = ""
		cfg.HostSecret = ""
	}
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// restartDaemon applies a new relay login to the running daemon. systemd-managed
// daemons are restarted via systemctl; plain `riffpad daemon start` processes
// are stopped and started again. A daemon that is not running is left alone.
func restartDaemon(dataDir string) {
	base := os.Getenv("RIFFPAD_URL")
	if base == "" {
		base = "http://127.0.0.1:8787"
	}
	if runtime.GOOS == "linux" {
		if out, err := exec.Command("systemctl", "--user", "is-active", "riffpad.service").CombinedOutput(); err == nil && strings.TrimSpace(string(out)) == "active" {
			if err := exec.Command("systemctl", "--user", "restart", "riffpad.service").Run(); err != nil {
				fmt.Println(t.T("login_restart_failed", err))
			} else {
				fmt.Println(t.T("login_restarted"))
			}
			return
		}
	}
	if !reachable(base) {
		return
	}
	if err := daemonStop(base, dataDir); err != nil {
		fmt.Println(t.T("login_restart_failed", err))
		return
	}
	if err := daemonStart(base, dataDir); err != nil {
		fmt.Println(t.T("login_restart_failed", err))
		return
	}
	fmt.Println(t.T("login_restarted"))
}

func reachable(base string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := daemonDo(client, http.MethodGet, base+"/api/status", nil)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// updateCmd checks the latest GitHub release, downloads the binary for the
// current platform, verifies its SHA256, and atomically replaces this
// executable (keeping a .riffpad.bak backup).
// killCmd triggers the daemon kill switch: stops all agent sessions and
// revokes all paired devices (local + cloud).
func killCmd(base string) error {
	resp, err := daemonDo(nil, http.MethodPost, base+"/api/killswitch", nil)
	if err != nil {
		return fmt.Errorf("daemon not reachable at %s: %w", base, err)
	}
	defer resp.Body.Close()
	var out struct {
		Killed   bool `json:"killed"`
		Sessions int  `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	fmt.Println(t.T("kill_done", out.Sessions))
	return nil
}

func updateCmd(args []string, dataDir string) error {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	force := fs.Bool("force", false, "reinstall even if already up to date")
	noRestart := fs.Bool("no-restart", false, "do not restart the daemon after updating")
	_ = fs.Parse(args)

	fmt.Println(t.T("update_current", version.Version))
	latest, err := latestReleaseTag()
	if err != nil {
		return err
	}
	fmt.Println(t.T("update_latest", latest))
	if !*force && compareVersions(version.Version, latest) >= 0 {
		fmt.Println(t.T("update_up_to_date"))
		return nil
	}

	osName, arch, err := updatePlatform()
	if err != nil {
		return err
	}
	asset := "riffpad-" + osName + "-" + arch
	base := updateDownloadBase
	fmt.Println(t.T("update_downloading", asset))

	exe, err := osExecutableFn()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".riffpad-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := downloadFile(base+asset, tmp); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := verifyChecksum(base+"sha256sums.txt", asset, tmpPath); err != nil {
		fmt.Println(t.T("update_checksum_failed"))
		return err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}

	backup := exe + ".riffpad.bak"
	if err := copyFile(exe, backup); err != nil {
		fmt.Println(t.T("update_backup_failed", err))
		return err
	}
	if err := os.Rename(tmpPath, exe); err != nil {
		fmt.Println(t.T("update_replace_failed", err))
		return err
	}
	fmt.Println(t.T("update_done", latest, backup))
	if !*noRestart && reachable(defaultDaemonBase()) {
		return daemonRestart(defaultDaemonBase(), dataDir)
	}
	fmt.Println(t.T("update_restart_hint"))
	return nil
}

func defaultDaemonBase() string {
	if b := os.Getenv("RIFFPAD_URL"); b != "" {
		return b
	}
	return "http://127.0.0.1:8787"
}

func latestReleaseTag() (string, error) {
	// http.DefaultClient has no timeout; a wedged network must fail the
	// update instead of hanging it forever (#174).
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(updateAPIBase + "/" + updateRepo + "/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", t.T("latest_release_failed", resp.Status))
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("%s", t.T("release_no_tag"))
	}
	return rel.TagName, nil
}

func updatePlatform() (string, string, error) {
	osName := strings.ToLower(runtime.GOOS)
	if osName != "linux" && osName != "darwin" {
		return "", "", fmt.Errorf("%s", t.T("update_platform_unsupported", runtime.GOOS))
	}
	arch := runtime.GOARCH
	if arch == "x86_64" {
		arch = "amd64"
	}
	if arch == "aarch64" {
		arch = "arm64"
	}
	if arch != "amd64" && arch != "arm64" {
		return "", "", fmt.Errorf("%s", t.T("update_arch_unsupported", runtime.GOARCH))
	}
	return osName, arch, nil
}

func downloadFile(url string, w io.Writer) error {
	// Binary downloads get a generous but bounded timeout (#174).
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", t.T("download_failed", url, resp.Status))
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func verifyChecksum(sumsURL, asset, path string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(sumsURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", t.T("download_failed", sumsURL, resp.Status))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	want := ""
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == asset {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("%s", t.T("checksum_missing", asset))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("SHA256 不匹配（期望 %s，实际 %s）", want, got)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// compareVersions compares semver-ish versions, ignoring "v" prefixes and
// prerelease/build suffixes (e.g. 0.1.0-m0 == v0.1.0).
func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	if i := strings.IndexAny(a, "-+"); i >= 0 {
		a = a[:i]
	}
	if i := strings.IndexAny(b, "-+"); i >= 0 {
		b = b[:i]
	}
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		av, bv := 0, 0
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

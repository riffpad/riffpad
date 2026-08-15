package commands

// Daemon lifecycle helpers: start/stop/restart of the background daemon
// process, managed via systemd (Linux), Task Scheduler (Windows), or a plain
// detached process. Migrated from cmd/riffpad during the #282 split.

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

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

// DaemonRestart restarts the daemon. When the systemd user service is active,
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
	if !cliutil.Reachable(base) {
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
	if cliutil.Reachable(base) {
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
		if cliutil.Reachable(base) {
			return true
		}
	}
	return false
}

// DaemonStart spawns the hidden `_daemon` process detached and waits for it
// to become reachable.
func daemonStart(base, dataDir string) error {
	if cliutil.Reachable(base) {
		return fmt.Errorf("%s", t.T("daemon_already_running", base))
	}
	exe, err := osExecutableFn()
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
		if cliutil.Reachable(base) {
			fmt.Println(t.T("daemon_started", base))
			return nil
		}
	}
	return fmt.Errorf("%s", t.T("daemon_start_wait_failed", filepath.Join(logDir, "daemon.out.log")))
}

// startDaemonFn is indirection so tests can observe/emulate lazy starts.
var startDaemonFn = daemonStart

// WithDaemon ensures the daemon is running before executing an operation that
// needs it. Only reachable checks and daemon start are guarded by a file lock,
// so concurrent riffpad invocations start at most one daemon.
func WithDaemon(fn func() error, base, dataDir string) error {
	if err := EnsureDaemon(base, dataDir); err != nil {
		return err
	}
	return fn()
}

// EnsureDaemon returns immediately when the daemon is reachable; otherwise it
// lazily starts it under a file lock (one starter wins).
func EnsureDaemon(base, dataDir string) error {
	if cliutil.Reachable(base) {
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
	if cliutil.Reachable(base) {
		return nil
	}
	if err := startDaemonFn(base, dataDir); err != nil {
		return err
	}
	return nil
}

func daemonStop(base, dataDir string) error {
	if !cliutil.Reachable(base) {
		return fmt.Errorf("%s", t.T("daemon_not_running"))
	}
	// Bound the shutdown request itself: a wedged daemon may accept the
	// connection yet never answer, and http.DefaultClient has no timeout.
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := cliutil.DaemonDo(client, http.MethodPost, base+"/api/shutdown", nil)
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if !cliutil.Reachable(base) {
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
		if !cliutil.Reachable(base) {
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

// DaemonCmd implements `riffpad daemon start|stop|restart`.
func DaemonCmd(args []string, base, dataDir string) error {
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

package commands

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// systemctlFn is an indirection so setup tests can run without touching the
// real systemd user session.
var systemctlFn = func(args ...string) ([]byte, error) {
	return exec.Command("systemctl", append([]string{"--user"}, args...)...).CombinedOutput()
}

// osExecutableFn is an indirection so setup/update tests can run against a
// throwaway binary instead of the real riffpad executable.
var osExecutableFn = os.Executable

// SetupCmd installs (or removes) a Linux systemd user service so the daemon
// starts at login and restarts after crashes.
func SetupCmd(args []string, dataDir string) error {
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
	exe, err := osExecutableFn()
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

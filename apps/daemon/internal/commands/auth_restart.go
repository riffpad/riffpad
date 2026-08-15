package commands

import (
	"fmt"
	"os"
	"runtime"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

// RestartDaemonAfterLogin applies a new relay login to the running daemon.
// systemd-managed daemons are restarted via systemctl; plain `riffpad daemon
// start` processes are stopped and started again. A daemon that is not
// running is left alone.
func RestartDaemonAfterLogin(dataDir string) {
	base := os.Getenv("RIFFPAD_URL")
	if base == "" {
		base = "http://127.0.0.1:8787"
	}
	if runtime.GOOS == "linux" {
		if active, err := systemdActiveFn(); err == nil && active {
			if err := systemdRestartFn(); err != nil {
				fmt.Println(t.T("login_restart_failed", err))
			} else {
				fmt.Println(t.T("login_restarted"))
			}
			return
		}
	}
	if !cliutil.Reachable(base) {
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

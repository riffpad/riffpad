//go:build !windows

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// isRiffpadDaemonProcess reports whether pid refers to a live riffpad
// executable, so forceKillDaemon never signals an unrelated process that
// recycled the pid from a stale daemon.pid file (#174).
func isRiffpadDaemonProcess(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	if runtime.GOOS == "linux" {
		exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if err != nil {
			return false
		}
		return strings.Contains(filepath.Base(exe), "riffpad")
	}
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err != nil {
		return false
	}
	return strings.Contains(filepath.Base(strings.TrimSpace(string(out))), "riffpad")
}

// terminateProcess asks the daemon to exit with SIGTERM first and escalates
// to SIGKILL if it is still alive after a grace period.
func terminateProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := p.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := p.Signal(syscall.Signal(0)); err != nil {
			return nil
		}
	}
	return p.Kill()
}

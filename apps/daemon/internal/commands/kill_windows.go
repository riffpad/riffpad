//go:build windows

package commands

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// isRiffpadDaemonProcess reports whether pid refers to a live riffpad
// executable, so forceKillDaemon never terminates an unrelated process that
// recycled the pid from a stale daemon.pid file (#174).
func isRiffpadDaemonProcess(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var buf [windows.MAX_PATH]uint16
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return false
	}
	name := filepath.Base(strings.ToLower(windows.UTF16ToString(buf[:size])))
	return strings.Contains(name, "riffpad")
}

// terminateProcess ends the daemon process. Windows has no SIGTERM
// equivalent reachable via os.Process, so this is an immediate kill; the
// graceful HTTP shutdown was already attempted before this runs.
func terminateProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

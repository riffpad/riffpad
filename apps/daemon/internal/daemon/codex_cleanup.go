package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// cleanupCodexProcesses kills app-server processes left over from an unclean
// daemon shutdown and removes their stale socket/pid files.
func (s *Server) cleanupCodexProcesses() {
	dir := filepath.Join(s.dataDir, "codex")
	// Sockets belonging to persisted sessions may be reused by
	// restoreSessions after a daemon restart; do not kill those app-servers
	// (their attached TUI must survive the restart).
	keep := map[string]bool{}
	if persisted, err := loadPersistedSessions(s.dataDir); err == nil {
		for _, ps := range persisted {
			if sock := ps.Connect["socket"]; sock != "" {
				keep[sock] = true
			}
		}
	}
	// Kill leftover app-server processes by pid file (written by current
	// adapters) and, on Linux, by scanning /proc cmdlines for any app-server
	// listening under our codex dir (covers processes spawned before pid
	// files existed, e.g. unclean upgrades).
	if runtime.GOOS == "linux" {
		s.killCodexByProcScan(dir, keep)
	}
	pidFiles, err := filepath.Glob(filepath.Join(dir, "*.pid"))
	if err != nil {
		return
	}
	for _, pf := range pidFiles {
		sock := strings.TrimSuffix(pf, ".pid")
		if keep[sock] {
			continue
		}
		data, err := os.ReadFile(pf)
		if err != nil {
			continue
		}
		var pid int
		n, _ := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
		if n != 1 || pid <= 0 {
			_ = os.Remove(pf)
			continue
		}
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Kill()
		}
		_ = os.Remove(pf)
		_ = os.Remove(strings.TrimSuffix(pf, ".pid"))
		s.log.Printf("cleaned up stale codex app-server pid=%d", pid)
	}
	// Remove any remaining stale sockets (their processes, if any, were
	// killed above or by pid file).
	if socks, err := filepath.Glob(filepath.Join(dir, "*.sock")); err == nil {
		for _, sf := range socks {
			if keep[sf] {
				continue
			}
			_ = os.Remove(sf)
		}
	}
}

func (s *Server) killCodexByProcScan(dir string, keep map[string]bool) {
	procs, err := filepath.Glob("/proc/[0-9]*/cmdline")
	if err != nil {
		return
	}
	for _, cmdlinePath := range procs {
		data, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}
		cmdline := strings.ReplaceAll(string(data), "\x00", " ")
		if !strings.Contains(cmdline, "app-server") || !strings.Contains(cmdline, dir) {
			continue
		}
		skip := false
		for sock := range keep {
			if strings.Contains(cmdline, sock) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		pidStr := strings.Trim(filepath.Base(filepath.Dir(cmdlinePath)), "/")
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			continue
		}
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Kill()
		}
		s.log.Printf("cleaned up stale codex app-server pid=%d (proc scan)", pid)
	}
}

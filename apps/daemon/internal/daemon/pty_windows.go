//go:build windows

package daemon

import "net/http"

// handleSessionPTY is unavailable on Windows: creack/pty reports
// ErrUnsupported, so foreground TUI attach is compiled out. `riffpad run`
// falls back to headless hosting there.
func (s *Server) handleSessionPTY(w http.ResponseWriter, r *http.Request, id string) {
	writeError(w, http.StatusNotImplemented, "PTY console is not supported on Windows yet")
}

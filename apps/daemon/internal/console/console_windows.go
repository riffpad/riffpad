//go:build windows

package console

import "fmt"

// AttachConsoleTUI is unavailable on Windows (creack/pty is unsupported);
// runCmd falls back to headless hosting before ever calling this.
func AttachConsoleTUI(base, sessionID, cliName string) error {
	return fmt.Errorf(t.T("windows_tui_unsupported"))
}

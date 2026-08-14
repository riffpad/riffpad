//go:build windows

package main

import "fmt"

// attachConsoleTUI is unavailable on Windows (creack/pty is unsupported);
// runCmd falls back to headless hosting before ever calling this.
func attachConsoleTUI(base, sessionID, cliName string) error {
	return fmt.Errorf(t.T("windows_tui_unsupported"))
}

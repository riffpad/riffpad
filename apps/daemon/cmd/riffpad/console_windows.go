//go:build windows

package main

import "fmt"

// attachConsoleTUI is unavailable on Windows (creack/pty is unsupported);
// runCmd falls back to headless hosting before ever calling this.
func attachConsoleTUI(base, sessionID string) error {
	return fmt.Errorf("前台 TUI 暂不支持 Windows")
}

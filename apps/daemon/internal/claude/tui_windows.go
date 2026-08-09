//go:build windows

package claude

import (
	"context"
	"fmt"
)

// spawnInteractive is unsupported on Windows (creack/pty reports
// ErrUnsupported); the adapter falls back to headless stream-json hosting.
func (c *Claude) spawnInteractive(ctx context.Context) error {
	return fmt.Errorf("interactive PTY not supported on Windows")
}

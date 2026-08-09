//go:build windows

package kimi

import (
	"context"
	"fmt"
)

// spawnInteractive is unsupported on Windows; the adapter falls back to ACP.
func (k *Kimi) spawnInteractive(ctx context.Context) error {
	return fmt.Errorf("interactive PTY not supported on Windows")
}

//go:build !windows

package claude

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
)

// spawnInteractive launches Claude Code's real interactive TUI on a PTY the
// daemon owns. Structured events and approvals keep flowing through the
// per-session settings hooks (see writeSettings); the PTY bytes are bridged
// to a local CLI console via AttachPTY. When no console is attached the read
// loop still drains the PTY so the TUI never blocks.
func (c *Claude) spawnInteractive(ctx context.Context) error {
	if err := c.writeSettings(); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	args := []string{"--settings", c.settingsPath, "--permission-mode", "default"}
	cmd := exec.CommandContext(ctx, c.binary, args...)
	if c.cwd != "" {
		cmd.Dir = c.cwd
	}
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("start %s TUI: %w", c.binary, err)
	}
	c.cmd = cmd
	c.pty = ptmx
	go func() {
		_ = cmd.Wait()
		c.mu.Lock()
		c.exited = true
		c.mu.Unlock()
		_ = ptmx.Close()
		c.closePTYSubs()
		close(c.doneCh)
	}()
	go c.ptyReadLoop()
	return nil
}

// ptyReadLoop drains the PTY master continuously and fans output out to the
// attached console(s). With no console, bytes are discarded (never block).
func (c *Claude) ptyReadLoop() {
	buf := make([]byte, 32*1024)
	for {
		n, err := c.pty.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			c.mu.Lock()
			for sub := range c.ptySubs {
				select {
				case sub.ch <- chunk:
				default: // slow console: drop rather than block the TUI
				}
			}
			c.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (c *Claude) closePTYSubs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for sub := range c.ptySubs {
		close(sub.ch)
		delete(c.ptySubs, sub)
	}
}

// AttachPTY hands a console to the interactive TUI. It waits briefly for the
// TUI process to start (the daemon returns the create response before spawn
// finishes), then registers the console. One console at a time is enforced
// by the daemon (a new `riffpad run` replaces the old attach).
func (c *Claude) AttachPTY() (adapter.Terminal, error) {
	deadline := time.Now().Add(15 * time.Second)
	for {
		c.mu.Lock()
		ptyFile := c.pty
		c.mu.Unlock()
		if ptyFile != nil {
			t := &ptyTerm{c: c, ch: make(chan []byte, 256)}
			c.mu.Lock()
			c.ptySubs[t] = struct{}{}
			c.mu.Unlock()
			return t, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("PTY not ready after 15s")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// ptyTerm implements adapter.Terminal for one console.
func (t *ptyTerm) Read(p []byte) (int, error) {
	chunk, ok := <-t.ch
	if !ok {
		return 0, io.EOF
	}
	n := copy(p, chunk)
	return n, nil
}

func (t *ptyTerm) Write(p []byte) (int, error) {
	if t.c.pty == nil {
		return 0, fmt.Errorf("PTY closed")
	}
	return t.c.pty.Write(p)
}

func (t *ptyTerm) Resize(cols, rows uint16) error {
	if t.c.pty == nil {
		return fmt.Errorf("PTY closed")
	}
	return pty.Setsize(t.c.pty, &pty.Winsize{Cols: cols, Rows: rows})
}

func (t *ptyTerm) Close() error {
	t.c.mu.Lock()
	defer t.c.mu.Unlock()
	if _, ok := t.c.ptySubs[t]; ok {
		delete(t.c.ptySubs, t)
		close(t.ch)
	}
	return nil
}

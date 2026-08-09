//go:build !windows

package kimi

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/creack/pty"
	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
)

// spawnInteractive launches Kimi's real interactive TUI on a PTY the daemon
// owns, using a per-session config.toml with [[hooks]] pointing at the local
// daemon. `--yolo` turns off Kimi's own in-terminal permission prompts; the
// PreToolUse hook becomes the (remote-controllable) approval gate instead.
func (k *Kimi) spawnInteractive(ctx context.Context) error {
	if err := k.writeSessionHome(); err != nil {
		return fmt.Errorf("write session home: %w", err)
	}
	kimiHome := filepath.Dir(k.configPath)
	args := []string{"--yolo"}
	cmd := exec.CommandContext(ctx, k.binary, args...)
	if k.cwd != "" {
		cmd.Dir = k.cwd
	}
	cmd.Env = append(os.Environ(), "KIMI_CODE_HOME="+kimiHome)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("start %s TUI: %w", k.binary, err)
	}
	k.mu.Lock()
	k.cmd = cmd
	k.pty = ptmx
	k.mu.Unlock()
	go func() {
		_ = cmd.Wait()
		k.mu.Lock()
		k.exited = true
		k.mu.Unlock()
		_ = ptmx.Close()
		k.closePTYSubs()
		close(k.doneCh)
	}()
	go k.ptyReadLoop()
	return nil
}

// ptyReadLoop drains the PTY master and fans output to attached consoles;
// with none attached, bytes are discarded so the TUI never blocks.
func (k *Kimi) ptyReadLoop() {
	buf := make([]byte, 32*1024)
	for {
		n, err := k.pty.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			k.mu.Lock()
			for sub := range k.ptySubs {
				select {
				case sub.ch <- chunk:
				default:
				}
			}
			k.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (k *Kimi) closePTYSubs() {
	k.mu.Lock()
	defer k.mu.Unlock()
	for sub := range k.ptySubs {
		close(sub.ch)
		delete(k.ptySubs, sub)
	}
}

// AttachPTY hands a console to the interactive TUI, waiting briefly for the
// TUI process to start.
func (k *Kimi) AttachPTY() (adapter.Terminal, error) {
	deadline := time.Now().Add(15 * time.Second)
	for {
		k.mu.Lock()
		ptyFile := k.pty
		k.mu.Unlock()
		if ptyFile != nil {
			t := &ptyTerm{k: k, ch: make(chan []byte, 256)}
			k.mu.Lock()
			k.ptySubs[t] = struct{}{}
			k.mu.Unlock()
			return t, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("PTY not ready after 15s")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (t *ptyTerm) Read(p []byte) (int, error) {
	chunk, ok := <-t.ch
	if !ok {
		return 0, io.EOF
	}
	return copy(p, chunk), nil
}

func (t *ptyTerm) Write(p []byte) (int, error) {
	if t.k.pty == nil {
		return 0, fmt.Errorf("PTY closed")
	}
	return t.k.pty.Write(p)
}

func (t *ptyTerm) Resize(cols, rows uint16) error {
	if t.k.pty == nil {
		return fmt.Errorf("PTY closed")
	}
	return pty.Setsize(t.k.pty, &pty.Winsize{Cols: cols, Rows: rows})
}

func (t *ptyTerm) Close() error {
	t.k.mu.Lock()
	defer t.k.mu.Unlock()
	if _, ok := t.k.ptySubs[t]; ok {
		delete(t.k.ptySubs, t)
		close(t.ch)
	}
	return nil
}

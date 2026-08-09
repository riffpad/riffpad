//go:build !windows

package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

// attachConsoleTUI bridges the user's terminal to the daemon-hosted
// interactive PTY (Claude foreground mode). Raw mode forwards keystrokes,
// window resizes are propagated, and Ctrl-C reaches the vendor TUI. When the
// vendor process exits (or the console disconnects) the daemon session is
// closed, matching the no-silent-hosting convention.
func attachConsoleTUI(base, sessionID, cliName string) error {
	fmt.Printf("正在启动 %s TUI（会话已托管到 daemon）…\n", cliName)
	wsURL := strings.Replace(base, "http://", "ws://", 1) +
		"/api/sessions/" + sessionID + "/pty"
	if tok := localToken(); tok != "" {
		wsURL += "?token=" + url.QueryEscape(tok)
	}
	var conn *websocket.Conn
	deadline := time.Now().Add(20 * time.Second)
	for {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			conn = c
			break
		}
		if time.Now().After(deadline) {
			// Session may be left running headless if the console never
			// attached; close it so we never silently host in the background.
			if resp, stopErr := daemonDo(nil, http.MethodPost, base+"/api/sessions/"+sessionID+"/stop", nil); stopErr == nil {
				_ = resp.Body.Close()
			}
			return fmt.Errorf("连接会话 PTY 失败: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	defer conn.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("raw terminal: %w", err)
	}
	defer term.Restore(fd, oldState)

	resize := func() {
		if w, h, err := term.GetSize(fd); err == nil && w > 0 && h > 0 {
			_ = conn.WriteJSON(map[string]any{
				"resize": map[string]uint16{"cols": uint16(w), "rows": uint16(h)},
			})
		}
	}
	resize()

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	go func() {
		for range winch {
			resize()
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			var msg struct {
				Out  string `json:"out"`
				Exit *int   `json:"exit"`
			}
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			if msg.Out != "" {
				if b, err := base64.StdEncoding.DecodeString(msg.Out); err == nil {
					_, _ = os.Stdout.Write(b)
				}
			}
			if msg.Exit != nil {
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				_ = conn.WriteJSON(map[string]any{
					"in": base64.StdEncoding.EncodeToString(buf[:n]),
				})
			}
			if err != nil {
				_ = conn.Close()
				return
			}
		}
	}()

	// Lease heartbeat so a dead CLI (SIGKILL/panic) lets the daemon close the
	// session instead of leaving a ghost.
	hbStop := make(chan struct{})
	defer close(hbStop)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if resp, err := daemonDo(nil, http.MethodPost, base+"/api/sessions/"+sessionID+"/heartbeat", nil); err == nil {
					_ = resp.Body.Close()
				}
			case <-hbStop:
				return
			}
		}
	}()

	<-done
	_ = term.Restore(fd, oldState)
	if resp, err := daemonDo(nil, http.MethodPost, base+"/api/sessions/"+sessionID+"/stop", nil); err == nil {
		_ = resp.Body.Close()
	}
	fmt.Printf("%s TUI 已退出，会话 %s 已关闭。\n", cliName, sessionID)
	return nil
}

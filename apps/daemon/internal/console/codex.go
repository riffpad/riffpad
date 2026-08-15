package console

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

// AttachCodexTUI waits for the daemon's Codex app-server thread and then runs
// `codex resume --remote` in the foreground so the TUI stays in the user's
// terminal (no-silent hosting). Ctrl-C exits the TUI; the daemon session
// remains available from the phone.
func AttachCodexTUI(base, sessionID string) error {
	fmt.Println(t.T("codex_tui_starting"))
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := cliutil.DaemonDo(client, http.MethodGet, base+"/api/sessions/"+sessionID+"/connect", nil)
	if err != nil {
		return fmt.Errorf("%s", t.T("codex_tui_wait_failed", err))
	}
	defer resp.Body.Close()
	var info struct {
		Socket   string `json:"socket"`
		ThreadID string `json:"threadId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK || info.Socket == "" || info.ThreadID == "" {
		return fmt.Errorf("%s", t.T("codex_tui_not_ready", resp.StatusCode))
	}
	codexBin, err := exec.LookPath("codex")
	if err != nil {
		return fmt.Errorf("%s", t.T("codex_not_found", err))
	}
	cmd := exec.Command(codexBin, "resume", "--remote", "unix://"+info.Socket, info.ThreadID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Ctrl+C (SIGINT) is delivered to the whole foreground process group,
	// i.e. both codex and this CLI. Without handling it, Go kills the CLI
	// immediately and the session cleanup below never runs — the daemon
	// session would stay alive and stay remote-controllable. So capture the
	// signal, let codex exit normally (or kill it after a timeout), then
	// close the session.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	exited := make(chan struct{})
	go func() {
		select {
		case <-sigCh:
			select {
			case <-exited:
			case <-time.After(5 * time.Second):
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			}
		case <-exited:
		}
	}()
	// Lease heartbeat: renew every 5s so the daemon can close the session if
	// this CLI dies without running cleanup (SIGKILL, panic, network loss).
	hbStop := make(chan struct{})
	defer close(hbStop)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if resp, err := cliutil.DaemonDo(nil, http.MethodPost, base+"/api/sessions/"+sessionID+"/heartbeat", nil); err == nil {
					_ = resp.Body.Close()
				}
			case <-hbStop:
				return
			}
		}
	}()
	runErr := cmd.Run()
	close(exited)
	if runErr != nil {
		fmt.Fprintln(os.Stderr, t.T("codex_tui_exit_error"), runErr)
	}
	// The user exited the TUI — per the no-silent-hosting convention, exiting
	// means exiting. Close the daemon session so it disappears from the client
	// and cannot be remote-controlled anymore. Users who want a persistent
	// session should run riffpad inside tmux themselves.
	if resp, err := cliutil.DaemonDo(nil, http.MethodPost, base+"/api/sessions/"+sessionID+"/stop", nil); err == nil {
		_ = resp.Body.Close()
	}
	fmt.Printf("%s\n", t.T("codex_tui_exited", sessionID))
	return nil
}

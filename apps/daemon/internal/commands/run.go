package commands

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
	"github.com/riffpad/riffpad/apps/daemon/internal/console"
)

func RunCmd(args []string, base string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	name := fs.String("name", "", "session name")
	prompt := fs.String("prompt", "", "initial prompt")
	cwd := fs.String("cwd", "", "working directory")
	cli := fs.String("cli", "", "agent CLI (claude|kimi|codex)")
	_ = fs.Parse(args)
	rest := fs.Args()
	if len(rest) > 1 {
		return fmt.Errorf("unexpected arguments: %v", rest)
	}
	if len(rest) == 1 {
		*cli = rest[0]
	}
	if *cli == "" {
		*cli = "claude"
	}
	if *cwd == "" {
		// Default to the directory where the user ran the command, not the
		// daemon's own cwd (the daemon may have been started elsewhere).
		if wd, err := os.Getwd(); err == nil {
			*cwd = wd
		}
	}
	// Resolve the CLI binary from the user's interactive PATH so sessions
	// work even when the daemon runs under systemd with a minimal PATH.
	// Empty is fine: the daemon falls back to its own PATH lookup.
	binary := ""
	if p, err := exec.LookPath(*cli); err == nil {
		binary = p
	}
	body, _ := json.Marshal(map[string]string{
		"name": *name, "prompt": *prompt, "cwd": *cwd, "cli": *cli, "binary": binary,
	})
	resp, err := cliutil.DaemonDo(nil, http.MethodPost, base+"/api/sessions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
	}
	defer resp.Body.Close()
	var data struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		CLI   string `json:"cli"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}
	if data.ID == "" {
		if data.Error != "" {
			return fmt.Errorf("%s", data.Error)
		}
		return fmt.Errorf("%s", t.T("run_failed_status", resp.StatusCode))
	}
	if data.CLI == "codex" {
		return console.AttachCodexTUI(base, data.ID)
	}
	if data.CLI == "demo" {
		fmt.Println(t.T("session_url", data.ID, data.URL))
		return nil
	}
	if data.CLI == "claude" || data.CLI == "kimi" {
		if runtime.GOOS == "windows" {
			fmt.Println(t.T("session_url", data.ID, data.URL))
			return nil
		}
		return console.AttachConsoleTUI(base, data.ID, data.CLI)
	}
	fmt.Println(t.T("session_url", data.ID, data.URL))
	return nil
}

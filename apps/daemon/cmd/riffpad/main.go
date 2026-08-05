// Command riffpad is the user-facing CLI controlling the local riffpadd daemon.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/logging"
)

const version = "0.1.0-m0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	base := os.Getenv("RIFFPAD_URL")
	if base == "" {
		base = "http://127.0.0.1:8787"
	}
	dataDir := os.Getenv("RIFFPAD_DIR")
	if dataDir == "" {
		d, err := config.DefaultDataDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "resolve data dir:", err)
			os.Exit(1)
		}
		dataDir = d
	}

	var err error
	switch os.Args[1] {
	case "daemon":
		err = daemonCmd(os.Args[2:], base, dataDir)
	case "status":
		err = statusCmd(base)
	case "pair":
		err = pairCmd(base)
	case "sessions":
		err = sessionsCmd(base)
	case "run":
		err = runCmd(os.Args[2:], base)
	case "logs":
		err = logsCmd(dataDir)
	case "attach":
		err = attachCmd(base)
	case "detach":
		err = detachCmd()
	case "version":
		fmt.Println("riffpad", version)
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "riffpad:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `riffpad — AI agent remote control (M0)

Usage:
  riffpad daemon start          start the background daemon (riffpadd)
  riffpad daemon stop           stop the daemon
  riffpad status                show daemon status
  riffpad pair                  print a pairing code and QR
  riffpad sessions              list sessions
  riffpad run [--name N] [--prompt P] [--cwd D] [--cli claude]
  riffpad attach                inject Claude Code hooks so the daemon captures your own CLI session
  riffpad detach                remove injected hooks
  riffpad logs                  tail daemon logs
  riffpad version`)
}

func daemonCmd(args []string, base, dataDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: riffpad daemon start|stop")
	}
	switch args[0] {
	case "start":
		return daemonStart(base, dataDir)
	case "stop":
		return daemonStop(base)
	default:
		return fmt.Errorf("usage: riffpad daemon start|stop")
	}
}

func daemonStart(base, dataDir string) error {
	if reachable(base) {
		return fmt.Errorf("daemon already running at %s", base)
	}
	bin, err := findRiffpadd()
	if err != nil {
		return err
	}
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(filepath.Join(logDir, "daemon.out.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	cmd := exec.Command(bin, "--data-dir", dataDir)
	cmd.Stdout = out
	cmd.Stderr = out
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start riffpadd: %w", err)
	}
	_ = os.WriteFile(filepath.Join(dataDir, "daemon.pid"), []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o600)
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if reachable(base) {
			fmt.Println("daemon started at", base)
			return nil
		}
	}
	return fmt.Errorf("daemon did not become reachable; check %s", filepath.Join(logDir, "daemon.out.log"))
}

func daemonStop(base string) error {
	if !reachable(base) {
		return fmt.Errorf("daemon is not running")
	}
	resp, err := http.Post(base+"/api/shutdown", "application/json", nil)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if !reachable(base) {
			fmt.Println("daemon stopped")
			return nil
		}
	}
	return fmt.Errorf("daemon did not stop")
}

func statusCmd(base string) error {
	resp, err := http.Get(base + "/api/status")
	if err != nil {
		return fmt.Errorf("daemon not reachable at %s: %w", base, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out bytes.Buffer
	if json.Indent(&out, body, "", "  ") != nil {
		out.Write(body)
	}
	fmt.Println(out.String())
	return nil
}

func pairCmd(base string) error {
	resp, err := http.Post(base+"/api/pairings", "application/json", nil)
	if err != nil {
		return fmt.Errorf("daemon not reachable at %s: %w", base, err)
	}
	defer resp.Body.Close()
	var data struct {
		Code string `json:"code"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}
	fmt.Printf("配对码：%s\n在手机/浏览器输入此配对码（或扫描二维码）\n", data.Code)
	qrterminal.GenerateWithConfig(data.URL, qrterminal.Config{
		Level:     qrterminal.L,
		Writer:    os.Stdout,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
		QuietZone: 1,
	})
	return nil
}

func sessionsCmd(base string) error {
	resp, err := http.Get(base + "/api/sessions")
	if err != nil {
		return fmt.Errorf("daemon not reachable at %s: %w", base, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out bytes.Buffer
	if json.Indent(&out, body, "", "  ") != nil {
		out.Write(body)
	}
	fmt.Println(out.String())
	return nil
}

func runCmd(args []string, base string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	name := fs.String("name", "", "session name")
	prompt := fs.String("prompt", "", "initial prompt")
	cwd := fs.String("cwd", "", "working directory")
	cli := fs.String("cli", "claude", "agent CLI (claude)")
	_ = fs.Parse(args)
	body, _ := json.Marshal(map[string]string{
		"name": *name, "prompt": *prompt, "cwd": *cwd, "cli": *cli,
	})
	resp, err := http.Post(base+"/api/sessions", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("daemon not reachable at %s: %w", base, err)
	}
	defer resp.Body.Close()
	var data struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}
	fmt.Printf("session: %s\n打开 %s 查看\n", data.ID, data.URL)
	return nil
}

func logsCmd(dataDir string) error {
	out, err := logging.Tail(dataDir, 200)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

// attachCmd injects Claude Code hooks pointing at the local daemon, so a
// normal interactive `claude` session is captured and approvals can be made
// from the web UI / mobile.
func attachCmd(base string) error {
	if !reachable(base) {
		return fmt.Errorf("daemon not reachable at %s (run: riffpad daemon start)", base)
	}
	port := defaultDaemonPort(base)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		return err
	}
	settings := map[string]any{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	}
	backup := settingsPath + ".riffpad.bak"
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		raw, _ := json.MarshalIndent(settings, "", "  ")
		if err := os.WriteFile(backup, raw, 0o600); err != nil {
			return err
		}
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	httpHook := func(path string, timeout int) map[string]any {
		return map[string]any{
			"matcher": "",
			"hooks": []any{
				map[string]any{"type": "http", "url": baseURL + path, "timeout": timeout},
			},
		}
	}
	settings["hooks"] = map[string]any{
		"SessionStart":      []any{httpHook("/hooks/claude/session-start", 10)},
		"SessionEnd":        []any{httpHook("/hooks/claude/session-end", 10)},
		"PreToolUse":        []any{httpHook("/hooks/claude/pre-tool-use", 10)},
		"PostToolUse":       []any{httpHook("/hooks/claude/post-tool-use", 10)},
		"PermissionRequest": []any{httpHook("/hooks/claude/permission", 600)},
		"Notification":      []any{httpHook("/hooks/claude/notification", 10)},
	}
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, raw, 0o600); err != nil {
		return err
	}
	fmt.Println("已注入 Claude Code hooks（备份在 settings.json.riffpad.bak）")
	fmt.Println("现在正常打开你的 claude（建议放在 tmux 里），daemon 会自动捕捉会话与审批。")
	fmt.Println("验证完运行: riffpad detach")
	return nil
}

// detachCmd restores the pre-attach settings file.
func detachCmd() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	backup := settingsPath + ".riffpad.bak"
	data, err := os.ReadFile(backup)
	if err != nil {
		return fmt.Errorf("no backup found (nothing to detach)")
	}
	if err := os.WriteFile(settingsPath, data, 0o600); err != nil {
		return err
	}
	// Keep the backup file; user can remove it manually.
	fmt.Println("已还原 Claude Code settings，hooks 已移除。")
	return nil
}

func defaultDaemonPort(base string) int {
	if i := strings.LastIndex(base, ":"); i >= 0 {
		if p, err := strconv.Atoi(base[i+1:]); err == nil {
			return p
		}
	}
	return 8787
}

func findRiffpadd() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "riffpadd")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	if p, err := exec.LookPath("riffpadd"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("riffpadd binary not found next to riffpad or on PATH")
}

func reachable(base string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(base + "/api/status")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

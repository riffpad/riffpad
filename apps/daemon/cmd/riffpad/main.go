// Command riffpad is the single Riffpad binary: user-facing CLI plus the
// hidden `_daemon` subcommand that runs the background daemon.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	"golang.org/x/term"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
	"github.com/riffpad/riffpad/apps/daemon/internal/logging"
)

const version = "0.1.0-m0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if os.Args[1] == "_daemon" {
		os.Exit(runDaemon(os.Args[2:]))
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
		err = withDaemon(func() error { return pairCmd(base) }, base, dataDir)
	case "sessions":
		err = withDaemon(func() error { return sessionsCmd(base) }, base, dataDir)
	case "run":
		err = withDaemon(func() error { return runCmd(os.Args[2:], base) }, base, dataDir)
	case "logs":
		err = logsCmd(dataDir)
	case "attach":
		err = withDaemon(func() error { return attachCmd(base) }, base, dataDir)
	case "detach":
		err = detachCmd()
	case "login":
		err = loginCmd(os.Args[2:], dataDir)
	case "logout":
		err = logoutCmd(dataDir)
	case "relay":
		err = relayCmd(os.Args[2:], dataDir)
	case "setup":
		err = setupCmd(os.Args[2:], dataDir)
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

// runDaemon is the hidden `riffpad _daemon` entry point used by daemon start
// and systemd. Keeping the daemon inside the same binary makes installation a
// single-file affair.
func runDaemon(args []string) int {
	fs := flag.NewFlagSet("_daemon", flag.ExitOnError)
	dataDir := fs.String("data-dir", "", "daemon data directory (default ~/.config/riffpad)")
	_ = fs.Parse(args)
	dir := *dataDir
	if dir == "" {
		var err error
		dir, err = config.DefaultDataDir()
		if err != nil {
			log.Printf("resolve data dir: %v", err)
			return 1
		}
	}
	logger, closer, err := logging.New(dir)
	if err != nil {
		log.Printf("init logging: %v", err)
		return 1
	}
	defer closer.Close()

	cfg, err := config.Load(dir)
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		logger.Fatalf("load keys: %v", err)
	}
	srv := daemon.New(cfg, keys, dir, logger, daemon.DefaultFactory())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("server: %v", err)
		}
		stop()
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown: %v", err)
	}
	logger.Printf("daemon stopped")
	return 0
}

func usage() {
	fmt.Fprintln(os.Stderr, `riffpad — AI agent remote control (M0)

Usage:
  riffpad daemon start          start the background daemon (same binary)
  riffpad daemon stop           stop the daemon
  riffpad status                show daemon status
  riffpad pair                  print a pairing code and QR
  riffpad sessions              list sessions
  riffpad run [--name N] [--prompt P] [--cwd D] [--cli claude|kimi|codex]
  riffpad attach                inject Claude Code hooks so the daemon captures your own CLI session
  riffpad detach                remove injected hooks
  riffpad login [--url wss://… --username …]
                                log in to Riffpad cloud (relay)
  riffpad logout                clear the saved login token
  riffpad setup                 install daemon auto-start (Linux systemd user service)
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
	exe, err := os.Executable()
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
	cmd := exec.Command(exe, "_daemon", "--data-dir", dataDir)
	cmd.Stdout = out
	cmd.Stderr = out
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
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

// startDaemonFn is indirection so tests can observe/emulate lazy starts.
var startDaemonFn = daemonStart

// withDaemon ensures the daemon is running before executing an operation that
// needs it. Only reachable checks and daemon start are guarded by a file lock,
// so concurrent riffpad invocations start at most one daemon.
func withDaemon(fn func() error, base, dataDir string) error {
	if err := ensureDaemon(base, dataDir); err != nil {
		return err
	}
	return fn()
}

func ensureDaemon(base, dataDir string) error {
	if reachable(base) {
		return nil
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	lockPath := filepath.Join(dataDir, "daemon.lock")
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open daemon lock: %w", err)
	}
	defer lf.Close()
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock daemon start: %w", err)
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)
	// Another process may have started the daemon while we waited for the lock.
	if reachable(base) {
		return nil
	}
	if err := startDaemonFn(base, dataDir); err != nil {
		return err
	}
	return nil
}

// setupCmd installs (or removes) a Linux systemd user service so the daemon
// starts at login and restarts after crashes.
func setupCmd(args []string, dataDir string) error {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	remove := fs.Bool("remove", false, "stop and remove the systemd user service")
	_ = fs.Parse(args)
	if runtime.GOOS != "linux" {
		return fmt.Errorf("setup currently supports Linux systemd only")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	unitPath := filepath.Join(unitDir, "riffpad.service")
	if *remove {
		_ = exec.Command("systemctl", "--user", "disable", "--now", "riffpad.service").Run()
		if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Println("已移除 riffpad systemd user 服务。")
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(unitDir, 0o700); err != nil {
		return err
	}
	execStart := exe
	if strings.Contains(exe, " ") {
		execStart = `"` + exe + `"`
	}
	unit := fmt.Sprintf(`[Unit]
Description=Riffpad daemon (AI agent remote control)
After=network-online.target

[Service]
ExecStart=%s _daemon --data-dir %s
Restart=on-failure
RestartSec=2

[Install]
WantedBy=default.target
`, execStart, dataDir)
	if err := os.WriteFile(unitPath, []byte(unit), 0o600); err != nil {
		return err
	}
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %v\n%s", err, out)
	}
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", "riffpad.service").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable riffpad: %v\n%s", err, out)
	}
	fmt.Printf("已安装并启用 %s\n", unitPath)
	fmt.Println("daemon 将随登录自启，崩溃后自动重启；以后可直接运行 riffpad run/attach/pair。")
	return nil
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
	cli := fs.String("cli", "claude", "agent CLI (claude|kimi|codex)")
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
		"UserPromptSubmit":  []any{httpHook("/hooks/claude/user-prompt-submit", 30)},
		"MessageDisplay":    []any{httpHook("/hooks/claude/message-display", 10)},
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

// loginCmd logs into the Riffpad relay and stores the token in the daemon
// config. `riffpad relay login` is kept as an alias.
func loginCmd(args []string, dataDir string) error {
	return relayCmd(args, dataDir)
}

// logoutCmd clears the stored relay token. `riffpad relay logout` is kept as
// an alias.
func logoutCmd(dataDir string) error {
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	cfg.RelayToken = ""
	if err := config.Save(dataDir, cfg); err != nil {
		return err
	}
	fmt.Println("已退出登录。")
	return nil
}

func relayCmd(args []string, dataDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: riffpad relay login|logout")
	}
	switch args[0] {
	case "login":
		fs := flag.NewFlagSet("login", flag.ExitOnError)
		url := fs.String("url", os.Getenv("RIFFPAD_RELAY_URL"), "relay URL (wss:// or ws://)")
		username := fs.String("username", os.Getenv("RIFFPAD_RELAY_USER"), "relay username")
		_ = fs.Parse(args[1:])
		if *url == "" || *username == "" {
			return fmt.Errorf("--url and --username are required")
		}
		password := os.Getenv("RIFFPAD_RELAY_PASSWORD")
		if password == "" {
			fmt.Print("密码: ")
			b, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}
			fmt.Println()
			password = string(b)
		}
		httpURL := strings.TrimSuffix(*url, "/")
		httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
		httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")
		body, _ := json.Marshal(map[string]string{"username": *username, "password": password})
		resp, err := http.Post(httpURL+"/api/auth/login", "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("登录失败: %w", err)
		}
		defer resp.Body.Close()
		var out struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return err
		}
		if out.Token == "" {
			return fmt.Errorf("登录失败（状态 %d）", resp.StatusCode)
		}
		cfg, err := config.Load(dataDir)
		if err != nil {
			return err
		}
		cfg.RelayURL = *url
		cfg.RelayToken = out.Token
		if err := config.Save(dataDir, cfg); err != nil {
			return err
		}
		fmt.Printf("已登录 %s，token 已保存到配置。\n", *username)
		return nil
	case "logout":
		cfg, err := config.Load(dataDir)
		if err != nil {
			return err
		}
		cfg.RelayToken = ""
		if err := config.Save(dataDir, cfg); err != nil {
			return err
		}
		fmt.Println("已退出登录。")
		return nil
	default:
		return fmt.Errorf("usage: riffpad relay login|logout")
	}
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

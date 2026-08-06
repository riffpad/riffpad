// Command riffpad is the single Riffpad binary: user-facing CLI plus the
// hidden `_daemon` subcommand that runs the background daemon.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/riffpad/riffpad/apps/daemon/internal/i18n"
	"github.com/riffpad/riffpad/apps/daemon/internal/logging"
)

const version = "0.1.0-m0"

const updateRepo = "riffpad/riffpad"

// t is the active language bundle, initialized in main from --lang / env.
var t = i18n.New(i18n.DefaultLang)

func main() {
	langFlag, args := extractLangFlag(os.Args[1:])
	t = i18n.New(i18n.Detect(langFlag))
	os.Args = append([]string{os.Args[0]}, args...)
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
			fmt.Fprintln(os.Stderr, t.T("resolve_data_dir", err))
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
	case "update":
		err = updateCmd(os.Args[2:])
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
	fmt.Fprintln(os.Stderr, t.T("usage"))
}

// extractLangFlag pulls a global --lang/-lang value out of args so it works
// before any subcommand, regardless of position.
func extractLangFlag(args []string) (lang string, rest []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--lang" || a == "-lang":
			if i+1 < len(args) {
				lang = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--lang="):
			lang = strings.TrimPrefix(a, "--lang=")
		case strings.HasPrefix(a, "-lang="):
			lang = strings.TrimPrefix(a, "-lang=")
		default:
			rest = append(rest, a)
		}
	}
	return lang, rest
}

func daemonCmd(args []string, base, dataDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s", t.T("usage_daemon"))
	}
	switch args[0] {
	case "start":
		return daemonStart(base, dataDir)
	case "stop":
		return daemonStop(base)
	default:
		return fmt.Errorf("%s", t.T("usage_daemon"))
	}
}

func daemonStart(base, dataDir string) error {
	if reachable(base) {
		return fmt.Errorf("%s", t.T("daemon_already_running", base))
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
			fmt.Println(t.T("daemon_started", base))
			return nil
		}
	}
	return fmt.Errorf("%s", t.T("daemon_start_wait_failed", filepath.Join(logDir, "daemon.out.log")))
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
		fmt.Println(t.T("setup_removed"))
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
	fmt.Println(t.T("setup_installed", unitPath))
	fmt.Println(t.T("setup_done"))
	return nil
}

func daemonStop(base string) error {
	if !reachable(base) {
		return fmt.Errorf("%s", t.T("daemon_not_running"))
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
			fmt.Println(t.T("daemon_stopped"))
			return nil
		}
	}
	return fmt.Errorf("%s", t.T("daemon_did_not_stop"))
}

func statusCmd(base string) error {
	resp, err := http.Get(base + "/api/status")
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
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
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
	}
	defer resp.Body.Close()
	var data struct {
		Code string `json:"code"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}
	fmt.Println(t.T("pair_code", data.Code))
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
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
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
	if *cwd == "" {
		// Default to the directory where the user ran the command, not the
		// daemon's own cwd (the daemon may have been started elsewhere).
		if wd, err := os.Getwd(); err == nil {
			*cwd = wd
		}
	}
	body, _ := json.Marshal(map[string]string{
		"name": *name, "prompt": *prompt, "cwd": *cwd, "cli": *cli,
	})
	resp, err := http.Post(base+"/api/sessions", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
	}
	defer resp.Body.Close()
	var data struct {
		ID  string `json:"id"`
		URL string `json:"url"`
		CLI string `json:"cli"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}
	if data.CLI == "codex" {
		return attachCodexTUI(base, data.ID)
	}
	fmt.Println(t.T("session_url", data.ID, data.URL))
	return nil
}

// attachCodexTUI waits for the daemon's Codex app-server thread and then runs
// `codex resume --remote` in the foreground so the TUI stays in the user's
// terminal (no-silent hosting). Ctrl-C exits the TUI; the daemon session
// remains available from the phone.
func attachCodexTUI(base, sessionID string) error {
	fmt.Println("正在启动 Codex TUI（会话已托管到 daemon）…")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(base + "/api/sessions/" + sessionID + "/connect")
	if err != nil {
		return fmt.Errorf("等待 Codex 会话就绪失败: %w", err)
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
		return fmt.Errorf("Codex 会话未就绪（状态 %d）", resp.StatusCode)
	}
	codexBin, err := exec.LookPath("codex")
	if err != nil {
		return fmt.Errorf("未找到 codex 可执行文件: %w", err)
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
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
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
	runErr := cmd.Run()
	close(exited)
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "Codex TUI 已退出：", runErr)
	}
	// The user exited the TUI — per the no-silent-hosting convention, exiting
	// means exiting. Close the daemon session so it disappears from the client
	// and cannot be remote-controlled anymore. Users who want a persistent
	// session should run riffpad inside tmux themselves.
	if resp, err := http.Post(base+"/api/sessions/"+sessionID+"/stop", "application/json", nil); err == nil {
		_ = resp.Body.Close()
	}
	fmt.Printf("Codex TUI 已退出，会话 %s 已关闭。\n", sessionID)
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
		return fmt.Errorf("%s", t.T("daemon_start_hint", base))
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
	fmt.Println(t.T("attach_injected"))
	fmt.Println(t.T("attach_next"))
	fmt.Println(t.T("attach_verify"))
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
	fmt.Println(t.T("detach_restored"))
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
	fmt.Println(t.T("logout_done"))
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
			fmt.Print(t.T("login_password"))
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
			return fmt.Errorf("%s: %w", t.T("login_failed"), err)
		}
		defer resp.Body.Close()
		var out struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return err
		}
		if out.Token == "" {
			return fmt.Errorf("%s", t.T("login_failed_status", resp.StatusCode))
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
		fmt.Println(t.T("login_success", *username))
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
		fmt.Println(t.T("logout_done"))
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

// updateCmd checks the latest GitHub release, downloads the binary for the
// current platform, verifies its SHA256, and atomically replaces this
// executable (keeping a .riffpad.bak backup).
func updateCmd(args []string) error {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	force := fs.Bool("force", false, "reinstall even if already up to date")
	_ = fs.Parse(args)

	fmt.Println(t.T("update_current", version))
	latest, err := latestReleaseTag()
	if err != nil {
		return err
	}
	fmt.Println(t.T("update_latest", latest))
	if !*force && compareVersions(version, latest) >= 0 {
		fmt.Println(t.T("update_up_to_date"))
		return nil
	}

	osName, arch, err := updatePlatform()
	if err != nil {
		return err
	}
	asset := "riffpad-" + osName + "-" + arch
	base := "https://github.com/" + updateRepo + "/releases/latest/download/"
	fmt.Println(t.T("update_downloading", asset))

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".riffpad-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := downloadFile(base+asset, tmp); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := verifyChecksum(base+"sha256sums.txt", asset, tmpPath); err != nil {
		fmt.Println(t.T("update_checksum_failed"))
		return err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}

	backup := exe + ".riffpad.bak"
	if err := copyFile(exe, backup); err != nil {
		fmt.Println(t.T("update_backup_failed", err))
		return err
	}
	if err := os.Rename(tmpPath, exe); err != nil {
		fmt.Println(t.T("update_replace_failed", err))
		return err
	}
	fmt.Println(t.T("update_done", latest, backup))
	fmt.Println(t.T("update_restart_hint"))
	return nil
}

func latestReleaseTag() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/" + updateRepo + "/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", t.T("latest_release_failed", resp.Status))
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("%s", t.T("release_no_tag"))
	}
	return rel.TagName, nil
}

func updatePlatform() (string, string, error) {
	osName := strings.ToLower(runtime.GOOS)
	if osName != "linux" && osName != "darwin" {
		return "", "", fmt.Errorf("%s", t.T("update_platform_unsupported", runtime.GOOS))
	}
	arch := runtime.GOARCH
	if arch == "x86_64" {
		arch = "amd64"
	}
	if arch == "aarch64" {
		arch = "arm64"
	}
	if arch != "amd64" && arch != "arm64" {
		return "", "", fmt.Errorf("%s", t.T("update_arch_unsupported", runtime.GOARCH))
	}
	return osName, arch, nil
}

func downloadFile(url string, w io.Writer) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", t.T("download_failed", url, resp.Status))
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func verifyChecksum(sumsURL, asset, path string) error {
	resp, err := http.Get(sumsURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", t.T("download_failed", sumsURL, resp.Status))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	want := ""
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == asset {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("%s", t.T("checksum_missing", asset))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("SHA256 不匹配（期望 %s，实际 %s）", want, got)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// compareVersions compares semver-ish versions, ignoring "v" prefixes and
// prerelease/build suffixes (e.g. 0.1.0-m0 == v0.1.0).
func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	if i := strings.IndexAny(a, "-+"); i >= 0 {
		a = a[:i]
	}
	if i := strings.IndexAny(b, "-+"); i >= 0 {
		b = b[:i]
	}
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		av, bv := 0, 0
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

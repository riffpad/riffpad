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
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
	"github.com/riffpad/riffpad/apps/daemon/internal/commands"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/console"
	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
	"github.com/riffpad/riffpad/apps/daemon/internal/i18n"
	"github.com/riffpad/riffpad/apps/daemon/internal/logging"
	"github.com/riffpad/riffpad/apps/daemon/internal/version"
)

const updateRepo = "riffpad/riffpad"

// updateAPIBase and updateDownloadBase point the updater at GitHub; they are
// vars so tests can serve a fake release server.
var (
	updateAPIBase      = "https://api.github.com/repos"
	updateDownloadBase = "https://github.com/" + updateRepo + "/releases/latest/download/"
)

// pairRetryDelay and pairRetryMaxWait control the transient "host offline"
// retry in mintPairingCode: right after the daemon starts, its relay
// WebSocket registration can lag the local HTTP endpoint by a few seconds.
var (
	pairRetryDelay   = time.Second
	pairRetryMaxWait = 10 * time.Second
)

// t is the active language bundle, initialized in main from --lang.
var t = i18n.New(i18n.DefaultLang)

// localToken and daemonDo delegate to internal/cliutil, which owns the
// daemon's local API token resolution shared by all CLI commands.
func localToken() string { return cliutil.LocalToken() }
func daemonDo(client *http.Client, method, url string, body io.Reader) (*http.Response, error) {
	return cliutil.DaemonDo(client, method, url, body)
}

func main() {
	langFlag, args := extractLangFlag(os.Args[1:])
	t = i18n.New(i18n.Detect(langFlag))
	commands.SetBundle(t)
	console.SetBundle(t)
	os.Args = append([]string{os.Args[0]}, args...)
	// Corrupted state files are backed up and rebuilt automatically (#172);
	// warn instead of dying at startup. runDaemon overrides this to also log.
	config.OnHeal = func(h config.Heal) {
		fmt.Fprintln(os.Stderr, t.T("config_file_healed", h.Path, h.Backup))
		if h.Kind == "keys" {
			fmt.Fprintln(os.Stderr, t.T("keys_regenerated_warn"))
		}
	}
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
	cliutil.SetDataDir(dataDir)

	var err error
	switch os.Args[1] {
	case "daemon":
		err = daemonCmd(os.Args[2:], base, dataDir)
	case "status":
		err = statusCmd(base)
	case "pair":
		err = withDaemon(func() error { return pairCmd(base, os.Args[2:]) }, base, dataDir)
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
	case "auth":
		err = authCmd(dataDir)
	case "login":
		err = loginCmd(os.Args[2:], dataDir)
	case "logout":
		err = logoutCmd(dataDir)
	case "relay":
		err = relayCmd(os.Args[2:], dataDir)
	case "setup":
		err = setupCmd(os.Args[2:], dataDir)
	case "kill":
		err = killCmd(base)
	case "update":
		err = updateCmd(os.Args[2:], dataDir)
	case "version":
		fmt.Println("riffpad", version.Version)
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
	enrichDaemonPath()
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
	// Route self-heal warnings (#172) through the daemon log as well.
	config.OnHeal = func(h config.Heal) {
		logger.Printf("%s", t.T("config_file_healed", h.Path, h.Backup))
		if h.Kind == "keys" {
			logger.Printf("%s", t.T("keys_regenerated_warn"))
		}
	}

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

// enrichDaemonPath appends common user bin directories to PATH so the daemon
// can spawn coding CLIs (codex/kimi/claude) even when it was started by
// systemd with a minimal PATH. Only existing directories are added; the
// authoritative PATH captured by `riffpad setup` and per-command binary paths
// passed by `riffpad run` take precedence because they are earlier in PATH.
func enrichDaemonPath() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	extra := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".codex", "bin"),
		filepath.Join(home, ".kimi-code", "bin"),
		filepath.Join(home, ".opencode", "bin"),
		filepath.Join(home, ".cargo", "bin"),
		filepath.Join(home, "go", "bin"),
		filepath.Join(home, ".bun", "bin"),
		filepath.Join(home, ".npm-global", "bin"),
	}
	seen := map[string]bool{}
	parts := make([]string, 0, 16)
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p != "" && !seen[p] {
			seen[p] = true
			parts = append(parts, p)
		}
	}
	for _, d := range extra {
		if seen[d] {
			continue
		}
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			parts = append(parts, d)
			seen[d] = true
		}
	}
	os.Setenv("PATH", strings.Join(parts, string(os.PathListSeparator)))
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
	return commands.DaemonCmd(args, base, dataDir)
}

func withDaemon(fn func() error, base, dataDir string) error {
	return commands.WithDaemon(fn, base, dataDir)
}

func statusCmd(base string) error { return commands.StatusCmd(base) }

func pairCmd(base string, args []string) error { return commands.PairCmd(base, args) }

func sessionsCmd(base string) error { return commands.SessionsCmd(base) }

func runCmd(args []string, base string) error { return commands.RunCmd(args, base) }

func logsCmd(dataDir string) error { return commands.LogsCmd(dataDir) }

func attachCmd(base string) error { return commands.AttachCmd(base) }

func detachCmd() error { return commands.DetachCmd() }

func setupCmd(args []string, dataDir string) error { return commands.SetupCmd(args, dataDir) }

// loginCmd logs into the Riffpad relay and stores the token in the daemon
// config. `riffpad relay login` is kept as an alias.
func loginCmd(args []string, dataDir string) error {
	return doLogin(args, dataDir)
}

// Injectable indirections so tests can stub browser opening and daemon
// restarts without touching a real daemon.
var (
	openBrowserFn   = openBrowser
	restartDaemonFn = restartDaemon
)

// authCmd prints which relay account the daemon is logged in as, verifying
// the saved token against the relay when possible.
func authCmd(dataDir string) error {
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	if cfg.RelayToken == "" {
		fmt.Println(t.T("auth_not_logged_in"))
		return nil
	}
	user := cfg.RelayUser
	relayURL := cfg.RelayURL
	if relayURL == "" {
		relayURL = "wss://api.riffpad.ai"
	}
	httpURL := strings.TrimSuffix(relayURL, "/")
	httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
	httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")
	req, err := http.NewRequest(http.MethodGet, httpURL+"/api/auth/me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.RelayToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(t.T("auth_logged_in_cached", user, relayURL))
		return nil
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var out struct {
			User struct {
				Username string `json:"username"`
			} `json:"user"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err == nil && out.User.Username != "" {
			user = out.User.Username
		}
		fmt.Println(t.T("auth_logged_in", user, relayURL))
	case http.StatusUnauthorized:
		fmt.Println(t.T("auth_token_invalid", user, relayURL))
	default:
		fmt.Println(t.T("auth_relay_error", resp.Status))
	}
	return nil
}

// logoutCmd clears the stored relay token. `riffpad relay logout` is kept as
// an alias. It also revokes the token on the relay (best effort) and forgets
// the saved username.
func logoutCmd(dataDir string) error {
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	if cfg.RelayToken != "" && cfg.RelayURL != "" {
		httpURL := strings.TrimSuffix(cfg.RelayURL, "/")
		httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
		httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")
		req, err := http.NewRequest(http.MethodPost, httpURL+"/api/auth/logout", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+cfg.RelayToken)
			client := &http.Client{Timeout: 5 * time.Second}
			if resp, err := client.Do(req); err == nil {
				resp.Body.Close()
			}
		}
	}
	cfg.RelayToken = ""
	cfg.RelayUser = ""
	// Merge under the config lock so a concurrently running daemon's writes
	// (host creds, fresh relay token) survive (#172).
	if err := config.Update(dataDir, func(c *config.Config) {
		c.RelayToken = cfg.RelayToken
		c.RelayUser = cfg.RelayUser
	}); err != nil {
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
		return doLogin(args[1:], dataDir)
	case "logout":
		return logoutCmd(dataDir)
	default:
		return fmt.Errorf("usage: riffpad relay login|logout")
	}
}

// doLogin implements `riffpad login` / `riffpad relay login`. With no
// --username it starts the GitHub device flow against the default relay
// (flag > env > config > wss://api.riffpad.ai).
func doLogin(args []string, dataDir string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	url := fs.String("url", "", "relay URL (wss:// or ws://)")
	username := fs.String("username", os.Getenv("RIFFPAD_RELAY_USER"), "relay username (password login)")
	github := fs.Bool("github", false, "log in with GitHub OAuth (opens browser)")
	_ = fs.Parse(args)

	relayURL := *url
	if relayURL == "" {
		relayURL = os.Getenv("RIFFPAD_RELAY_URL")
	}
	if relayURL == "" {
		if cfg, err := config.Load(dataDir); err == nil {
			relayURL = cfg.RelayURL
		}
	}
	if relayURL == "" {
		relayURL = "wss://api.riffpad.ai"
	}
	httpURL := strings.TrimSuffix(relayURL, "/")
	httpURL = strings.ReplaceAll(httpURL, "wss://", "https://")
	httpURL = strings.ReplaceAll(httpURL, "ws://", "http://")

	if *github || *username == "" {
		return oauthDeviceLogin(httpURL, relayURL, dataDir)
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
	cfg.RelayURL = relayURL
	cfg.RelayToken = out.Token
	if err := syncHostCreds(httpURL, out.Token, *username, cfg); err != nil {
		return err
	}
	cfg.RelayUser = *username
	// Persist under the config lock, merging onto the current on-disk state:
	// the running daemon may have written host credentials since our Load,
	// and a blind Save would clobber them (last-write-wins, #172).
	if err := config.Update(dataDir, func(c *config.Config) {
		c.RelayURL = cfg.RelayURL
		c.RelayToken = cfg.RelayToken
		c.RelayUser = cfg.RelayUser
		c.HostID = cfg.HostID
		c.HostSecret = cfg.HostSecret
	}); err != nil {
		return err
	}
	fmt.Println(t.T("login_success", *username))
	restartDaemonFn(dataDir)
	return nil
}

// oauthDeviceLogin uses the relay's GitHub device flow so passwordless
// accounts can log in from the CLI. The CLI polls until the user authorizes
// in the browser (https://app.riffpad.ai/device?code=…).
func oauthDeviceLogin(httpURL, relayURL, dataDir string) error {
	// A wedged relay connection must not hang the login forever; every
	// request in the device flow gets a hard timeout.
	oauthClient := &http.Client{Timeout: 10 * time.Second}
	body, _ := json.Marshal(map[string]string{})
	resp, err := oauthClient.Post(httpURL+"/api/auth/oauth/device", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("login_oauth_failed"), err)
	}
	defer resp.Body.Close()
	var dev struct {
		UserCode        string `json:"userCode"`
		VerificationURL string `json:"verificationURL"`
		ExpiresIn       int    `json:"expiresIn"`
		Interval        int    `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dev); err != nil {
		return err
	}
	if dev.UserCode == "" {
		return fmt.Errorf("%s", t.T("login_oauth_failed_status", resp.StatusCode))
	}
	if dev.VerificationURL == "" {
		dev.VerificationURL = strings.TrimSuffix(httpURL, "/") + "/device?code=" + url.QueryEscape(dev.UserCode)
	}
	fmt.Println(t.T("login_oauth_open", dev.VerificationURL, dev.UserCode))
	openBrowserFn(dev.VerificationURL)
	if dev.Interval <= 0 {
		dev.Interval = 3
	}
	if dev.ExpiresIn <= 0 {
		dev.ExpiresIn = 600
	}
	deadline := time.Now().Add(time.Duration(dev.ExpiresIn) * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(dev.Interval) * time.Second)
		payload, _ := json.Marshal(map[string]string{"code": dev.UserCode})
		presp, err := oauthClient.Post(httpURL+"/api/auth/oauth/device/poll", "application/json", bytes.NewReader(payload))
		if err != nil {
			// Transient network failure: keep polling until the deadline.
			lastErr = err
			continue
		}
		if presp.StatusCode == http.StatusUnauthorized {
			var relayErr struct {
				Error string `json:"error"`
			}
			_ = json.NewDecoder(presp.Body).Decode(&relayErr)
			presp.Body.Close()
			if relayErr.Error != "" {
				return fmt.Errorf("%s", relayErr.Error)
			}
			return fmt.Errorf("%s", t.T("login_oauth_failed_status", presp.StatusCode))
		}
		if presp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d", presp.StatusCode)
			presp.Body.Close()
			continue
		}
		var out struct {
			Pending  bool   `json:"pending"`
			Token    string `json:"token"`
			Username string `json:"username"`
		}
		decodeErr := json.NewDecoder(presp.Body).Decode(&out)
		presp.Body.Close()
		if decodeErr != nil {
			lastErr = decodeErr
			continue
		}
		if out.Pending {
			continue
		}
		if out.Token == "" || out.Username == "" {
			return fmt.Errorf("%s", t.T("login_oauth_failed_status", presp.StatusCode))
		}
		cfg, err := config.Load(dataDir)
		if err != nil {
			return err
		}
		cfg.RelayURL = relayURL
		cfg.RelayToken = out.Token
		if err := syncHostCreds(httpURL, out.Token, out.Username, cfg); err != nil {
			return err
		}
		cfg.RelayUser = out.Username
		// Persist under the config lock, merging onto the current on-disk
		// state so concurrent daemon writes are not clobbered (#172).
		if err := config.Update(dataDir, func(c *config.Config) {
			c.RelayURL = cfg.RelayURL
			c.RelayToken = cfg.RelayToken
			c.RelayUser = cfg.RelayUser
			c.HostID = cfg.HostID
			c.HostSecret = cfg.HostSecret
		}); err != nil {
			return err
		}
		fmt.Println(t.T("login_success", out.Username))
		restartDaemonFn(dataDir)
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("%s（最后错误：%v）", t.T("login_oauth_timeout"), lastErr)
	}
	return fmt.Errorf("%s", t.T("login_oauth_timeout"))
}

// syncHostCreds clears stale host credentials when the stored host belongs to
// a different relay account than the one just logged into.
//
// An account switch is handled locally (no network needed) so a successful
// authorization is never thrown away because the relay is briefly unreachable.
// Same-account re-login keeps the credentials; the relay ownership check is
// best-effort and only clears credentials when it can confirm the host no
// longer belongs to the user.
func syncHostCreds(httpURL, token, username string, cfg *config.Config) error {
	if cfg.HostID == "" || cfg.HostSecret == "" {
		return nil
	}
	if cfg.RelayUser != "" && cfg.RelayUser != username {
		cfg.HostID = ""
		cfg.HostSecret = ""
		return nil
	}
	req, err := http.NewRequest(http.MethodGet, httpURL+"/api/hosts", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, t.T("login_host_check_warn", err))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var out struct {
		Hosts []struct {
			ID string `json:"id"`
		} `json:"hosts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}
	owned := false
	for _, h := range out.Hosts {
		if h.ID == cfg.HostID {
			owned = true
			break
		}
	}
	if !owned {
		cfg.HostID = ""
		cfg.HostSecret = ""
	}
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// restartDaemon applies a new relay login to the running daemon. systemd-managed
// daemons are restarted via systemctl; plain `riffpad daemon start` processes
// are stopped and started again. A daemon that is not running is left alone.
// restartDaemon applies a new relay login to the running daemon; see
// internal/commands (daemon lifecycle). Kept here only until the auth
// migration moves its callers.
func restartDaemon(dataDir string) { commands.RestartDaemonAfterLogin(dataDir) }

func reachable(base string) bool { return cliutil.Reachable(base) }

func killCmd(base string) error { return commands.KillCmd(base) }

func updateCmd(args []string, dataDir string) error { return commands.UpdateCmd(args, dataDir) }

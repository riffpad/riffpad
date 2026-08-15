package commands

// Relay authentication commands: login (password + GitHub device flow),
// logout, auth status, and the `riffpad relay` alias group. Thick commands
// migrated last per issue #282's incremental plan.

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
)

// LoginCmd logs into the Riffpad relay and stores the token in the daemon
// config. `riffpad relay login` is kept as an alias.
func LoginCmd(args []string, dataDir string) error {
	return doLogin(args, dataDir)
}

// Injectable indirections so tests can stub browser opening and daemon
// restarts without touching a real daemon.
var (
	openBrowserFn   = openBrowser
	restartDaemonFn = RestartDaemonAfterLogin
)

// AuthCmd prints which relay account the daemon is logged in as, verifying
// the saved token against the relay when possible.
func AuthCmd(dataDir string) error {
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

// LogoutCmd clears the stored relay token. `riffpad relay logout` is kept as
// an alias. It also revokes the token on the relay (best effort) and forgets
// the saved username.
func LogoutCmd(dataDir string) error {
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

// RelayCmd implements `riffpad relay login|logout`.
func RelayCmd(args []string, dataDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: riffpad relay login|logout")
	}
	switch args[0] {
	case "login":
		return doLogin(args[1:], dataDir)
	case "logout":
		return LogoutCmd(dataDir)
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

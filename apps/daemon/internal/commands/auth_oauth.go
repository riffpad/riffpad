package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
)

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

// openBrowser hands a URL to the desktop. It is best-effort, and skipped
// entirely when RIFFPAD_NO_BROWSER is set: on a headless box (or a test
// harness driving its own browser) launching the user's default browser is
// both useless and rude.
func openBrowser(url string) {
	if os.Getenv("RIFFPAD_NO_BROWSER") != "" {
		return
	}
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

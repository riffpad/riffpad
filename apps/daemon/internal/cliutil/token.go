// Package cliutil holds helpers shared by the riffpad CLI commands: the
// local API token resolution and the authenticated daemon HTTP client.
package cliutil

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
)

// cliDataDir and cliToken feed LocalToken; cliDataDir is set via SetDataDir
// before dispatch, cliToken is loaded lazily on the first daemon API call.
var (
	cliDataDir string
	cliToken   string
	tokenOnce  sync.Once
)

// SetDataDir points LocalToken at the daemon's data directory (config.json
// lives there). Call once before dispatch; further calls are ignored so a
// stray re-entry cannot swap tokens mid-process.
func SetDataDir(dir string) {
	if dir != "" {
		cliDataDir = dir
	}
}

// SetToken overrides token resolution (used by tests to stub the token).
func SetToken(token string) {
	cliToken = token
	tokenOnce = sync.Once{}
}

// LocalToken returns the daemon's local API token. It lives in config.json
// (created on demand by config.Load), so the CLI and the daemon always agree
// on the same token without any user setup.
func LocalToken() string {
	tokenOnce.Do(func() {
		if cliToken == "" && cliDataDir != "" {
			if cfg, err := config.Load(cliDataDir); err == nil {
				cliToken = cfg.LocalToken
			}
		}
	})
	return cliToken
}

// DaemonDo performs an authenticated request against the local daemon API.
// A nil client uses http.DefaultClient.
func DaemonDo(client *http.Client, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok := LocalToken(); tok != "" {
		req.Header.Set(daemon.LocalTokenHeader, tok)
	}
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

// Reachable reports whether the daemon at base answers /api/status with the
// local token (a token-requiring daemon is reachable only when the CLI can
// authenticate).
func Reachable(base string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := DaemonDo(client, http.MethodGet, base+"/api/status", nil)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

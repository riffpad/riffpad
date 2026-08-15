package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mdp/qrterminal/v3"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

// pairRetryDelay and pairRetryMaxWait control the transient "host offline"
// retry in mintPairingCode: right after the daemon starts, its relay
// WebSocket registration can lag the local HTTP endpoint by a few seconds.
var (
	pairRetryDelay   = time.Second
	pairRetryMaxWait = 10 * time.Second
)

func PairCmd(base string, args []string) error {
	fs := flag.NewFlagSet("pair", flag.ExitOnError)
	local := fs.Bool("local", false, "mint a local-only code for the embedded UI at 8787, even when connected to a relay")
	_ = fs.Parse(args)
	pairURL := base + "/api/pairings"
	if *local {
		pairURL += "?local=1"
	}
	data, err := pairWithRetry(func() (pairingResult, error) { return requestPairing(pairURL) })
	if err != nil {
		return fmt.Errorf("%s: %w", t.T("daemon_not_reachable", base), err)
	}
	if data.Code == "" {
		if data.ErrorCode == "relay_auth_expired" {
			return fmt.Errorf("%s", t.T("pair_login_expired"))
		}
		if data.Error != "" {
			return fmt.Errorf("%s", data.Error)
		}
		return fmt.Errorf("%s", t.T("pair_failed_status", data.Status))
	}
	if data.Local {
		if !*local {
			return fmt.Errorf("%s", t.T("pair_requires_login"))
		}
		// Local mode: the URL points at 127.0.0.1 and is only meaningful in a
		// browser on this machine, so a QR code (which implies scanning with
		// another device) would be misleading — print the URL instead.
		fmt.Println(t.T("pair_local", data.Code, data.URL))
		return nil
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

type pairingResult struct {
	Status    int    `json:"-"`
	Code      string `json:"code"`
	URL       string `json:"url"`
	Local     bool   `json:"local"`
	Error     string `json:"error"`
	ErrorCode string `json:"errorCode"`
}

func requestPairing(pairURL string) (pairingResult, error) {
	var data struct {
		Code      string `json:"code"`
		URL       string `json:"url"`
		Local     bool   `json:"local"`
		Error     string `json:"error"`
		ErrorCode string `json:"errorCode"`
	}
	resp, err := cliutil.DaemonDo(nil, http.MethodPost, pairURL, nil)
	if err != nil {
		return pairingResult{}, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return pairingResult{}, err
	}
	return pairingResult{
		Status:    resp.StatusCode,
		Code:      data.Code,
		URL:       data.URL,
		Local:     data.Local,
		Error:     data.Error,
		ErrorCode: data.ErrorCode,
	}, nil
}

// pairWithRetry treats "host offline" as transient and keeps retrying until
// the daemon registers with the relay or the deadline passes. All other
// responses (success, auth expiry, host not found, etc.) return immediately.
func pairWithRetry(post func() (pairingResult, error)) (pairingResult, error) {
	deadline := time.Now().Add(pairRetryMaxWait)
	waited := false
	for {
		data, err := post()
		if err != nil {
			return data, err
		}
		if data.Code != "" || data.ErrorCode != "" || data.Error != "host offline" || !time.Now().Before(deadline) {
			return data, nil
		}
		if !waited {
			fmt.Fprintln(os.Stderr, t.T("pair_waiting_host"))
			waited = true
		}
		time.Sleep(pairRetryDelay)
	}
}

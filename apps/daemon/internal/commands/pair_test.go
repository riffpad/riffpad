package commands

// Tests for pairCmd's retry/decode behavior and runCmd's request/attach
// dispatch, moved from cmd/riffpad during the #282 split. The daemon API is
// stubbed with httptest servers.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPairWithRetryRecoversFromHostOffline(t *testing.T) {
	oldDelay := pairRetryDelay
	pairRetryDelay = time.Millisecond
	t.Cleanup(func() { pairRetryDelay = oldDelay })

	var calls int32
	res, err := pairWithRetry(func() (pairingResult, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return pairingResult{Error: "host offline"}, nil
		}
		return pairingResult{Code: "ABCDEF", URL: "https://app.riffpad.ai/pair?code=ABCDEF"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != "ABCDEF" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 pairing attempts, got %d", got)
	}
}

func TestPairWithRetryFailsFastOnNonTransientError(t *testing.T) {
	oldDelay := pairRetryDelay
	pairRetryDelay = time.Millisecond
	t.Cleanup(func() { pairRetryDelay = oldDelay })

	var calls int32
	_, err := pairWithRetry(func() (pairingResult, error) {
		atomic.AddInt32(&calls, 1)
		return pairingResult{Error: "host not found"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 attempt for host not found, got %d", got)
	}

	calls = 0
	_, _ = pairWithRetry(func() (pairingResult, error) {
		atomic.AddInt32(&calls, 1)
		return pairingResult{Error: "登录已过期", ErrorCode: "relay_auth_expired"}, nil
	})
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 attempt for auth expiry, got %d", got)
	}
}

func TestPairWithRetryStopsAtDeadline(t *testing.T) {
	oldDelay, oldWait := pairRetryDelay, pairRetryMaxWait
	pairRetryDelay = time.Millisecond
	pairRetryMaxWait = 15 * time.Millisecond
	t.Cleanup(func() { pairRetryDelay, pairRetryMaxWait = oldDelay, oldWait })

	var calls int32
	res, err := pairWithRetry(func() (pairingResult, error) {
		atomic.AddInt32(&calls, 1)
		return pairingResult{Error: "host offline"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Error != "host offline" {
		t.Fatalf("expected final host offline error, got %+v", res)
	}
	if got := atomic.LoadInt32(&calls); got < 5 {
		t.Fatalf("expected several retries before deadline, got %d", got)
	}
}

func TestRequestPairingDecodesErrorAndStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"host offline"}`))
	}))
	t.Cleanup(srv.Close)

	res, err := requestPairing(srv.URL + "/api/pairings")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusBadGateway || res.Error != "host offline" {
		t.Fatalf("unexpected pairing result: %+v", res)
	}
}

func TestPairCmdRejectsLocalWithoutFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "ABC123", "url": "http://127.0.0.1:8787/?pair=ABC123", "local": true,
		})
	}))
	t.Cleanup(srv.Close)

	err := PairCmd(srv.URL, nil)
	if err == nil || !strings.Contains(err.Error(), "pair --local") {
		t.Fatalf("expected login-required error mentioning --local, got %v", err)
	}
}

func TestPairCmdAllowsLocalWithFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "ABC123", "url": "http://127.0.0.1:8787/?pair=ABC123", "local": true,
		})
	}))
	t.Cleanup(srv.Close)

	if err := PairCmd(srv.URL, []string{"--local"}); err != nil {
		t.Fatalf("pair --local failed: %v", err)
	}
}

// TestPairCmdLocalSendsQuery: `riffpad pair --local` must ask the daemon for a
// local-only code by passing ?local=1, so a relay-connected daemon mints a code
// the embedded 8787 UI can claim instead of a cloud code.
func TestPairCmdLocalSendsQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "ABC123", "url": "http://127.0.0.1:8787/?pair=ABC123", "local": true,
		})
	}))
	t.Cleanup(srv.Close)

	if err := PairCmd(srv.URL, []string{"--local"}); err != nil {
		t.Fatalf("pair --local failed: %v", err)
	}
	if !strings.Contains(gotQuery, "local=1") {
		t.Fatalf("expected local=1 in pairing request query, got %q", gotQuery)
	}
}

// --- run command error propagation (issue #174 #1) ---

func TestRunCmdPropagatesDaemonError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": `unsupported cli "claud"`})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	err := RunCmd([]string{"--cli", "claud"}, srv.URL)
	if err == nil || !strings.Contains(err.Error(), `unsupported cli "claud"`) {
		t.Fatalf("expected unsupported cli error, got %v", err)
	}
}

func TestRunCmdFailsOnErrorStatusWithoutMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	err := RunCmd(nil, srv.URL)
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected HTTP status error, got %v", err)
	}
}

func TestRunCmdSuccess(t *testing.T) {
	var gotBinary string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Binary string `json:"binary"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotBinary = req.Binary
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "s1", "url": "http://x/s1", "cli": "sh"})
	}))
	t.Cleanup(srv.Close)

	if err := RunCmd([]string{"--cli", "sh"}, srv.URL); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	want, err := exec.LookPath("sh")
	if err != nil {
		t.Fatal(err)
	}
	if gotBinary != want {
		t.Fatalf("expected resolved binary %q in request, got %q", want, gotBinary)
	}
}

func TestRunCmdAcceptsPositionalCLI(t *testing.T) {
	var gotCLI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CLI string `json:"cli"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotCLI = req.CLI
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "s1", "url": "http://x/s1", "cli": req.CLI})
	}))
	t.Cleanup(srv.Close)

	if err := RunCmd([]string{"sh"}, srv.URL); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if gotCLI != "sh" {
		t.Fatalf("expected positional cli sh, got %q", gotCLI)
	}
}

func TestRunCmdRejectsMultiplePositionals(t *testing.T) {
	err := RunCmd([]string{"codex", "claude"}, "http://127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("expected unexpected-arguments error, got %v", err)
	}
}

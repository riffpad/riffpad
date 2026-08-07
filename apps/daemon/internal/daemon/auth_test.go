package daemon

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
)

// newAuthTestServer returns a daemon test server and its local API token.
func newAuthTestServer(t *testing.T) (string, *httptest.Server) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	if cfg.LocalToken == "" {
		t.Fatal("expected New to provide a local token")
	}
	return cfg.LocalToken, ts
}

func doRaw(t *testing.T, method, url string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

// A browser on a malicious page can only send simple requests — no custom
// headers — so every destructive endpoint must reject token-less calls.
func TestLocalAuthRejectsMissingToken(t *testing.T) {
	_, ts := newAuthTestServer(t)
	paths := []string{
		"/api/status",
		"/api/pairings",
		"/api/pair",
		"/api/devices",
		"/api/killswitch",
		"/api/sessions",
		"/api/shutdown",
		"/hooks/claude/session-start",
		"/ws",
	}
	for _, p := range paths {
		resp := doRaw(t, http.MethodPost, ts.URL+p, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s without token: expected 401, got %d", p, resp.StatusCode)
		}
	}
}

func TestLocalAuthRejectsWrongToken(t *testing.T) {
	_, ts := newAuthTestServer(t)
	resp := doRaw(t, http.MethodPost, ts.URL+"/api/killswitch", map[string]string{LocalTokenHeader: "wrong"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	resp = doRaw(t, http.MethodPost, ts.URL+"/api/killswitch?token=wrong", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong query token, got %d", resp.StatusCode)
	}
}

func TestLocalAuthAcceptsHeaderToken(t *testing.T) {
	token, ts := newAuthTestServer(t)
	h := map[string]string{LocalTokenHeader: token}
	resp := doRaw(t, http.MethodGet, ts.URL+"/api/status", h)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status with token: expected 200, got %d", resp.StatusCode)
	}
	resp = doRaw(t, http.MethodPost, ts.URL+"/api/killswitch", h)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("killswitch with token: expected 200, got %d", resp.StatusCode)
	}
	resp = doRaw(t, http.MethodPost, ts.URL+"/api/shutdown", h)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shutdown with token: expected 200, got %d", resp.StatusCode)
	}
}

// Claude hook processes cannot set headers; their URLs carry the token as a
// query parameter instead.
func TestLocalAuthAcceptsQueryToken(t *testing.T) {
	token, ts := newAuthTestServer(t)
	body := `{"hook_event_name":"SessionStart","session_id":"q1","cwd":"/tmp"}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/hooks/claude/session-start?token="+token, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("hook with query token: expected 200, got %d", resp.StatusCode)
	}
}

// DNS-rebinding defense: Host must be loopback even with a valid token.
func TestLocalAuthRejectsNonLoopbackHost(t *testing.T) {
	token, ts := newAuthTestServer(t)
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "evil.example.com"
	req.Header.Set(LocalTokenHeader, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-loopback Host: expected 403, got %d", resp.StatusCode)
	}
}

func TestLocalAuthOriginChecks(t *testing.T) {
	token, ts := newAuthTestServer(t)
	h := map[string]string{LocalTokenHeader: token, "Origin": "https://evil.example.com"}
	if resp := doRaw(t, http.MethodGet, ts.URL+"/api/status", h); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign Origin: expected 403, got %d", resp.StatusCode)
	}
	for _, origin := range []string{"http://127.0.0.1:8787", "http://localhost:3000", "http://[::1]:8080"} {
		h := map[string]string{LocalTokenHeader: token, "Origin": origin}
		if resp := doRaw(t, http.MethodGet, ts.URL+"/api/status", h); resp.StatusCode != http.StatusOK {
			t.Fatalf("loopback Origin %s: expected 200, got %d", origin, resp.StatusCode)
		}
	}
}

// The webui entry page and its assets stay public; only the API is guarded.
func TestLocalAuthStaticAssetsOpen(t *testing.T) {
	_, ts := newAuthTestServer(t)
	if resp := doRaw(t, http.MethodGet, ts.URL+"/", nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("index without token: expected 200, got %d", resp.StatusCode)
	}
}

func TestLocalAuthWebSocket(t *testing.T) {
	token, ts := newAuthTestServer(t)
	wsBase := "ws" + strings.TrimPrefix(ts.URL, "http")

	// No token at all: rejected before the upgrade.
	_, resp, err := websocket.DefaultDialer.Dial(wsBase+"/ws?device=d&session=s&eph=x", nil)
	if err == nil {
		t.Fatal("expected WS dial without token to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %+v", resp)
	}

	// Valid token but a foreign Origin: rejected (upgrader/middleware).
	header := http.Header{"Origin": []string{"https://evil.example.com"}}
	_, resp, err = websocket.DefaultDialer.Dial(wsBase+"/ws?device=d&session=s&eph=x&token="+token, header)
	if err == nil {
		t.Fatal("expected WS dial with foreign origin to fail")
	}
	if resp == nil || (resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnauthorized) {
		t.Fatalf("expected 403/401, got %+v", resp)
	}

	// Valid token, no Origin (non-browser client): passes auth and reaches the
	// handler, which then complains about the missing parameters (400, not 401).
	_, resp, err = websocket.DefaultDialer.Dial(wsBase+"/ws?token="+token, nil)
	if err == nil {
		t.Fatal("expected WS dial with missing params to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("token should have passed auth and reached the handler (400), got %+v", resp)
	}
}

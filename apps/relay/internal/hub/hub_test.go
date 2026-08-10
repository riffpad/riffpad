package hub

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
)

func newTestHub(t *testing.T) (*Hub, *httptest.Server) {
	t.Helper()
	h, err := New(log.New(io.Discard, "", 0), t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(h.Handler())
	t.Cleanup(ts.Close)
	return h, ts
}

func registerUser(t *testing.T, ts *httptest.Server, username string) string {
	t.Helper()
	resp, err := http.Post(ts.URL+"/api/auth/register", "application/json",
		strings.NewReader(`{"username":"`+username+`","password":"secret123"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Token == "" {
		t.Fatalf("register returned no token (status %d)", resp.StatusCode)
	}
	return out.Token
}

func registerHost(t *testing.T, ts *httptest.Server, token, name string) (string, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/hosts/register", strings.NewReader(`{"name":"`+name+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		HostID     string `json:"hostId"`
		HostSecret string `json:"hostSecret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.HostID == "" || out.HostSecret == "" {
		t.Fatalf("register host failed (status %d)", resp.StatusCode)
	}
	return out.HostID, out.HostSecret
}

func authPost(ts *httptest.Server, path, token, body string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func TestRelayRoutesViewerToHost(t *testing.T) {
	h, ts := newTestHub(t)
	token := registerUser(t, ts, "alice")
	hostID, hostSecret := registerHost(t, ts, token, "laptop")

	hostURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/host?hostId=" + hostID + "&token=" + hostSecret
	hostConn, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer hostConn.Close()
	fr, _ := json.Marshal(hostFrame{Kind: "sessions", Sessions: []SessionMeta{{ID: "s1", Name: "demo", CLI: "claude", Cwd: "/tmp", Status: "running"}}})
	if err := hostConn.WriteMessage(websocket.TextMessage, fr); err != nil {
		t.Fatal(err)
	}

	resp, err := authPost(ts, "/api/pairings", token, `{"hostId":"`+hostID+`","curve":"p256","publicKey":"AAA"}`)
	if err != nil {
		t.Fatal(err)
	}
	var pr struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = authPost(ts, "/api/pair", token, `{"code":"`+pr.Code+`","name":"phone","curve":"p256","publicKey":"BBB"}`)
	if err != nil {
		t.Fatal(err)
	}
	var pair struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pair); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	viewerURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?session=s1&device=" + pair.DeviceID + "&eph=EPH&token=" + token
	viewerConn, _, err := websocket.DefaultDialer.Dial(viewerURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer viewerConn.Close()

	hostConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := hostConn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var join hostFrame
	if err := json.Unmarshal(raw, &join); err != nil {
		t.Fatal(err)
	}
	if join.Kind != "join" || join.SessionID != "s1" || join.DeviceID != pair.DeviceID || join.Pub != "BBB" {
		t.Fatalf("unexpected join frame: %+v", join)
	}

	helloData := []byte(`{"kind":"hello"}`)
	hf, _ := json.Marshal(hostFrame{Kind: "viewer", ViewerID: join.ViewerID, Data: base64.RawStdEncoding.EncodeToString(helloData)})
	if err := hostConn.WriteMessage(websocket.TextMessage, hf); err != nil {
		t.Fatal(err)
	}
	viewerConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, got, err := viewerConn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(helloData) {
		t.Fatalf("viewer got %q, want %q", got, helloData)
	}

	envelope := []byte(`{"v":1,"ciphertext":"xyz"}`)
	if err := viewerConn.WriteMessage(websocket.TextMessage, envelope); err != nil {
		t.Fatal(err)
	}
	hostConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err = hostConn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var vf hostFrame
	if err := json.Unmarshal(raw, &vf); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawStdEncoding.DecodeString(vf.Data)
	if err != nil {
		t.Fatal(err)
	}
	if vf.Kind != "viewer" || string(decoded) != string(envelope) {
		t.Fatalf("unexpected viewer frame: %+v %q", vf, decoded)
	}
	_ = h
}

func TestAPIClientHostDoesNotServeWebUI(t *testing.T) {
	h, ts := newTestHub(t)
	h.apiHosts = []string{"api.riffpad.ai"}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
	req.Host = "api.riffpad.ai"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != http.StatusNotFound || out.Error != "not found" {
		t.Fatalf("api host status %d error %q, want JSON 404", resp.StatusCode, out.Error)
	}

	// The app host still gets the web UI.
	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
	req2.Host = "app.riffpad.ai"
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK || !strings.Contains(resp2.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("app host status %d content-type %q", resp2.StatusCode, resp2.Header.Get("Content-Type"))
	}
}

func TestPairingRequiresOnlineAndOwnedHost(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "bob")
	hostID, _ := registerHost(t, ts, token, "laptop")
	resp, err := authPost(ts, "/api/pairings", token, `{"hostId":"`+hostID+`","curve":"p256","publicKey":"AAA"}`)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for offline host, got %d", resp.StatusCode)
	}

	// A different user cannot pair against alice's host either.
	other := registerUser(t, ts, "mallory")
	resp, err = authPost(ts, "/api/pairings", other, `{"hostId":"`+hostID+`","curve":"p256","publicKey":"AAA"}`)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign host, got %d", resp.StatusCode)
	}
}

// TestPairingURLUsesAppURL covers the reverse-proxy deployment: TLS is
// terminated upstream (r.TLS == nil) and the request Host is the API domain,
// so the pairing URL must be built from the configured appURL instead.
func TestPairingURLUsesAppURL(t *testing.T) {
	h, ts := newTestHub(t)
	token := registerUser(t, ts, "alice")
	hostID, hostSecret := registerHost(t, ts, token, "laptop")

	hostURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/host?hostId=" + hostID + "&token=" + hostSecret
	hostConn, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer hostConn.Close()

	pairingURL := func(t *testing.T) string {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/pairings",
			strings.NewReader(`{"hostId":"`+hostID+`","curve":"p256","publicKey":"AAA"}`))
		if err != nil {
			t.Fatal(err)
		}
		// Simulate a TLS-terminating reverse proxy in front of the API vhost.
		req.Host = "api.riffpad.ai"
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("pairing status %d", resp.StatusCode)
		}
		var pr struct {
			Code string `json:"code"`
			URL  string `json:"url"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
			t.Fatal(err)
		}
		if pr.Code == "" || !strings.HasSuffix(pr.URL, "/?pair="+pr.Code) {
			t.Fatalf("bad pairing response code=%q url=%q", pr.Code, pr.URL)
		}
		return pr.URL
	}

	h.appURL = "https://app.riffpad.ai"
	if got, want := pairingURL(t), "https://app.riffpad.ai/?pair="; !strings.HasPrefix(got, want) {
		t.Fatalf("pairing url %q, want prefix %q", got, want)
	}

	// Trailing slashes in appURL must not produce a double-slash path.
	h.appURL = "https://app.riffpad.ai/"
	if got, want := pairingURL(t), "https://app.riffpad.ai/?pair="; !strings.HasPrefix(got, want) {
		t.Fatalf("pairing url %q, want prefix %q", got, want)
	}

	// Without appURL the handler falls back to the request scheme/host.
	h.appURL = ""
	if got, want := pairingURL(t), "http://api.riffpad.ai/?pair="; !strings.HasPrefix(got, want) {
		t.Fatalf("pairing url %q, want prefix %q", got, want)
	}
}

func TestPairForeignOwnerReturnsGenericError(t *testing.T) {
	_, ts := newTestHub(t)
	alice := registerUser(t, ts, "alice")
	hostID, hostSecret := registerHost(t, ts, alice, "laptop")

	hostURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/host?hostId=" + hostID + "&token=" + hostSecret
	hostConn, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer hostConn.Close()

	resp, err := authPost(ts, "/api/pairings", alice, `{"hostId":"`+hostID+`","curve":"p256","publicKey":"AAA"}`)
	if err != nil {
		t.Fatal(err)
	}
	var pr struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if pr.Code == "" {
		t.Fatal("no pairing code returned")
	}

	// A different user must not be able to tell that the code is live.
	mallory := registerUser(t, ts, "mallory")
	resp, err = authPost(ts, "/api/pair", mallory, `{"code":"`+pr.Code+`","name":"phone","curve":"p256","publicKey":"BBB"}`)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || out.Error != "invalid or expired pairing code" {
		t.Fatalf("foreign pair status %d error %q, want generic 401", resp.StatusCode, out.Error)
	}

	// The owner can still use the same code afterwards.
	resp, err = authPost(ts, "/api/pair", alice, `{"code":"`+pr.Code+`","name":"phone","curve":"p256","publicKey":"BBB"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner pair status %d", resp.StatusCode)
	}
}

// createPairingCode brings the host online over a throwaway WebSocket and
// creates a pairing code. Pairing itself does not require the host to stay
// online, so the connection is closed before returning.
func createPairingCode(t *testing.T, ts *httptest.Server, token, hostID, hostSecret string) string {
	t.Helper()
	hostURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/host?hostId=" + hostID + "&token=" + hostSecret
	hc, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := authPost(ts, "/api/pairings", token, `{"hostId":"`+hostID+`","curve":"p256","publicKey":"AAA"}`)
	_ = hc.Close()
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var pr struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil || pr.Code == "" {
		t.Fatalf("create pairing failed (status %d): %v", resp.StatusCode, err)
	}
	return pr.Code
}

// TestPairConcurrentSameCode fires simultaneous pair attempts at a single
// code: exactly one may create a device, the rest must get pairing_code_used.
func TestPairConcurrentSameCode(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "alice")
	hostID, hostSecret := registerHost(t, ts, token, "laptop")
	code := createPairingCode(t, ts, token, hostID, hostSecret)

	const n = 6
	statuses := make([]int, n)
	errCodes := make([]string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := authPost(ts, "/api/pair", token, `{"code":"`+code+`","name":"phone","curve":"p256","publicKey":"BBB"}`)
			if err != nil {
				t.Error(err)
				return
			}
			defer resp.Body.Close()
			var out struct {
				Code string `json:"code"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&out)
			statuses[i] = resp.StatusCode
			errCodes[i] = out.Code
		}(i)
	}
	wg.Wait()

	ok, used := 0, 0
	for i := range statuses {
		switch {
		case statuses[i] == http.StatusOK:
			ok++
		case statuses[i] == http.StatusConflict && errCodes[i] == "pairing_code_used":
			used++
		default:
			t.Errorf("attempt %d: status %d code %q", i, statuses[i], errCodes[i])
		}
	}
	if ok != 1 || used != n-1 {
		t.Fatalf("concurrent pair: ok=%d used=%d, want 1 and %d", ok, used, n-1)
	}

	// Exactly one device was created.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/devices", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var dl struct {
		Devices []json.RawMessage `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dl); err != nil {
		t.Fatal(err)
	}
	if len(dl.Devices) != 1 {
		t.Fatalf("devices after concurrent pair: %d, want 1", len(dl.Devices))
	}
}

// TestPairUsedCodeDistinctError: a consumed code gets a distinct 409 +
// pairing_code_used, while an unknown code keeps the generic 401.
func TestPairUsedCodeDistinctError(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "alice")
	hostID, hostSecret := registerHost(t, ts, token, "laptop")
	code := createPairingCode(t, ts, token, hostID, hostSecret)

	pair := func(code string) (int, string, string) {
		resp, err := authPost(ts, "/api/pair", token, `{"code":"`+code+`","name":"phone","curve":"p256","publicKey":"BBB"}`)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out.Error, out.Code
	}

	if st, _, _ := pair(code); st != http.StatusOK {
		t.Fatalf("first pair status %d", st)
	}
	st, _, errCode := pair(code)
	if st != http.StatusConflict || errCode != "pairing_code_used" {
		t.Fatalf("reused code: status %d code %q, want 409 pairing_code_used", st, errCode)
	}
	st, msg, errCode := pair("ZZZZZZ")
	if st != http.StatusUnauthorized || errCode != "" || msg != "invalid or expired pairing code" {
		t.Fatalf("wrong code: status %d error %q code %q, want generic 401", st, msg, errCode)
	}
}

// TestPairingSurvivesRestart: pairing codes live in the database, so a relay
// restart (deploy) does not invalidate in-flight codes.
func TestPairingSurvivesRestart(t *testing.T) {
	h, ts := newTestHub(t)
	token := registerUser(t, ts, "dave")
	hostID, hostSecret := registerHost(t, ts, token, "laptop")
	code := createPairingCode(t, ts, token, hostID, hostSecret)
	ts.Close()

	// "Restart": a new Hub over the same data directory.
	h2, err := New(log.New(io.Discard, "", 0), h.dataDir, "")
	if err != nil {
		t.Fatal(err)
	}
	ts2 := httptest.NewServer(h2.Handler())
	defer ts2.Close()

	resp, err := authPost(ts2, "/api/pair", token, `{"code":"`+code+`","name":"phone","curve":"p256","publicKey":"BBB"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pair after restart: status %d", resp.StatusCode)
	}
}

func TestPersistenceAcrossRestart(t *testing.T) {
	h, ts := newTestHub(t)
	token := registerUser(t, ts, "carol")
	hostID, hostSecret := registerHost(t, ts, token, "laptop")

	h2, err := New(log.New(io.Discard, "", 0), h.dataDir, "")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := h2.store.GetHost(hostID)
	if err != nil || rec.Secret != hostSecret {
		t.Fatalf("host not persisted across restart: %v", err)
	}
}

func TestGitHubOAuthCallback(t *testing.T) {
	h, ts := newTestHub(t)
	h.githubID = "test-client-id"
	h.githubSecret = "test-client-secret"
	var gotAccept, gotContentType string
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"gh_token_123","token_type":"bearer"}`)
	}))
	defer tokenSrv.Close()
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer gh_token_123" {
			t.Errorf("unexpected authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":424242,"login":"octocat","email":"octo@example.com"}`)
	}))
	defer userSrv.Close()
	h.githubTokenURL = tokenSrv.URL
	h.githubUserURL = userSrv.URL
	h.mu.Lock()
	h.oauthStates["teststate"] = oauthState{expires: time.Now().Add(time.Minute)}
	h.mu.Unlock()

	resp, err := http.Get(ts.URL + "/api/auth/github/callback?code=testcode&state=teststate")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("callback status %d: %s", resp.StatusCode, body)
	}
	if gotAccept != "application/json" {
		t.Errorf("token exchange Accept = %q, want application/json", gotAccept)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("token exchange Content-Type = %q, want application/x-www-form-urlencoded", gotContentType)
	}
	if !strings.Contains(string(body), "riffpad-oauth") || !strings.Contains(string(body), `token: "`) {
		t.Fatalf("callback did not hand token back: %s", body)
	}
	var u User
	if err := h.store.db.Where("username = ?", "octocat").First(&u).Error; err != nil {
		t.Fatalf("github user not created: %v", err)
	}
	if u.Email != "octo@example.com" {
		t.Fatalf("unexpected email: %s", u.Email)
	}
}

func TestGitHubOAuthLoopbackOpener(t *testing.T) {
	h, ts := newTestHub(t)
	h.githubID = "test-client-id"
	h.githubSecret = "test-client-secret"
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"gh_token_123","token_type":"bearer"}`)
	}))
	defer tokenSrv.Close()
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":424244,"login":"dev-user","email":""}`)
	}))
	defer userSrv.Close()
	h.githubTokenURL = tokenSrv.URL
	h.githubUserURL = userSrv.URL

	// A loopback opener is accepted; the callback posts the token back to it.
	h.mu.Lock()
	h.oauthStates["devstate"] = oauthState{
		expires: time.Now().Add(time.Minute),
		opener:  "http://localhost:5174",
	}
	h.mu.Unlock()
	resp, err := http.Get(ts.URL + "/api/auth/github/callback?code=testcode&state=devstate")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `postMessage(data, "http://localhost:5174")`) {
		t.Fatalf("loopback opener not honored: %d %s", resp.StatusCode, body)
	}

	// Non-allowlisted openers are rejected at login time.
	resp, err = http.Get(ts.URL + "/api/auth/github/login?opener=https%3A%2F%2Fevil.example")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("evil opener status %d, want 400", resp.StatusCode)
	}
}

func TestOAuthCallbackErrorPagesAreStyled(t *testing.T) {
	h, ts := newTestHub(t)
	h.githubID = "test-client-id"
	h.githubSecret = "test-client-secret"

	// Missing code/state renders a styled HTML error, not raw JSON.
	resp, err := http.Get(ts.URL + "/api/auth/github/callback")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest ||
		!strings.Contains(resp.Header.Get("Content-Type"), "text/html") ||
		!strings.Contains(string(body), "链接不完整") {
		t.Fatalf("missing-params status %d type %q body %s", resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}

	// Unknown state renders a styled expiry error.
	resp, err = http.Get(ts.URL + "/api/auth/github/callback?code=x&state=unknown&lang=en")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized ||
		!strings.Contains(string(body), "Link expired") ||
		!strings.Contains(string(body), "again") {
		t.Fatalf("invalid-state status %d body %s", resp.StatusCode, body)
	}
}

func TestDeviceLoginFlow(t *testing.T) {
	h, ts := newTestHub(t)
	h.githubID = "test-client-id"
	h.githubSecret = "test-client-secret"
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"gh_token_123","token_type":"bearer"}`)
	}))
	defer tokenSrv.Close()
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":424243,"login":"cli-user","email":""}`)
	}))
	defer userSrv.Close()
	h.githubTokenURL = tokenSrv.URL
	h.githubUserURL = userSrv.URL

	resp, err := http.Post(ts.URL+"/api/auth/oauth/device", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var dev struct {
		UserCode        string `json:"userCode"`
		VerificationURL string `json:"verificationURL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dev); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if dev.UserCode == "" || !strings.Contains(dev.VerificationURL, "code="+dev.UserCode) {
		t.Fatalf("unexpected device response: %+v", dev)
	}

	// Poll before authorization -> pending.
	payload, _ := json.Marshal(map[string]string{"code": dev.UserCode})
	resp, err = http.Post(ts.URL+"/api/auth/oauth/device/poll", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	var pending struct {
		Pending bool `json:"pending"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pending); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !pending.Pending {
		t.Fatal("expected pending before authorization")
	}

	// Authorize via the GitHub callback, as the device page would.
	h.mu.Lock()
	h.oauthStates["devstate"] = oauthState{expires: time.Now().Add(time.Minute), device: dev.UserCode, lang: "en"}
	h.mu.Unlock()
	resp, err = http.Get(ts.URL + "/api/auth/github/callback?code=testcode&state=devstate")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "CLI login authorized") {
		t.Fatalf("authorize status %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "Return to your terminal") {
		t.Fatalf("english receipt not rendered: %s", body)
	}

	// Poll after authorization -> token, one-time only.
	resp, err = http.Post(ts.URL+"/api/auth/oauth/device/poll", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Pending  bool   `json:"pending"`
		Token    string `json:"token"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if out.Pending || out.Token == "" || out.Username != "cli-user" {
		t.Fatalf("unexpected poll result: %+v", out)
	}
	resp, err = http.Post(ts.URL+"/api/auth/oauth/device/poll", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	var again struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&again)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || again.Error != "invalid or expired code" {
		t.Fatalf("second poll status %d error %q, want uniform 401", resp.StatusCode, again.Error)
	}

	// A device flow with an unknown code cannot start GitHub auth.
	resp, err = http.Get(ts.URL + "/api/auth/github/login?device=UNKNOWN")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown device code status %d, want 400", resp.StatusCode)
	}

	// Authorizing a code whose CLI has stopped polling (no recent poll) must
	// warn that no terminal is waiting instead of claiming success.
	code2 := "STALE12"
	h.mu.Lock()
	h.deviceLogins[code2] = &deviceLogin{UserCode: code2, ExpiresAt: time.Now().Add(10 * time.Minute)}
	h.oauthStates["stalestate"] = oauthState{expires: time.Now().Add(time.Minute), device: code2, lang: "en"}
	h.mu.Unlock()
	resp, err = http.Get(ts.URL + "/api/auth/github/callback?code=testcode&state=stalestate")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "no waiting terminal") {
		t.Fatalf("stale receipt missing: %s", body)
	}
}

func TestDeviceLoginStatusAndOAuthErrorPages(t *testing.T) {
	h, ts := newTestHub(t)
	h.githubID = "test-client-id"
	h.githubSecret = "test-client-secret"

	// Start a device login and verify the status endpoint reports it as valid.
	resp, err := http.Post(ts.URL+"/api/auth/oauth/device", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var dev struct {
		UserCode string `json:"userCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dev); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = http.Get(ts.URL + "/api/auth/oauth/device/status?code=" + dev.UserCode)
	if err != nil {
		t.Fatal(err)
	}
	var status struct {
		Valid     bool `json:"valid"`
		ExpiresIn int  `json:"expiresIn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !status.Valid || status.ExpiresIn <= 0 {
		t.Fatalf("unexpected status for fresh code: %+v", status)
	}

	// Unknown / used / expired codes must be reported as invalid, not leaky.
	resp, err = http.Get(ts.URL + "/api/auth/oauth/device/status?code=ZZZZ99")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if status.Valid {
		t.Fatalf("unknown code reported valid: %+v", status)
	}

	// Starting GitHub auth with a stale code must render a styled page, not
	// raw JSON.
	resp, err = http.Get(ts.URL + "/api/auth/github/login?device=ZZZZ99&lang=en")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Code expired") {
		t.Fatalf("stale code page status %d: %s", resp.StatusCode, body)
	}

	// A callback with an unknown state must still render in the language
	// embedded in the state parameter.
	resp, err = http.Get(ts.URL + "/api/auth/github/callback?code=x&state=deadbeef.en")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || !strings.Contains(string(body), "Link expired") {
		t.Fatalf("unknown state page status %d: %s", resp.StatusCode, body)
	}

	// A user declining GitHub authorization must see a styled cancellation
	// page instead of a JSON error.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"error":"access_denied","error_description":"user denied"}`)
	}))
	defer tokenSrv.Close()
	h.githubTokenURL = tokenSrv.URL
	h.mu.Lock()
	h.oauthStates["denystate.en"] = oauthState{expires: time.Now().Add(time.Minute), lang: "en"}
	h.mu.Unlock()
	resp, err = http.Get(ts.URL + "/api/auth/github/callback?code=x&state=denystate.en")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "Authorization canceled") {
		t.Fatalf("access denied page missing: %s", body)
	}
}

func TestDevicesListAndRevoke(t *testing.T) {
	h, ts := newTestHub(t)
	token := registerUser(t, ts, "dave")
	hostID, _ := registerHost(t, ts, token, "laptop")
	// Resolve the user id, then pair a device directly via the store.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	me, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var meData struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(me.Body).Decode(&meData); err != nil {
		t.Fatal(err)
	}
	me.Body.Close()
	if _, err := h.store.CreateDevice(meData.User.ID, hostID, "phone", "p256", "pub"); err != nil {
		t.Fatal(err)
	}
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/devices", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		Devices []Device `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(list.Devices) != 1 {
		t.Fatalf("unexpected devices: %+v", list.Devices)
	}
	// Revoke via API.
	did := list.Devices[0].ID
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/devices/"+did, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rev, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	rev.Body.Close()
	if rev.StatusCode != http.StatusOK {
		t.Fatalf("revoke status %d", rev.StatusCode)
	}
	if _, err := h.store.GetDevice(did); err == nil {
		t.Fatal("device should be deleted")
	}
}

func TestPairingRateLimit(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "dave")
	var got429 bool
	for i := 0; i < 11; i++ {
		resp, _ := authPost(ts, "/api/pair", token, `{"code":"WRONG","name":"x","curve":"p256","publicKey":"BBB"}`)
		if resp.StatusCode == http.StatusTooManyRequests {
			got429 = true
		}
		resp.Body.Close()
	}
	if !got429 {
		t.Fatal("expected a 429 after exceeding the per-IP rate limit")
	}
}

func TestAuthProtection(t *testing.T) {
	_, ts := newTestHub(t)
	resp, err := http.Get(ts.URL + "/api/sessions")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}
}

// shrinkHeartbeat shortens the heartbeat parameters for tests and restores
// them afterwards. The parameters are atomics, so this is race-safe even
// while connection goroutines are still running.
func shrinkHeartbeat(t *testing.T) {
	t.Helper()
	oldWrite, oldPing, oldPong := wsWriteWait.Load(), wsPingPeriod.Load(), wsPongWait.Load()
	wsWriteWait.Store(int64(100 * time.Millisecond))
	wsPingPeriod.Store(int64(50 * time.Millisecond))
	wsPongWait.Store(int64(300 * time.Millisecond))
	t.Cleanup(func() {
		wsWriteWait.Store(oldWrite)
		wsPingPeriod.Store(oldPing)
		wsPongWait.Store(oldPong)
	})
}

func dialHostWS(t *testing.T, ts *httptest.Server, hostID, secret string) *websocket.Conn {
	t.Helper()
	hostURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/host?hostId=" + hostID + "&token=" + secret
	conn, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// expectClosedByPeer simulates a half-open peer (pings are swallowed, never
// answered) and asserts the relay tears the connection down within maxWait.
func expectClosedByPeer(t *testing.T, conn *websocket.Conn, maxWait time.Duration) {
	t.Helper()
	conn.SetPingHandler(func(string) error { return nil })
	start := time.Now()
	conn.SetReadDeadline(start.Add(5 * time.Second))
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	if elapsed := time.Since(start); elapsed > maxWait {
		t.Fatalf("relay did not close the half-open connection (waited %v)", elapsed)
	}
}

func TestHostHeartbeatDropsSilentPeer(t *testing.T) {
	shrinkHeartbeat(t)
	h, ts := newTestHub(t)
	token := registerUser(t, ts, "hb-host")
	hostID, hostSecret := registerHost(t, ts, token, "laptop")
	conn := dialHostWS(t, ts, hostID, hostSecret)

	expectClosedByPeer(t, conn, 2*time.Second)

	h.mu.Lock()
	_, ok := h.hosts[hostID]
	h.mu.Unlock()
	if ok {
		t.Fatal("silent host still registered after heartbeat timeout")
	}
}

func TestHostHeartbeatKeepsResponsivePeer(t *testing.T) {
	shrinkHeartbeat(t)
	h, ts := newTestHub(t)
	token := registerUser(t, ts, "hb-alive")
	hostID, hostSecret := registerHost(t, ts, token, "laptop")
	conn := dialHostWS(t, ts, hostID, hostSecret)

	readErr := make(chan error, 1)
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				readErr <- err
				return
			}
		}
	}()
	// The default ping handler answers pongs while reading: the connection
	// must survive well beyond the pong deadline.
	time.Sleep(3 * wsPongTimeout())
	select {
	case err := <-readErr:
		t.Fatalf("responsive host connection dropped: %v", err)
	default:
	}
	h.mu.Lock()
	_, ok := h.hosts[hostID]
	h.mu.Unlock()
	if !ok {
		t.Fatal("responsive host removed from hub")
	}
}

func TestViewerHeartbeatDropsSilentPeer(t *testing.T) {
	shrinkHeartbeat(t)
	h, ts := newTestHub(t)
	token := registerUser(t, ts, "hb-viewer")
	hostID, hostSecret := registerHost(t, ts, token, "laptop")
	hostConn := dialHostWS(t, ts, hostID, hostSecret)
	fr, _ := json.Marshal(hostFrame{Kind: "sessions", Sessions: []SessionMeta{{ID: "s1", Name: "demo", CLI: "claude", Cwd: "/tmp", Status: "running"}}})
	if err := hostConn.WriteMessage(websocket.TextMessage, fr); err != nil {
		t.Fatal(err)
	}

	resp, err := authPost(ts, "/api/pairings", token, `{"hostId":"`+hostID+`","curve":"p256","publicKey":"AAA"}`)
	if err != nil {
		t.Fatal(err)
	}
	var pr struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = authPost(ts, "/api/pair", token, `{"code":"`+pr.Code+`","name":"phone","curve":"p256","publicKey":"BBB"}`)
	if err != nil {
		t.Fatal(err)
	}
	var pair struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pair); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	viewerURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?session=s1&device=" + pair.DeviceID + "&eph=EPH&token=" + token
	vc, _, err := websocket.DefaultDialer.Dial(viewerURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { vc.Close() })

	expectClosedByPeer(t, vc, 2*time.Second)

	h.mu.Lock()
	n := len(h.viewers)
	h.mu.Unlock()
	if n != 0 {
		t.Fatalf("silent viewer still registered: %d", n)
	}
}

// announceSessions sends a "sessions" frame from a host connection.
func announceSessions(t *testing.T, conn *websocket.Conn, sessions ...SessionMeta) {
	t.Helper()
	fr, _ := json.Marshal(hostFrame{Kind: "sessions", Sessions: sessions})
	if err := conn.WriteMessage(websocket.TextMessage, fr); err != nil {
		t.Fatal(err)
	}
}

func listSessionIDs(t *testing.T, ts *httptest.Server, token string) map[string]bool {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Sessions []SessionMeta `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, s := range out.Sessions {
		ids[s.ID] = true
	}
	return ids
}

// waitForSessions polls /api/sessions until exactly the wanted session ids
// are listed (announces are processed asynchronously on the host read loop).
func waitForSessions(t *testing.T, ts *httptest.Server, token string, want ...string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		got := listSessionIDs(t, ts, token)
		if len(got) == len(want) {
			match := true
			for _, id := range want {
				if !got[id] {
					match = false
					break
				}
			}
			if match {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("sessions = %v, want %v", got, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// sessionMetaMap fetches the live session list as a map keyed by session id.
func sessionMetaMap(t *testing.T, ts *httptest.Server, token string) map[string]SessionView {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Sessions []SessionView `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	m := map[string]SessionView{}
	for _, s := range out.Sessions {
		m[s.ID] = s
	}
	return m
}

// One host re-announcing its sessions must not clear the sessions announced
// by another host of the same account (desktop + laptop) (#169).
func TestHostAnnouncesDoNotClearOtherHosts(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "multi-host")
	hostA, secretA := registerHost(t, ts, token, "desktop")
	hostB, secretB := registerHost(t, ts, token, "laptop")

	connA := dialHostWS(t, ts, hostA, secretA)
	connB := dialHostWS(t, ts, hostB, secretB)

	announceSessions(t, connA, SessionMeta{ID: "a1", Name: "a1", CLI: "claude", Status: "running"})
	announceSessions(t, connB, SessionMeta{ID: "b1", Name: "b1", CLI: "claude", Status: "running"})
	waitForSessions(t, ts, token, "a1", "b1")

	// A replaces its own list; B's sessions must survive.
	announceSessions(t, connA, SessionMeta{ID: "a2", Name: "a2", CLI: "claude", Status: "running"})
	waitForSessions(t, ts, token, "a2", "b1")
}

// Sessions announced without a timestamp must get a live LastSeenAt stamped
// by the relay; otherwise /api/sessions returns Go's zero time and clients
// show "739835d ago".
func TestAnnouncedSessionsGetLiveLastSeenAt(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "lastseen")
	hostID, secret := registerHost(t, ts, token, "laptop")

	conn := dialHostWS(t, ts, hostID, secret)
	announceSessions(t, conn, SessionMeta{ID: "s1", Name: "s1", CLI: "claude", Status: "running"})
	waitForSessions(t, ts, token, "s1")

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Sessions []SessionMeta `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	for _, s := range out.Sessions {
		if s.ID != "s1" {
			continue
		}
		if s.LastSeenAt.IsZero() || time.Since(s.LastSeenAt) > time.Minute {
			t.Fatalf("expected live lastSeenAt, got %v", s.LastSeenAt)
		}
	}
}

// Re-announcing one session must not refresh the LastSeenAt of the others:
// every session carries its own timestamp and the relay has to keep it. The
// relay used to stamp every announced session with now, so activity in one
// session made the whole list show "just now".
func TestAnnouncedSessionsKeepTheirOwnLastSeenAt(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "lastseen-own")
	hostID, secret := registerHost(t, ts, token, "laptop")
	conn := dialHostWS(t, ts, hostID, secret)

	old1 := time.Now().Add(-30 * time.Minute)
	old2 := time.Now().Add(-5 * time.Minute)
	announceSessions(t, conn,
		SessionMeta{ID: "s1", CLI: "claude", Status: "running", LastSeenAt: old1},
		SessionMeta{ID: "s2", CLI: "claude", Status: "running", LastSeenAt: old2},
	)
	waitForSessions(t, ts, token, "s1", "s2")

	// s2 just had activity: only its timestamp advances on re-announce.
	announceSessions(t, conn,
		SessionMeta{ID: "s1", CLI: "claude", Status: "running", LastSeenAt: old1},
		SessionMeta{ID: "s2", CLI: "claude", Status: "running", LastSeenAt: time.Now()},
	)

	deadline := time.Now().Add(3 * time.Second)
	for {
		got := sessionMetaMap(t, ts, token)
		if len(got) == 2 && got["s2"].LastSeenAt.After(old2.Add(time.Minute)) {
			if delta := got["s1"].LastSeenAt.Sub(old1); delta > time.Minute || delta < -time.Minute {
				t.Fatalf("s1 lastSeenAt refreshed to %v; want %v", got["s1"].LastSeenAt, old1)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("second announce never landed: s1=%v s2=%v", got["s1"].LastSeenAt, got["s2"].LastSeenAt)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// The client polls /api/sessions every few seconds; a random order (Go map
// range) made the list reshuffle on every poll. The relay must return a
// stable order: most recently active first.
func TestSessionsListOrderedByLastSeenAt(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "sorted-sessions")
	hostID, secret := registerHost(t, ts, token, "laptop")
	conn := dialHostWS(t, ts, hostID, secret)

	old := time.Now().Add(-10 * time.Minute)
	mid := time.Now().Add(-5 * time.Minute)
	announceSessions(t, conn,
		SessionMeta{ID: "old", CLI: "claude", Status: "running", LastSeenAt: old},
		SessionMeta{ID: "mid", CLI: "claude", Status: "running", LastSeenAt: mid},
		SessionMeta{ID: "fresh", CLI: "claude", Status: "running", LastSeenAt: time.Now()},
	)

	var ids []string
	deadline := time.Now().Add(3 * time.Second)
	for {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			Sessions []SessionMeta `json:"sessions"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			t.Fatal(err)
		}
		resp.Body.Close()
		if len(out.Sessions) == 3 {
			ids = make([]string, 0, 3)
			for _, s := range out.Sessions {
				ids = append(ids, s.ID)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("sessions never fully announced (got %d)", len(out.Sessions))
		}
		time.Sleep(20 * time.Millisecond)
	}
	want := []string{"fresh", "mid", "old"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("session order = %v, want %v", ids, want)
	}
}

func TestSessionClientMetaUpdateAndJoin(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "meta-user")
	hostID, secret := registerHost(t, ts, token, "laptop")
	conn := dialHostWS(t, ts, hostID, secret)
	announceSessions(t, conn, SessionMeta{ID: "s1", Name: "demo", CLI: "claude", Status: "running", LastSeenAt: time.Now()})
	waitForSessions(t, ts, token, "s1")

	putMeta := func(body string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/sessions/s1/meta", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	resp := putMeta(`{"hostId":"` + hostID + `","displayName":"我的会话","hidden":true}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put meta status %d", resp.StatusCode)
	}
	resp.Body.Close()

	got := sessionMetaMap(t, ts, token)
	s := got["s1"]
	if s.DisplayName != "我的会话" || !s.Hidden {
		t.Fatalf("meta not joined: %+v", s)
	}
	if s.Name != "demo" {
		t.Fatalf("host-announced name lost: %+v", s)
	}

	// Rename + unhide: the upsert must merge, not reset the other field.
	resp = putMeta(`{"hostId":"` + hostID + `","displayName":"renamed","hidden":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put meta status %d", resp.StatusCode)
	}
	resp.Body.Close()
	got = sessionMetaMap(t, ts, token)
	s = got["s1"]
	if s.DisplayName != "renamed" || s.Hidden {
		t.Fatalf("meta not updated: %+v", s)
	}

	// Clear the custom name: empty string removes it.
	resp = putMeta(`{"hostId":"` + hostID + `","displayName":""}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put meta status %d", resp.StatusCode)
	}
	resp.Body.Close()
	got = sessionMetaMap(t, ts, token)
	if got["s1"].DisplayName != "" {
		t.Fatalf("display name not cleared: %+v", got["s1"])
	}
}

func TestSessionClientMetaForbiddenForOtherUser(t *testing.T) {
	_, ts := newTestHub(t)
	tokenOwner := registerUser(t, ts, "meta-owner")
	hostID, _ := registerHost(t, ts, tokenOwner, "laptop")
	tokenIntruder := registerUser(t, ts, "meta-intruder")

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/sessions/s1/meta",
		strings.NewReader(`{"hostId":"`+hostID+`","displayName":"hack"}`))
	req.Header.Set("Authorization", "Bearer "+tokenIntruder)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for foreign host, got %d", resp.StatusCode)
	}
}

// A reconnecting host registers its new connection before the old one's
// deferred removeHost runs; the old cleanup must not wipe the sessions the
// new connection just announced (#169).
func TestReconnectDoesNotWipeReannouncedSessions(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "reconnect")
	hostID, secret := registerHost(t, ts, token, "laptop")

	conn1 := dialHostWS(t, ts, hostID, secret)
	announceSessions(t, conn1, SessionMeta{ID: "s1", Name: "s1", CLI: "claude", Status: "running"})
	waitForSessions(t, ts, token, "s1")

	// New connection supersedes the old one (e.g. after a network flap), and
	// the daemon re-announces. The old connection's deferred removeHost then
	// runs and must leave the fresh entries alone.
	conn2 := dialHostWS(t, ts, hostID, secret)
	announceSessions(t, conn2, SessionMeta{ID: "s2", Name: "s2", CLI: "claude", Status: "running"})
	waitForSessions(t, ts, token, "s2")

	// Give the old connection's read loop time to notice the close and run
	// its deferred removeHost, then confirm the sessions are still there.
	conn1.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		if _, _, err := conn1.ReadMessage(); err != nil {
			break
		}
	}
	time.Sleep(100 * time.Millisecond)
	waitForSessions(t, ts, token, "s2")
}

// A plain disconnect (no replacement connection) still clears the host's
// sessions from the live list.
func TestHostDisconnectClearsSessions(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "disconnect")
	hostID, secret := registerHost(t, ts, token, "laptop")

	conn := dialHostWS(t, ts, hostID, secret)
	announceSessions(t, conn, SessionMeta{ID: "s1", Name: "s1", CLI: "claude", Status: "running"})
	waitForSessions(t, ts, token, "s1")

	_ = conn.Close()
	waitForSessions(t, ts, token)
}

// fetchHostOnline reads the hostOnline flag from /api/sessions.
func fetchHostOnline(t *testing.T, ts *httptest.Server, token string) bool {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		HostOnline bool `json:"hostOnline"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out.HostOnline
}

// /api/sessions must tell the client whether any of the user's hosts is
// currently connected, so an empty list can be rendered as "daemon offline"
// instead of the misleading "no sessions, go run one" empty state (#174).
func TestSessionsReportsHostOnline(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "host-online")
	hostID, secret := registerHost(t, ts, token, "laptop")

	// Host registered but not connected: offline.
	if fetchHostOnline(t, ts, token) {
		t.Fatal("hostOnline should be false before the host connects")
	}

	conn := dialHostWS(t, ts, hostID, secret)
	deadline := time.Now().Add(3 * time.Second)
	for !fetchHostOnline(t, ts, token) {
		if time.Now().After(deadline) {
			t.Fatal("hostOnline did not turn true after the host connected")
		}
		time.Sleep(20 * time.Millisecond)
	}

	_ = conn.Close()
	deadline = time.Now().Add(3 * time.Second)
	for fetchHostOnline(t, ts, token) {
		if time.Now().After(deadline) {
			t.Fatal("hostOnline did not turn false after the host disconnected")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// When a new connection with the same host credentials replaces an old one,
// the relay must send a superseded frame first so the old daemon stops
// reconnecting instead of kick-looping (#169).
func TestSupersededFrameSentToReplacedHost(t *testing.T) {
	_, ts := newTestHub(t)
	token := registerUser(t, ts, "supersede")
	hostID, secret := registerHost(t, ts, token, "laptop")

	conn1 := dialHostWS(t, ts, hostID, secret)
	_ = dialHostWS(t, ts, hostID, secret)

	conn1.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("old connection got no superseded frame: %v", err)
	}
	var fr hostFrame
	if err := json.Unmarshal(raw, &fr); err != nil {
		t.Fatal(err)
	}
	if fr.Kind != "superseded" {
		t.Fatalf("old connection got kind %q, want superseded", fr.Kind)
	}
	// The relay drops the old connection right after the notice.
	conn1.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn1.ReadMessage(); err == nil {
		t.Fatal("old connection still open after superseded")
	}
}

// connectViewer pairs a device and dials a viewer for session s1 (announced
// by hostConn), returning the viewer connection and its relay viewer id.
func connectViewer(t *testing.T, ts *httptest.Server, token, hostID string, hostConn *websocket.Conn) (*websocket.Conn, string) {
	t.Helper()
	resp, err := authPost(ts, "/api/pairings", token, `{"hostId":"`+hostID+`","curve":"p256","publicKey":"AAA"}`)
	if err != nil {
		t.Fatal(err)
	}
	var pr struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = authPost(ts, "/api/pair", token, `{"code":"`+pr.Code+`","name":"phone","curve":"p256","publicKey":"BBB"}`)
	if err != nil {
		t.Fatal(err)
	}
	var pair struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pair); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	viewerURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?session=s1&device=" + pair.DeviceID + "&eph=EPH&token=" + token
	vc, _, err := websocket.DefaultDialer.Dial(viewerURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { vc.Close() })

	hostConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := hostConn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var join hostFrame
	if err := json.Unmarshal(raw, &join); err != nil {
		t.Fatal(err)
	}
	if join.Kind != "join" || join.ViewerID == "" {
		t.Fatalf("unexpected join frame: %+v", join)
	}
	return vc, join.ViewerID
}

// Issue #173: the daemon asks the relay to close a viewer's browser
// connection ("kick") after dropping it, so the client reconnects and
// replays instead of hanging on a silent socket.
func TestKickFrameClosesViewer(t *testing.T) {
	h, ts := newTestHub(t)
	token := registerUser(t, ts, "kick")
	hostID, secret := registerHost(t, ts, token, "laptop")
	hostConn := dialHostWS(t, ts, hostID, secret)
	announceSessions(t, hostConn, SessionMeta{ID: "s1", Name: "s1", CLI: "claude", Status: "running"})
	vc, viewerID := connectViewer(t, ts, token, hostID, hostConn)

	kick, _ := json.Marshal(hostFrame{Kind: protocol.RelayFrameKick, ViewerID: viewerID})
	if err := hostConn.WriteMessage(websocket.TextMessage, kick); err != nil {
		t.Fatal(err)
	}

	vc.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		if _, _, err := vc.ReadMessage(); err != nil {
			break
		}
	}
	h.mu.Lock()
	_, ok := h.viewers[viewerID]
	h.mu.Unlock()
	if ok {
		t.Fatal("kicked viewer still registered")
	}
}

// Issue #173: a viewer send-buffer overflow must drop the connection (the
// payload is encrypted, so the relay cannot tell critical from noise) and
// log it, instead of silently dropping the message.
func TestViewerSendBufferFullDropsViewer(t *testing.T) {
	var buf bytes.Buffer
	h, err := New(log.New(&buf, "", 0), t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	host := &hostConn{id: "h1", send: make(chan []byte, 1), done: make(chan struct{})}
	v := &viewerConn{id: "v1", sessionID: "s1", host: host, send: make(chan []byte, 1), done: make(chan struct{})}
	h.mu.Lock()
	h.viewers["v1"] = v
	h.mu.Unlock()
	v.send <- []byte("filler") // buffer full; writeLoop not running

	h.sendToViewer(v, []byte("payload"))

	h.mu.Lock()
	_, ok := h.viewers["v1"]
	h.mu.Unlock()
	if ok {
		t.Fatal("overflowing viewer still registered")
	}
	select {
	case <-v.done:
	default:
		t.Fatal("overflowing viewer not closed")
	}
	if !strings.Contains(buf.String(), "send buffer full") ||
		!strings.Contains(buf.String(), "v1") ||
		!strings.Contains(buf.String(), "s1") {
		t.Fatalf("missing warn log: %q", buf.String())
	}
}

// Issue #173: same policy on the host-bound queue (viewer → daemon
// direction): overflow closes the host connection so the daemon reconnects
// and everything replays.
func TestHostSendBufferFullClosesHost(t *testing.T) {
	var buf bytes.Buffer
	h, err := New(log.New(&buf, "", 0), t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	host := &hostConn{id: "h1", send: make(chan []byte, 1), done: make(chan struct{})}
	host.send <- []byte("filler")

	h.hostSend(host, hostFrame{Kind: "viewer", ViewerID: "v1"})

	select {
	case <-host.done:
	default:
		t.Fatal("overflowing host connection not closed")
	}
	if !strings.Contains(buf.String(), "host send buffer full") || !strings.Contains(buf.String(), "h1") {
		t.Fatalf("missing warn log: %q", buf.String())
	}
}

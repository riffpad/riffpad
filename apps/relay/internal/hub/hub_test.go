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
	"testing"
	"time"

	"github.com/gorilla/websocket"
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
	h.oauthStates["devstate"] = oauthState{expires: time.Now().Add(time.Minute), device: dev.UserCode}
	h.mu.Unlock()
	resp, err = http.Get(ts.URL + "/api/auth/github/callback?code=testcode&state=devstate")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "授权成功") {
		t.Fatalf("authorize status %d: %s", resp.StatusCode, body)
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

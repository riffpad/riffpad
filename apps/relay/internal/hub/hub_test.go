package hub

import (
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

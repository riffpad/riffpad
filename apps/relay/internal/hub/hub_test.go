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

func TestRelayRoutesViewerToHost(t *testing.T) {
	h := New(log.New(io.Discard, "", 0), "regkey", t.TempDir())
	ts := httptest.NewServer(h.Handler())
	defer ts.Close()

	// 0. Register a host and connect with its per-host secret.
	resp, err := http.Post(ts.URL+"/api/hosts/register", "application/json", strings.NewReader(`{"name":"laptop","registrationKey":"regkey"}`))
	if err != nil {
		t.Fatal(err)
	}
	var reg struct {
		HostID     string `json:"hostId"`
		HostSecret string `json:"hostSecret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// 1. Host connects and announces a session.
	hostURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/host?hostId=" + reg.HostID + "&token=" + reg.HostSecret
	hostConn, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer hostConn.Close()
	sessions := []SessionMeta{{ID: "s1", Name: "demo", CLI: "claude", Cwd: "/tmp", Status: "running"}}
	fr, _ := json.Marshal(hostFrame{Kind: "sessions", Sessions: sessions})
	if err := hostConn.WriteMessage(websocket.TextMessage, fr); err != nil {
		t.Fatal(err)
	}

	// 2. Create pairing and pair a device.
	resp, err = http.Post(ts.URL+"/api/pairings", "application/json", strings.NewReader(`{"hostId":"`+reg.HostID+`","curve":"p256","publicKey":"AAA"}`))
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
	resp, err = http.Post(ts.URL+"/api/pair", "application/json", strings.NewReader(`{"code":"`+pr.Code+`","name":"phone","curve":"p256","publicKey":"BBB"}`))
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

	// 3. Viewer connects; host must receive a join frame.
	viewerURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?session=s1&device=" + pair.DeviceID + "&eph=EPH"
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

	// 4. Host sends hello data; viewer receives it.
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

	// 5. Viewer sends an envelope; host receives it wrapped.
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
}

func TestPairingRequiresOnlineHost(t *testing.T) {
	h := New(log.New(io.Discard, "", 0), "", t.TempDir())
	ts := httptest.NewServer(h.Handler())
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/pairings", "application/json", strings.NewReader(`{"hostId":"offline","curve":"p256","publicKey":"AAA"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for offline host, got %d", resp.StatusCode)
	}
}

func TestHostRegistrationAndPersistence(t *testing.T) {
	dir := t.TempDir()
	h := New(log.New(io.Discard, "", 0), "regkey", dir)
	ts := httptest.NewServer(h.Handler())

	resp, err := http.Post(ts.URL+"/api/hosts/register", "application/json", strings.NewReader(`{"name":"laptop","registrationKey":"wrong"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong registration key, got %d", resp.StatusCode)
	}

	resp, err = http.Post(ts.URL+"/api/hosts/register", "application/json", strings.NewReader(`{"name":"laptop","registrationKey":"regkey"}`))
	if err != nil {
		t.Fatal(err)
	}
	var reg struct {
		HostID     string `json:"hostId"`
		HostSecret string `json:"hostSecret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if reg.HostID == "" || reg.HostSecret == "" {
		t.Fatal("registration returned empty credentials")
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/host?hostId=" + reg.HostID + "&token=" + reg.HostSecret
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("host ws with secret should connect: %v", err)
	}
	conn.Close()

	badURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/host?hostId=" + reg.HostID + "&token=wrong"
	if _, _, err := websocket.DefaultDialer.Dial(badURL, nil); err == nil {
		t.Fatal("host ws with wrong secret should be rejected")
	}

	// A fresh hub on the same data dir must still know the host.
	h2 := New(log.New(io.Discard, "", 0), "regkey", dir)
	if _, ok := h2.hostRecords[reg.HostID]; !ok {
		t.Fatal("host record not persisted across relay restart")
	}
}

func TestPairingRateLimit(t *testing.T) {
	h := New(log.New(io.Discard, "", 0), "regkey", t.TempDir())
	ts := httptest.NewServer(h.Handler())
	defer ts.Close()

	var got429 bool
	for i := 0; i < 11; i++ {
		resp, _ := http.Post(ts.URL+"/api/pair", "application/json", strings.NewReader(`{"code":"WRONG","name":"x","curve":"p256","publicKey":"BBB"}`))
		if resp.StatusCode == http.StatusTooManyRequests {
			got429 = true
		}
		resp.Body.Close()
	}
	if !got429 {
		t.Fatal("expected a 429 after exceeding the per-IP rate limit")
	}
}

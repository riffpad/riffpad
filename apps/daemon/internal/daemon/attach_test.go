package daemon

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/packages/protocol"
)

func TestAttachHookFlow(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	factory := func(_ context.Context, req adapter.CreateRequest) (adapter.Session, error) {
		f := &fakeSession{id: req.ID, events: make(chan protocol.Event, 16), approvals: make(chan string, 1), prompts: make(chan string, 1), stopCalled: make(chan struct{}, 1)}
		return f, nil
	}
	srv := New(cfg, keys, dir, logger, factory)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Claude session starts -> hook creates an attached session.
	post := func(path, body string) *http.Response {
		t.Helper()
		resp, err := http.Post(ts.URL+path, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	resp := post("/hooks/claude/session-start", `{"hook_event_name":"SessionStart","session_id":"claude-sess-1","cwd":"/tmp/proj"}`)
	resp.Body.Close()

	sessResp, err := http.Get(ts.URL + "/api/sessions")
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		Sessions []struct {
			ID   string `json:"id"`
			CLI  string `json:"cli"`
			Name string `json:"name"`
		} `json:"sessions"`
	}
	if err := json.NewDecoder(sessResp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	sessResp.Body.Close()
	if len(list.Sessions) != 1 || list.Sessions[0].ID != "claude-sess-1" || list.Sessions[0].CLI != "claude (attach)" {
		t.Fatalf("unexpected sessions: %+v", list.Sessions)
	}

	// 2. Pair a web client and connect to the attached session.
	pairResp, err := http.Post(ts.URL+"/api/pairings", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var pr struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(pairResp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	pairResp.Body.Close()
	clientID, _ := protocol.GenerateKeyPair(protocol.CurveP256)
	body, _ := json.Marshal(map[string]string{"code": pr.Code, "name": "t", "curve": "p256", "publicKey": protocol.EncodeKey(clientID.PublicKey)})
	pairResp, err = http.Post(ts.URL+"/api/pair", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	var pair struct {
		DeviceID        string `json:"deviceId"`
		ServerPublicKey string `json:"serverPublicKey"`
	}
	if err := json.NewDecoder(pairResp.Body).Decode(&pair); err != nil {
		t.Fatal(err)
	}
	pairResp.Body.Close()
	serverPub, _ := protocol.DecodeKey(pair.ServerPublicKey)
	deviceSecret, _ := protocol.NewDeviceSecret(clientID, serverPub)

	eph, _ := protocol.GenerateKeyPair(protocol.CurveP256)
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?device=" + pair.DeviceID + "&session=claude-sess-1&eph=" + protocol.EncodeKey(eph.PublicKey)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var hello protocol.Hello
	if err := json.Unmarshal(data, &hello); err != nil {
		t.Fatal(err)
	}
	serverEphPub, _ := protocol.DecodeKey(hello.ServerEphPub)
	ephSecret, _ := protocol.ECDH(eph, serverEphPub)
	key, _ := protocol.DeriveSessionKey(deviceSecret, ephSecret, "claude-sess-1")

	readEvent := func() protocol.Event {
		t.Helper()
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var env protocol.Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatal(err)
		}
		plain, err := env.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		var ev protocol.Event
		if err := json.Unmarshal(plain, &ev); err != nil {
			t.Fatal(err)
		}
		return ev
	}
	ev := readEvent()
	if ev.Type != protocol.EventSessionStart {
		t.Fatalf("expected session_start, got %s", ev.Type)
	}

	// 3. Tool use hook -> tool_call event.
	resp = post("/hooks/claude/pre-tool-use", `{"hook_event_name":"PreToolUse","session_id":"claude-sess-1","tool_use_id":"tu1","tool_use":{"name":"Bash","input":{"command":"ls"}}}`)
	resp.Body.Close()
	ev = readEvent()
	if ev.Type != protocol.EventToolCall {
		t.Fatalf("expected tool_call, got %s", ev.Type)
	}

	// 3b. User prompt and assistant message hooks flow into the timeline.
	resp = post("/hooks/claude/user-prompt-submit", `{"hook_event_name":"UserPromptSubmit","session_id":"claude-sess-1","prompt":"你好"}`)
	resp.Body.Close()
	ev = readEvent()
	if ev.Type != protocol.EventUserMessage {
		t.Fatalf("expected user_message, got %s", ev.Type)
	}
	var up protocol.PromptPayload
	if err := ev.DecodePayload(&up); err != nil {
		t.Fatal(err)
	}
	if up.Text != "你好" {
		t.Fatalf("unexpected prompt %q", up.Text)
	}

	resp = post("/hooks/claude/message-display", `{"hook_event_name":"MessageDisplay","session_id":"claude-sess-1","message_id":"m1","delta":"你好！","final":true}`)
	resp.Body.Close()
	ev = readEvent()
	if ev.Type != protocol.EventAgentMessage {
		t.Fatalf("expected agent_message, got %s", ev.Type)
	}

	// 4. PermissionRequest hook blocks until the phone approves.
	permDone := make(chan string, 1)
	go func() {
		resp := post("/hooks/claude/permission", `{"hook_event_name":"PermissionRequest","session_id":"claude-sess-1","tool_use_id":"tu2","tool_use":{"name":"Bash","input":{"command":"rm -rf build"}}}`)
		defer resp.Body.Close()
		var out struct {
			HookSpecificOutput struct {
				Decision struct {
					Behavior string `json:"behavior"`
				} `json:"decision"`
			} `json:"hookSpecificOutput"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		permDone <- out.HookSpecificOutput.Decision.Behavior
	}()

	ev = readEvent()
	if ev.Type != protocol.EventApprovalReq {
		t.Fatalf("expected approval_request, got %s", ev.Type)
	}
	var ap protocol.ApprovalRequestPayload
	if err := ev.DecodePayload(&ap); err != nil {
		t.Fatal(err)
	}
	if ap.Action != "Bash" || ap.Summary != "rm -rf build" {
		t.Fatalf("unexpected approval: %+v", ap)
	}

	// 5. Approve from the client.
	payload, _ := json.Marshal(protocol.ApprovalResponsePayload{RequestID: ap.RequestID, Decision: "approve"})
	msg := protocol.Event{ID: protocol.NewID(), SessionID: "claude-sess-1", Timestamp: time.Now().UnixMilli(), Type: protocol.EventApprovalResp, Payload: payload}
	sendEncrypted(t, conn, "claude-sess-1", key, msg)

	select {
	case d := <-permDone:
		if d != "allow" {
			t.Fatalf("expected allow, got %q", d)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("permission hook did not resolve")
	}
}

func TestFindClaudePane(t *testing.T) {
	out := "%0\tclaude\t/tmp/a\n" +
		"%1\tnode /usr/local/bin/claude\t/tmp/b\n" +
		"%2\tzsh\t/tmp/a\n" +
		"%3\tclaude\t/tmp/a\n"
	if got := findClaudePane(out, "/tmp/a"); got != "%0" {
		t.Fatalf("expected %%0, got %q", got)
	}
	if got := findClaudePane(out, "/tmp/b"); got != "%1" {
		t.Fatalf("expected %%1, got %q", got)
	}
	if got := findClaudePane(out, "/tmp/nope"); got != "" {
		t.Fatalf("expected no pane, got %q", got)
	}
}

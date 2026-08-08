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
		return authRequest(t, http.MethodPost, ts.URL+path, cfg.LocalToken, strings.NewReader(body))
	}
	resp := post("/hooks/claude/session-start", `{"hook_event_name":"SessionStart","session_id":"claude-sess-1","cwd":"/tmp/proj"}`)
	resp.Body.Close()

	sessResp := authRequest(t, http.MethodGet, ts.URL+"/api/sessions", cfg.LocalToken, nil)
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
	pairResp := authRequest(t, http.MethodPost, ts.URL+"/api/pairings", cfg.LocalToken, nil)
	var pr struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(pairResp.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	pairResp.Body.Close()
	clientID, _ := protocol.GenerateKeyPair(protocol.CurveP256)
	body, _ := json.Marshal(map[string]string{"code": pr.Code, "name": "t", "curve": "p256", "publicKey": protocol.EncodeKey(clientID.PublicKey)})
	pairResp = authRequest(t, http.MethodPost, ts.URL+"/api/pair", cfg.LocalToken, strings.NewReader(string(body)))
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
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?device=" + pair.DeviceID + "&session=claude-sess-1&eph=" + protocol.EncodeKey(eph.PublicKey) + "&token=" + cfg.LocalToken
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

	// 5b. The resolution is broadcast to every viewer and lands in the session
	// history, so other tabs grey out the card and replays keep it settled (#171).
	ev = readEvent()
	if ev.Type != protocol.EventApprovalResolved {
		t.Fatalf("expected approval_resolved, got %s", ev.Type)
	}
	var arp protocol.ApprovalResolvedPayload
	if err := ev.DecodePayload(&arp); err != nil {
		t.Fatal(err)
	}
	if arp.RequestID != ap.RequestID || arp.Decision != "approve" || arp.DeviceID != pair.DeviceID {
		t.Fatalf("unexpected approval_resolved: %+v", arp)
	}
	sess := srv.getSession("claude-sess-1")
	inHistory := false
	for _, hev := range sess.snapshot() {
		if hev.Type == protocol.EventApprovalResolved {
			inHistory = true
		}
	}
	if !inHistory {
		t.Fatal("approval_resolved missing from session history")
	}

	// 5c. A second response for the same requestID loses the race: the daemon
	// acks only the sender with an "expired" notify, no second resolved event.
	payload, _ = json.Marshal(protocol.ApprovalResponsePayload{RequestID: ap.RequestID, Decision: "reject"})
	msg = protocol.Event{ID: protocol.NewID(), SessionID: "claude-sess-1", Timestamp: time.Now().UnixMilli(), Type: protocol.EventApprovalResp, Payload: payload}
	sendEncrypted(t, conn, "claude-sess-1", key, msg)
	ev = readEvent()
	if ev.Type != protocol.EventNotify {
		t.Fatalf("expected notify, got %s", ev.Type)
	}
	var np protocol.NotifyPayload
	if err := ev.DecodePayload(&np); err != nil {
		t.Fatal(err)
	}
	if np.Level != "error" || np.RequestID != ap.RequestID {
		t.Fatalf("unexpected notify: %+v", np)
	}
}

func TestAttachPermissionTimeoutResolves(t *testing.T) {
	old := approvalHookTimeout
	approvalHookTimeout = 50 * time.Millisecond
	defer func() { approvalHookTimeout = old }()

	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	post := func(path, body string) *http.Response {
		t.Helper()
		return authRequest(t, http.MethodPost, ts.URL+path, cfg.LocalToken, strings.NewReader(body))
	}
	resp := post("/hooks/claude/session-start", `{"hook_event_name":"SessionStart","session_id":"claude-sess-2","cwd":"/tmp/proj"}`)
	resp.Body.Close()

	// Nobody answers the permission prompt: the hook must time out with deny
	// and settle the card for every viewer via approval_resolved (#171).
	resp = post("/hooks/claude/permission", `{"hook_event_name":"PermissionRequest","session_id":"claude-sess-2","tool_use_id":"tu9","tool_use":{"name":"Bash","input":{"command":"rm -rf build"}}}`)
	defer resp.Body.Close()
	var out struct {
		HookSpecificOutput struct {
			Decision struct {
				Behavior string `json:"behavior"`
			} `json:"decision"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.HookSpecificOutput.Decision.Behavior != "deny" {
		t.Fatalf("expected deny after timeout, got %q", out.HookSpecificOutput.Decision.Behavior)
	}

	sess := srv.getSession("claude-sess-2")
	var resolved *protocol.ApprovalResolvedPayload
	for _, hev := range sess.snapshot() {
		if hev.Type != protocol.EventApprovalResolved {
			continue
		}
		var p protocol.ApprovalResolvedPayload
		if err := hev.DecodePayload(&p); err != nil {
			t.Fatal(err)
		}
		resolved = &p
	}
	if resolved == nil {
		t.Fatal("approval_resolved missing from session history after timeout")
	}
	if resolved.Decision != "reject" || resolved.DeviceID != "" {
		t.Fatalf("unexpected timeout resolution: %+v", resolved)
	}
	srv.mu.Lock()
	pending := len(srv.pendingHooks)
	srv.mu.Unlock()
	if pending != 0 {
		t.Fatalf("pendingHooks leaked after timeout: %d entries", pending)
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

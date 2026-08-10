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

	// 3. Tool use hook -> Bash renders as a single "$ cmd" row: started
	// without an exit code, completed with one. No tool_call row (which
	// would show a duplicate spinner + completed pair).
	resp = post("/hooks/claude/pre-tool-use", `{"hook_event_name":"PreToolUse","session_id":"claude-sess-1","tool_use_id":"tu1","tool_use":{"name":"Bash","input":{"command":"ls"}}}`)
	resp.Body.Close()
	ev = readEvent()
	if ev.Type != protocol.EventCommand {
		t.Fatalf("expected command, got %s", ev.Type)
	}
	var cp protocol.CommandPayload
	if err := ev.DecodePayload(&cp); err != nil {
		t.Fatal(err)
	}
	if cp.Command != "ls" || cp.ExitCode != nil {
		t.Fatalf("expected running command ls, got %+v", cp)
	}
	resp = post("/hooks/claude/post-tool-use", `{"hook_event_name":"PostToolUse","session_id":"claude-sess-1","tool_use_id":"tu1","tool_use":{"name":"Bash","input":{"command":"ls"}}}`)
	resp.Body.Close()
	ev = readEvent()
	if ev.Type != protocol.EventCommand {
		t.Fatalf("expected completed command, got %s", ev.Type)
	}
	if err := ev.DecodePayload(&cp); err != nil {
		t.Fatal(err)
	}
	if cp.Command != "ls" || cp.ExitCode == nil || *cp.ExitCode != 0 {
		t.Fatalf("expected completed command ls with exit 0, got %+v", cp)
	}

	// 3b. User prompt and assistant message hooks flow into the timeline.
	// A submitted prompt flips a waiting session back to running so the
	// client shows the activity indicator (#253).
	srv.mu.Lock()
	attached := srv.sessions["claude-sess-1"]
	srv.mu.Unlock()
	attached.mu.Lock()
	attached.status = protocol.StatusWaitingInput
	attached.mu.Unlock()

	resp = post("/hooks/claude/user-prompt-submit", `{"hook_event_name":"UserPromptSubmit","session_id":"claude-sess-1","prompt":"你好"}`)
	resp.Body.Close()
	ev = readEvent()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected running agent_status after prompt, got %s", ev.Type)
	}
	var st protocol.AgentStatusPayload
	if err := ev.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusRunning {
		t.Fatalf("expected running, got %q", st.Status)
	}
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

	// Simulate a finished turn (waiting for input): the next MessageDisplay
	// must flip the session back to running so the client shows the activity
	// indicator (#253).
	attached.mu.Lock()
	attached.status = protocol.StatusWaitingInput
	attached.mu.Unlock()

	resp = post("/hooks/claude/message-display", `{"hook_event_name":"MessageDisplay","session_id":"claude-sess-1","message_id":"m1","delta":"你好！","final":true}`)
	resp.Body.Close()
	ev = readEvent()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected running agent_status before agent_message, got %s", ev.Type)
	}
	if err := ev.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusRunning {
		t.Fatalf("expected running, got %q", st.Status)
	}
	ev = readEvent()
	if ev.Type != protocol.EventAgentMessage {
		t.Fatalf("expected agent_message, got %s", ev.Type)
	}

	// idle_prompt notification flips the session back to waiting_input.
	resp = post("/hooks/claude/notification", `{"hook_event_name":"Notification","session_id":"claude-sess-1","notification":{"notification_type":"idle_prompt","message":"Claude is waiting for your input"}}`)
	resp.Body.Close()
	ev = readEvent()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected waiting_input agent_status, got %s", ev.Type)
	}
	if err := ev.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusWaitingInput {
		t.Fatalf("expected waiting_input, got %q", st.Status)
	}
	ev = readEvent()
	if ev.Type != protocol.EventNotify {
		t.Fatalf("expected notify, got %s", ev.Type)
	}

	// Stop hook also flips a running session back to waiting_input — and
	// unlike idle_prompt it fires as soon as the previous turn finishes, so
	// the client clears the activity indicator without a ~60s lag (#257).
	attached.mu.Lock()
	attached.status = protocol.StatusRunning
	attached.mu.Unlock()
	resp = post("/hooks/claude/stop", `{"hook_event_name":"Stop","session_id":"claude-sess-1","reason":"turn_end"}`)
	resp.Body.Close()
	ev = readEvent()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected waiting_input agent_status after stop hook, got %s", ev.Type)
	}
	if err := ev.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusWaitingInput {
		t.Fatalf("expected waiting_input after stop hook, got %q", st.Status)
	}
	// A Stop hook on an already-waiting session must not emit a duplicate.
	waitingStatuses := func(events []protocol.Event) int {
		n := 0
		for _, hev := range events {
			if hev.Type != protocol.EventAgentStatus {
				continue
			}
			var p protocol.AgentStatusPayload
			if hev.DecodePayload(&p) == nil && p.Status == protocol.StatusWaitingInput {
				n++
			}
		}
		return n
	}
	before := waitingStatuses(attached.snapshot())
	resp = post("/hooks/claude/stop", `{"hook_event_name":"Stop","session_id":"claude-sess-1","reason":"turn_end"}`)
	resp.Body.Close()
	if after := waitingStatuses(attached.snapshot()); after != before {
		t.Fatalf("expected no duplicate waiting_input event, got %d -> %d", before, after)
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

// TestAttachToolCompletionKeepsInPlaceIdentity covers #210 for the hook path:
// the completed tool_call must carry the same summary/args as the started one
// so the client's in-place merge key matches (no duplicate spinner + completed
// rows).
func TestAttachToolCompletionKeepsInPlaceIdentity(t *testing.T) {
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
	post := func(path, body string) {
		t.Helper()
		resp := authRequest(t, http.MethodPost, ts.URL+path, cfg.LocalToken, strings.NewReader(body))
		resp.Body.Close()
	}

	post("/hooks/claude/session-start", `{"hook_event_name":"SessionStart","session_id":"claude-sess-3","cwd":"/tmp/proj"}`)
	post("/hooks/claude/pre-tool-use", `{"hook_event_name":"PreToolUse","session_id":"claude-sess-3","tool_use_id":"tu3","tool_use":{"name":"Write","input":{"file_path":"/tmp/a.txt"}}}`)
	post("/hooks/claude/post-tool-use", `{"hook_event_name":"PostToolUse","session_id":"claude-sess-3","tool_use_id":"tu3","tool_use":{"name":"Write","input":{"file_path":"/tmp/a.txt"}}}`)

	sess := srv.getSession("claude-sess-3")
	var started, completed *protocol.ToolCallPayload
	var fileChanged bool
	for _, hev := range sess.snapshot() {
		switch hev.Type {
		case protocol.EventToolCall:
			var p protocol.ToolCallPayload
			if err := hev.DecodePayload(&p); err != nil {
				t.Fatal(err)
			}
			switch p.Status {
			case "started":
				started = &p
			case "completed":
				completed = &p
			}
		case protocol.EventFileChange:
			fileChanged = true
		}
	}
	if started == nil || completed == nil {
		t.Fatalf("expected started and completed tool_call, got %+v / %+v", started, completed)
	}
	if started.Summary != completed.Summary || started.Tool != completed.Tool {
		t.Fatalf("completed tool_call identity mismatch: started=%+v completed=%+v", started, completed)
	}
	if completed.Args == nil || completed.Args["file_path"] != "/tmp/a.txt" {
		t.Fatalf("completed tool_call missing args: %+v", completed)
	}
	if !fileChanged {
		t.Fatal("expected file_change event")
	}
}

// TestAttachPostToolUseFailure covers #261: Claude Code fires
// PostToolUseFailure (not PostToolUse) when a tool fails. A failed Bash
// command must resolve the running row with exit 1, and a failed non-Bash
// tool must emit tool_call(failed) with the same identity so the client
// merges in place instead of keeping the spinner.
func TestAttachPostToolUseFailure(t *testing.T) {
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
	post := func(path, body string) {
		t.Helper()
		resp := authRequest(t, http.MethodPost, ts.URL+path, cfg.LocalToken, strings.NewReader(body))
		resp.Body.Close()
	}

	// Failed Bash command: started row resolves with exit code 1.
	post("/hooks/claude/session-start", `{"hook_event_name":"SessionStart","session_id":"claude-sess-4","cwd":"/tmp/proj"}`)
	post("/hooks/claude/pre-tool-use", `{"hook_event_name":"PreToolUse","session_id":"claude-sess-4","tool_use_id":"tu4","tool_use":{"name":"Bash","input":{"command":"false"}}}`)
	post("/hooks/claude/post-tool-use-failure", `{"hook_event_name":"PostToolUseFailure","session_id":"claude-sess-4","tool_use_id":"tu4","tool_use":{"name":"Bash","input":{"command":"false"}},"error":"Command failed: exit status 1"}`)

	sess := srv.getSession("claude-sess-4")
	var commands []protocol.CommandPayload
	for _, hev := range sess.snapshot() {
		if hev.Type != protocol.EventCommand {
			continue
		}
		var p protocol.CommandPayload
		if err := hev.DecodePayload(&p); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, p)
	}
	if len(commands) != 2 {
		t.Fatalf("expected started + failed command events, got %d (%+v)", len(commands), commands)
	}
	if commands[0].Command != "false" || commands[0].ExitCode != nil {
		t.Fatalf("expected running command, got %+v", commands[0])
	}
	if commands[1].Command != "false" || commands[1].ExitCode == nil || *commands[1].ExitCode != 1 {
		t.Fatalf("expected failed command with exit 1, got %+v", commands[1])
	}
	if !strings.Contains(commands[1].Output, "exit status 1") {
		t.Fatalf("expected error output, got %q", commands[1].Output)
	}

	// Failed non-Bash tool: tool_call(failed) keeps the started identity.
	post("/hooks/claude/session-start", `{"hook_event_name":"SessionStart","session_id":"claude-sess-5","cwd":"/tmp/proj"}`)
	post("/hooks/claude/pre-tool-use", `{"hook_event_name":"PreToolUse","session_id":"claude-sess-5","tool_use_id":"tu5","tool_use":{"name":"Write","input":{"file_path":"/tmp/b.txt"}}}`)
	post("/hooks/claude/post-tool-use-failure", `{"hook_event_name":"PostToolUseFailure","session_id":"claude-sess-5","tool_use_id":"tu5","tool_use":{"name":"Write","input":{"file_path":"/tmp/b.txt"}},"error":"permission denied"}`)

	sess = srv.getSession("claude-sess-5")
	var started, failed *protocol.ToolCallPayload
	var notifyErr bool
	for _, hev := range sess.snapshot() {
		switch hev.Type {
		case protocol.EventToolCall:
			var p protocol.ToolCallPayload
			if err := hev.DecodePayload(&p); err != nil {
				t.Fatal(err)
			}
			switch p.Status {
			case "started":
				started = &p
			case "failed":
				failed = &p
			}
		case protocol.EventNotify:
			var p protocol.NotifyPayload
			if err := hev.DecodePayload(&p); err == nil && p.Level == "error" {
				notifyErr = true
			}
		}
	}
	if started == nil || failed == nil {
		t.Fatalf("expected started and failed tool_call, got %+v / %+v", started, failed)
	}
	if started.Summary != failed.Summary || started.Tool != failed.Tool {
		t.Fatalf("failed tool_call identity mismatch: started=%+v failed=%+v", started, failed)
	}
	if failed.Args == nil || failed.Args["file_path"] != "/tmp/b.txt" {
		t.Fatalf("failed tool_call missing args: %+v", failed)
	}
	if !notifyErr {
		t.Fatal("expected error notify")
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

// TestAttachRevivesRestoredSession covers #170: after a daemon restart a live
// claude session comes back as a read-only restoredAdapter; the agent's next
// hook must swap it for a live attachAdapter in place.
func TestAttachRevivesRestoredSession(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the pre-restart state: a live attach session on disk.
	ps := &PersistedSession{
		ID: "claude-restored-1", Name: "proj", CLI: "claude (attach)",
		Cwd: "/tmp/proj", Status: "restored", CreatedAt: time.Now(),
	}
	if err := persistSessionMeta(dir, ps); err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := New(cfg, keys, dir, logger, nil)

	sess := srv.getSession("claude-restored-1")
	if sess == nil {
		t.Fatal("session should be restored")
	}
	if _, ok := sess.getAdapter().(*restoredAdapter); !ok {
		t.Fatalf("expected restoredAdapter, got %T", sess.getAdapter())
	}
	if err := sess.getAdapter().SendPrompt("hi"); err == nil || !strings.Contains(err.Error(), "not attached") {
		t.Fatalf("restored session should reject prompts, got %v", err)
	}

	// The still-alive agent fires its next hook.
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	resp := authRequest(t, http.MethodPost, ts.URL+"/hooks/claude/session-start", cfg.LocalToken,
		strings.NewReader(`{"hook_event_name":"SessionStart","session_id":"claude-restored-1","cwd":"/tmp/proj"}`))
	resp.Body.Close()

	sess = srv.getSession("claude-restored-1")
	if _, ok := sess.getAdapter().(*attachAdapter); !ok {
		t.Fatalf("expected attachAdapter after hook, got %T", sess.getAdapter())
	}
	if sess.status != protocol.StatusRunning || sess.ended {
		t.Fatalf("expected running live session, got status=%s ended=%v", sess.status, sess.ended)
	}
	// SendPrompt now reaches the tmux injection path (no claude pane exists in
	// the test environment) instead of the restored placeholder error.
	if err := sess.getAdapter().SendPrompt("hi"); err == nil || strings.Contains(err.Error(), "not attached") {
		t.Fatalf("expected tmux injection error, got %v", err)
	}
}

// TestSweepIdleAttachSession covers #170: after `kill -9` on claude the
// SessionEnd hook never fires, so the sweeper must mark a hook-silent attach
// session ended — and a later hook must revive it (idle false positive).
func TestSweepIdleAttachSession(t *testing.T) {
	old := attachIdleTimeout
	attachIdleTimeout = time.Minute
	defer func() { attachIdleTimeout = old }()

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
	post := func(path, body string) {
		t.Helper()
		resp := authRequest(t, http.MethodPost, ts.URL+path, cfg.LocalToken, strings.NewReader(body))
		resp.Body.Close()
	}

	post("/hooks/claude/session-start", `{"hook_event_name":"SessionStart","session_id":"claude-idle-1","cwd":"/tmp/proj"}`)
	sess := srv.getSession("claude-idle-1")
	if sess == nil || sess.status != protocol.StatusRunning {
		t.Fatalf("expected running attach session, got %+v", sess)
	}

	// Simulate kill -9: hooks just stop coming.
	sess.mu.Lock()
	sess.lastSeen = time.Now().Add(-2 * time.Minute)
	sess.mu.Unlock()
	srv.sweepOnce()

	sess = srv.getSession("claude-idle-1")
	if sess == nil {
		t.Fatal("idle attach session should be marked ended, not removed")
	}
	if !sess.ended || sess.status != protocol.StatusDone {
		t.Fatalf("expected ended/done, got status=%s ended=%v", sess.status, sess.ended)
	}
	found := false
	for _, hev := range sess.snapshot() {
		if hev.Type != protocol.EventSessionEnd {
			continue
		}
		var p protocol.SessionEndPayload
		if err := hev.DecodePayload(&p); err == nil && p.Reason == "no_activity" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a session_end(no_activity) event in history")
	}

	// False-positive path: the agent was merely idle; its next hook revives
	// the session.
	post("/hooks/claude/user-prompt-submit", `{"hook_event_name":"UserPromptSubmit","session_id":"claude-idle-1","prompt":"继续"}`)
	sess = srv.getSession("claude-idle-1")
	if sess.ended || sess.status != protocol.StatusRunning {
		t.Fatalf("expected revived session, got status=%s ended=%v", sess.status, sess.ended)
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

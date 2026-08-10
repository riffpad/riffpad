package kimi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/packages/protocol"
)

func TestWriteSessionConfigRegistersAllHooks(t *testing.T) {
	k := New(adapter.CreateRequest{
		ID:        "s1",
		DataDir:   t.TempDir(),
		HookBase:  "http://127.0.0.1:8787",
		HookToken: "tok",
	})
	if err := k.writeSessionHome(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(k.configPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if got := strings.Count(body, "[[hooks]]"); got != len(kimiHookSpecs()) {
		t.Fatalf("expected %d hooks, got %d\n%s", len(kimiHookSpecs()), got, body)
	}
	for _, spec := range kimiHookSpecs() {
		if !strings.Contains(body, "event = "+strconvQuote(spec.event)) {
			t.Fatalf("missing hook event %s\n%s", spec.event, body)
		}
		if !strings.Contains(body, "/hooks/kimi/"+spec.route+"?session=s1") {
			t.Fatalf("hook %s missing session-routed URL\n%s", spec.event, body)
		}
	}
	if !strings.Contains(body, "token=tok") {
		t.Fatalf("hook commands missing token\n%s", body)
	}
}

func TestWriteSessionConfigPreservesUserConfig(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".kimi-code")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"),
		[]byte("default_model = \"kimi-code/k3\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	k := New(adapter.CreateRequest{ID: "s1", DataDir: t.TempDir(), HookBase: "http://127.0.0.1:8787"})
	if err := k.writeSessionHome(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(k.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "default_model = \"kimi-code/k3\"") {
		t.Fatalf("user config not preserved:\n%s", data)
	}
}

func strconvQuote(s string) string {
	return "\"" + s + "\""
}

func TestKimiTurnResultResetsRunning(t *testing.T) {
	k := New(adapter.CreateRequest{ID: "s1"})
	// SendPrompt flips turnActive on; the turn result must flip it back and
	// emit waiting_input so the client stops the running indicator (#255).
	k.mu.Lock()
	k.turnActive = true
	k.mu.Unlock()

	k.handleLine([]byte(`{"jsonrpc":"2.0","id":3,"result":{"sessionId":"s1","stopReason":"end_turn"}}`))
	ev := <-k.Events()
	if ev.Type != protocol.EventAgentStatus {
		t.Fatalf("expected agent_status, got %s", ev.Type)
	}
	var st protocol.AgentStatusPayload
	if err := ev.DecodePayload(&st); err != nil {
		t.Fatal(err)
	}
	if st.Status != protocol.StatusWaitingInput {
		t.Fatalf("expected waiting_input, got %q", st.Status)
	}
	k.mu.Lock()
	active := k.turnActive
	k.mu.Unlock()
	if active {
		t.Fatal("turnActive not reset after turn result")
	}
}

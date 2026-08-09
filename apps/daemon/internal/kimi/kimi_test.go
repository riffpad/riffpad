package kimi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
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

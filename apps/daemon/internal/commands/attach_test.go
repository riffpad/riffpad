package commands

// attach/detach hook-merging tests (issue #168), moved from cmd/riffpad
// during the #282 split. Attach is deprecated (#214): the entry point must
// refuse without touching settings; detach keeps working.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

// setupAttachEnv points HOME at a temp dir and stubs the local token so
// attachCmd writes to a throwaway ~/.claude/settings.json.
func setupAttachEnv(t *testing.T) (home string, base string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	cliutil.SetToken("")
	t.Cleanup(func() { cliutil.SetToken("") })
	ready := &atomic.Bool{}
	ready.Store(true)
	return home, fakeDaemon(t, ready).URL
}

func userHookEntry(command string) map[string]any {
	return map[string]any{
		"matcher": "Bash",
		"hooks":   []any{map[string]any{"type": "command", "command": command}},
	}
}

func riffpadHookEntry(path string) map[string]any {
	return map[string]any{
		"matcher": "",
		"hooks": []any{map[string]any{
			"type": "http",
			"url":  "http://127.0.0.1:8787/hooks/claude/" + path,
		}},
	}
}

func writeClaudeSettings(t *testing.T, home string, settings map[string]any) {
	t.Helper()
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readClaudeSettings(t *testing.T, home string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	settings := map[string]any{}
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatal(err)
	}
	return settings
}

// countHooks classifies every entry under settings["hooks"] as riffpad-owned
// (URL path under /hooks/claude/) or user-owned.
func countHooks(t *testing.T, settings map[string]any) (riffpad, user int) {
	t.Helper()
	hooks, _ := settings["hooks"].(map[string]any)
	for _, list := range hooks {
		entries, _ := list.([]any)
		for _, e := range entries {
			entry, _ := e.(map[string]any)
			nested, _ := entry["hooks"].([]any)
			isRiffpad := false
			for _, h := range nested {
				hm, _ := h.(map[string]any)
				if raw, _ := hm["url"].(string); strings.Contains(raw, "/hooks/claude/") {
					isRiffpad = true
				}
			}
			if isRiffpad {
				riffpad++
			} else {
				user++
			}
		}
	}
	return riffpad, user
}

func TestAttachPreservesUserHooks(t *testing.T) {
	home, base := setupAttachEnv(t)
	writeClaudeSettings(t, home, map[string]any{
		"model": "opus",
		"hooks": map[string]any{
			"PreToolUse":   []any{userHookEntry("my-formatter")},
			"Notification": []any{userHookEntry("my-notifier")},
		},
	})

	if err := AttachCmd(base); err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("expected deprecated error, got %v", err)
	}
	settings := readClaudeSettings(t, home)
	if settings["model"] != "opus" {
		t.Fatalf("unrelated setting lost: %+v", settings)
	}
	riffpad, user := countHooks(t, settings)
	if riffpad != 0 || user != 2 {
		t.Fatalf("disabled attach must not touch settings: riffpad=%d user=%d", riffpad, user)
	}
}

func TestAttachCmdDeprecated(t *testing.T) {
	err := AttachCmd("http://127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("expected deprecated error, got %v", err)
	}
}

func TestDetachRemovesOnlyRiffpadHooks(t *testing.T) {
	home, _ := setupAttachEnv(t)
	legacyHooks := map[string]any{
		"MessageDisplay":    []any{riffpadHookEntry("message-display")},
		"Notification":      []any{riffpadHookEntry("notification")},
		"PermissionRequest": []any{riffpadHookEntry("permission")},
		"PostToolUse":       []any{riffpadHookEntry("post-tool-use")},
		"PreToolUse":        []any{riffpadHookEntry("pre-tool-use")},
		"SessionEnd":        []any{riffpadHookEntry("session-end")},
		"SessionStart":      []any{riffpadHookEntry("session-start")},
		"UserPromptSubmit":  []any{riffpadHookEntry("user-prompt-submit")},
	}
	writeClaudeSettings(t, home, map[string]any{
		"model": "opus",
		"hooks": map[string]any{
			"PreToolUse":   []any{userHookEntry("my-formatter"), riffpadHookEntry("pre-tool-use")},
			"Notification": []any{userHookEntry("my-notifier")},
			"PostToolUse":  legacyHooks["PostToolUse"],
			"SessionStart": legacyHooks["SessionStart"],
		},
	})

	if err := DetachCmd(); err != nil {
		t.Fatal(err)
	}
	settings := readClaudeSettings(t, home)
	if settings["model"] != "opus" {
		t.Fatalf("unrelated setting lost: %+v", settings)
	}
	riffpad, user := countHooks(t, settings)
	if riffpad != 0 {
		t.Fatalf("expected no riffpad entries after detach, got %d", riffpad)
	}
	if user != 2 {
		t.Fatalf("expected both user hook entries kept, got %d", user)
	}
}

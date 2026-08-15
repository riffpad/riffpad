package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

// AttachCmd injects Claude Code hooks pointing at the local daemon, so a
// normal interactive `claude` session is captured and approvals can be made
// from the web UI / mobile. Existing user hooks are preserved: only riffpad's
// own entries are replaced, making repeated attaches idempotent.
func AttachCmd(base string) error {
	// Attach mode is deprecated: it permanently edits the user's global
	// Claude settings, which breaks normal claude startup when the daemon is
	// off or unpaired. The implementation is kept below for reference; only
	// the entry point is disabled (#214).
	return fmt.Errorf("%s", t.T("attach_deprecated"))
	if !cliutil.Reachable(base) {
		return fmt.Errorf("%s", t.T("daemon_start_hint", base))
	}
	port := defaultDaemonPort(base)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		return err
	}
	settings := map[string]any{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	}
	backup := settingsPath + ".riffpad.bak"
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		raw, _ := json.MarshalIndent(settings, "", "  ")
		if err := os.WriteFile(backup, raw, 0o600); err != nil {
			return err
		}
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	httpHook := func(path string, timeout int) map[string]any {
		hookURL := baseURL + path
		if tok := cliutil.LocalToken(); tok != "" {
			// The hook process cannot set headers; the daemon also accepts the
			// local API token as a query parameter.
			hookURL += "?token=" + url.QueryEscape(tok)
		}
		return map[string]any{
			"matcher": "",
			"hooks": []any{
				map[string]any{"type": "http", "url": hookURL, "timeout": timeout},
			},
		}
	}
	// Merge, don't overwrite: drop any previously injected riffpad entries
	// (idempotent re-attach) while keeping the user's own hooks intact.
	if stripRiffpadHooks(settings) {
		fmt.Println(t.T("attach_keep_existing"))
	}
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = map[string]any{}
		settings["hooks"] = hooks
	}
	add := func(event string, entry map[string]any) {
		list, _ := hooks[event].([]any)
		hooks[event] = append(list, entry)
	}
	add("SessionStart", httpHook("/hooks/claude/session-start", 10))
	add("SessionEnd", httpHook("/hooks/claude/session-end", 10))
	add("UserPromptSubmit", httpHook("/hooks/claude/user-prompt-submit", 30))
	add("MessageDisplay", httpHook("/hooks/claude/message-display", 10))
	add("PreToolUse", httpHook("/hooks/claude/pre-tool-use", 10))
	add("PostToolUse", httpHook("/hooks/claude/post-tool-use", 10))
	add("PermissionRequest", httpHook("/hooks/claude/permission", 600))
	add("Notification", httpHook("/hooks/claude/notification", 10))
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, raw, 0o600); err != nil {
		return err
	}
	fmt.Println(t.T("attach_injected"))
	fmt.Println(t.T("attach_next"))
	fmt.Println(t.T("attach_verify"))
	return nil
}

// riffpadHookPathPrefix namespaces every hook URL the daemon injects, so
// attach/detach can tell riffpad's own entries apart from the user's — query
// parameters (e.g. the ?token= added for local auth) are ignored by matching
// on the parsed URL path only.
const riffpadHookPathPrefix = "/hooks/claude/"

func isRiffpadHook(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && strings.HasPrefix(u.Path, riffpadHookPathPrefix)
}

// stripRiffpadHooks removes riffpad-injected hook entries from
// settings["hooks"] in place, keeping everything the user added. It reports
// whether any user-defined (non-riffpad) hooks remain.
func stripRiffpadHooks(settings map[string]any) bool {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return false
	}
	userHooks := false
	for event, list := range hooks {
		entries, ok := list.([]any)
		if !ok {
			userHooks = true
			continue
		}
		kept := make([]any, 0, len(entries))
		for _, e := range entries {
			entry, ok := e.(map[string]any)
			if !ok {
				kept = append(kept, e)
				continue
			}
			nested, ok := entry["hooks"].([]any)
			if !ok {
				kept = append(kept, e)
				continue
			}
			keptNested := make([]any, 0, len(nested))
			for _, h := range nested {
				if hm, ok := h.(map[string]any); ok {
					if raw, _ := hm["url"].(string); raw != "" && isRiffpadHook(raw) {
						continue
					}
				}
				keptNested = append(keptNested, h)
			}
			if len(keptNested) == 0 {
				continue
			}
			entry["hooks"] = keptNested
			kept = append(kept, entry)
		}
		if len(kept) == 0 {
			delete(hooks, event)
			continue
		}
		hooks[event] = kept
		userHooks = true
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	}
	return userHooks
}

// DetachCmd removes only the riffpad-injected hook entries, leaving any user
// configuration (including hooks added after attach) untouched. The
// settings.json.riffpad.bak snapshot from the first attach is no longer used
// for restoration; it is kept on disk for manual reference.
func DetachCmd() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("no settings found (nothing to detach)")
	}
	settings := map[string]any{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parse %s: %w", settingsPath, err)
	}
	stripRiffpadHooks(settings)
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, raw, 0o600); err != nil {
		return err
	}
	fmt.Println(t.T("detach_restored"))
	return nil
}

func defaultDaemonPort(base string) int {
	if i := strings.LastIndex(base, ":"); i >= 0 {
		if p, err := strconv.Atoi(base[i+1:]); err == nil {
			return p
		}
	}
	return 8787
}

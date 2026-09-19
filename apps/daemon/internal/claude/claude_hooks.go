package claude

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
)

type hookSpec struct {
	path    string // daemon route suffix (kebab-case)
	timeout int    // seconds
}

// hookSpecs returns the Claude hooks to register, keyed by the event name
// Claude expects in settings.json. The interactive TUI needs the full set
// (structured events + permissions flow through hooks); headless
// stream-json mode only needs Notification.
func (c *Claude) hookSpecs() map[string]hookSpec {
	if !c.interactive {
		return map[string]hookSpec{"Notification": {path: "notification", timeout: 10}}
	}
	return map[string]hookSpec{
		"SessionStart":       {path: "session-start", timeout: 10},
		"SessionEnd":         {path: "session-end", timeout: 10},
		"UserPromptSubmit":   {path: "user-prompt-submit", timeout: 30},
		"MessageDisplay":     {path: "message-display", timeout: 10},
		"PreToolUse":         {path: "pre-tool-use", timeout: 10},
		"PostToolUse":        {path: "post-tool-use", timeout: 10},
		"PostToolUseFailure": {path: "post-tool-use-failure", timeout: 10},
		"PermissionRequest":  {path: "permission", timeout: 600},
		"Notification":       {path: "notification", timeout: 10},
		// Stop fires when the previous turn finishes, so the daemon can flip
		// the session back to waiting_input immediately instead of waiting
		// for idle_prompt's ~60s heuristic (#257).
		"Stop": {path: "stop", timeout: 10},
	}
}

func (c *Claude) writeSettings() error {
	dir := filepath.Join(c.dataDir, "sessions", c.id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	c.settingsPath = filepath.Join(dir, "settings.json")
	settings := map[string]any{"hooks": map[string]any{}}
	hooks := map[string]any{}
	if c.hookBase != "" {
		for event, spec := range c.hookSpecs() {
			hookURL := c.hookBase + "/hooks/claude/" + spec.path + "?session=" + c.id
			if c.hookToken != "" {
				hookURL += "&token=" + url.QueryEscape(c.hookToken)
			}
			hooks[event] = []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{
							"type":    "http",
							"url":     hookURL,
							"timeout": spec.timeout,
						},
					},
				},
			}
		}
	}
	if len(hooks) > 0 {
		settings["hooks"] = hooks
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.settingsPath, data, 0o600)
}

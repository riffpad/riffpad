package kimi

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// kimiHookSpec describes one [[hooks]] entry in the per-session config.
type kimiHookSpec struct {
	event   string // Kimi hook event name
	route   string // daemon route suffix
	timeout int
}

func kimiHookSpecs() []kimiHookSpec {
	return []kimiHookSpec{
		{event: "SessionStart", route: "session-start", timeout: 10},
		{event: "SessionEnd", route: "session-end", timeout: 10},
		{event: "UserPromptSubmit", route: "user-prompt-submit", timeout: 30},
		{event: "PreToolUse", route: "pre-tool-use", timeout: 600},
		{event: "PostToolUse", route: "post-tool-use", timeout: 30},
		{event: "PostToolUseFailure", route: "post-tool-use-failure", timeout: 30},
		{event: "Stop", route: "stop", timeout: 30},
		{event: "Notification", route: "notification", timeout: 30},
	}
}

// writeSessionHome builds an isolated per-session kimi home:
//
//	<dataDir>/sessions/<id>/kimi-home/
//	  config.toml   = user's config + riffpad [[hooks]] (daemon-routed)
//	  credentials/  = symlink to the user's real home (auth preserved)
//	  sessions/…    = symlinked so resume and history still work
//
// kimi is launched with KIMI_CODE_HOME=<that dir>, so plain `kimi` runs never
// see riffpad hooks. The session id rides in each hook URL so the daemon
// routes events to the right session.
func (k *Kimi) writeSessionHome() error {
	dir := filepath.Join(k.dataDir, "sessions", k.id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	home := filepath.Join(dir, "kimi-home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		return err
	}
	k.configPath = filepath.Join(home, "config.toml")

	var body strings.Builder
	if realHome, userCfg, err := findUserKimiHome(); err == nil && userCfg != "" {
		body.WriteString(userCfg)
		if !strings.HasSuffix(body.String(), "\n") {
			body.WriteString("\n")
		}
		// Symlink everything else (credentials, sessions, blobs, store, …) so
		// login state and session history behave exactly like a normal kimi.
		entries, _ := os.ReadDir(realHome)
		for _, e := range entries {
			if e.Name() == "config.toml" || e.Name() == "config.json" {
				continue
			}
			src := filepath.Join(realHome, e.Name())
			dst := filepath.Join(home, e.Name())
			if _, err := os.Lstat(dst); os.IsNotExist(err) {
				_ = os.Symlink(src, dst)
			}
		}
	}
	body.WriteString("\n# riffpad hooks (per-session, generated)\n")
	for _, spec := range kimiHookSpecs() {
		hookURL := k.hookBase + "/hooks/kimi/" + spec.route + "?session=" + k.id
		if k.hookToken != "" {
			hookURL += "&token=" + url.QueryEscape(k.hookToken)
		}
		fmt.Fprintf(&body, "[[hooks]]\n")
		fmt.Fprintf(&body, "event = %q\n", spec.event)
		fmt.Fprintf(&body, "matcher = \"\"\n")
		fmt.Fprintf(&body, "command = %q\n", "curl -sS --max-time 590 -X POST '"+hookURL+"' -H 'Content-Type: application/json' --data-binary @-")
		fmt.Fprintf(&body, "timeout = %d\n\n", spec.timeout)
	}
	return os.WriteFile(k.configPath, []byte(body.String()), 0o600)
}

// findUserKimiHome returns the user's real kimi home and config.toml text.
func findUserKimiHome() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	for _, dir := range []string{
		filepath.Join(home, ".kimi-code"),
		filepath.Join(home, ".kimi"),
	} {
		raw, err := os.ReadFile(filepath.Join(dir, "config.toml"))
		if err == nil {
			return dir, string(raw), nil
		}
		if !os.IsNotExist(err) {
			return "", "", err
		}
	}
	return "", "", os.ErrNotExist
}

// Package console bridges the user's terminal to daemon-hosted interactive
// sessions: the PTY WebSocket console (claude/kimi foreground mode) and the
// Codex TUI resume flow.
package console

import (
	"github.com/riffpad/riffpad/apps/daemon/internal/i18n"
)

// t is the active language bundle, kept in sync with the CLI's via SetBundle.
var t = i18n.New(i18n.DefaultLang)

// SetBundle switches the console package's translations. Call after parsing
// --lang, before dispatching a command.
func SetBundle(b *i18n.Bundle) {
	if b != nil {
		t = b
	}
}

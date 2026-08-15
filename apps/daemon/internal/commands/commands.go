// Package commands implements the riffpad CLI subcommands. main stays a thin
// entry point: it parses --lang, dispatches, and delegates here.
package commands

import (
	"github.com/riffpad/riffpad/apps/daemon/internal/i18n"
)

// t is the active language bundle, kept in sync with the CLI's via SetBundle.
var t = i18n.New(i18n.DefaultLang)

// SetBundle switches the command package's translations. Call after parsing
// --lang, before dispatching a command.
func SetBundle(b *i18n.Bundle) {
	if b != nil {
		t = b
	}
}

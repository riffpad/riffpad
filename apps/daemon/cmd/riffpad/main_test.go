package main

import (
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

// withCliToken stubs the CLI's local token for the duration of a test.
// All other tests from this file moved to internal/commands and
// internal/cliutil during the #282 split.
func withCliToken(t *testing.T, token string) {
	t.Helper()
	cliutil.SetToken(token)
	t.Cleanup(func() { cliutil.SetToken("") })
}

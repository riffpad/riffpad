package main

import (
	"strings"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
)

// withCliToken stubs the CLI's local token for the duration of a test.
func withCliToken(t *testing.T, token string) {
	t.Helper()
	cliutil.SetToken(token)
	t.Cleanup(func() { cliutil.SetToken("") })
}

// extractLangFlag tests: the global --lang/-lang extraction must work in any
// argument position (it runs before subcommand dispatch).
func TestExtractLangFlag(t *testing.T) {
	cases := []struct {
		in   []string
		lang string
		rest []string
	}{
		{[]string{"run", "--lang", "zh"}, "zh", []string{"run"}},
		{[]string{"run", "--lang=zh"}, "zh", []string{"run"}},
		{[]string{"-lang=zh", "run"}, "zh", []string{"run"}},
		{[]string{"run"}, "", []string{"run"}},
		{[]string{"--lang"}, "", []string{}},
	}
	for _, c := range cases {
		lang, rest := extractLangFlag(c.in)
		if lang != c.lang || strings.Join(rest, ",") != strings.Join(c.rest, ",") {
			t.Errorf("extractLangFlag(%v) = (%q, %v), want (%q, %v)", c.in, lang, rest, c.lang, c.rest)
		}
	}
}

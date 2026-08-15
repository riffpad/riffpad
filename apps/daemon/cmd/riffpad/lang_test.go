package main

// extractLangFlag tests: the global --lang/-lang extraction must work in any
// argument position (it runs before subcommand dispatch).

import (
	"strings"
	"testing"
)

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

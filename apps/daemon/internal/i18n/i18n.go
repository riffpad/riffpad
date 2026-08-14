// Package i18n provides lightweight localization for the riffpad CLI.
//
// Language selection order:
//  1. --lang flag (highest priority)
//  2. default: en
//
// The CLI no longer infers language from LC_ALL/LC_MESSAGES/LANG; English is
// the default, and Chinese is only used when explicitly requested with
// `--lang zh`. Unsupported languages fall back to English instead of erroring.
package i18n

import (
	"fmt"
	"strings"
)

// DefaultLang is the fallback language.
const DefaultLang = "en"

// Bundle translates message keys for a resolved language.
type Bundle struct {
	lang string
}

// New returns a bundle for lang (already resolved via Detect).
func New(lang string) *Bundle {
	return &Bundle{lang: lang}
}

// Lang returns the bundle's language code.
func (b *Bundle) Lang() string {
	if b == nil {
		return DefaultLang
	}
	return b.lang
}

// T translates key, formatting with args.
func (b *Bundle) T(key string, args ...any) string {
	lang := DefaultLang
	if b != nil && b.lang != "" {
		lang = b.lang
	}
	table, ok := messages[lang]
	if !ok {
		table = messages[DefaultLang]
	}
	format, ok := table[key]
	if !ok {
		format = messages[DefaultLang][key]
	}
	if format == "" {
		return key
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// Detect resolves the language code from an explicit --lang flag value.
// Returns a supported code ("zh" or "en"); unsupported values fall back to
// DefaultLang ("en"). When langFlag is empty, English is used.
func Detect(langFlag string) string {
	if langFlag != "" {
		if l := resolve(langFlag); l != "" {
			return l
		}
	}
	return DefaultLang
}

// resolve normalizes a locale string ("zh_CN.UTF-8", "en_US", "fr", "C") to a
// supported language code, or "" if unsupported.
func resolve(raw string) string {
	code := strings.ToLower(strings.TrimSpace(raw))
	if i := strings.IndexAny(code, "._@"); i >= 0 {
		code = code[:i]
	}
	switch code {
	case "zh", "zh-hans", "zh-hant", "zh-cn", "zh-tw", "zh-sg", "zh-hk", "zh-mo":
		return "zh"
	case "en", "en-us", "en-gb":
		return "en"
	default:
		return ""
	}
}

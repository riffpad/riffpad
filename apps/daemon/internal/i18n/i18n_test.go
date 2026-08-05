package i18n

import "testing"

func TestDetectFlagOverridesEnv(t *testing.T) {
	t.Setenv("LANG", "zh_CN.UTF-8")
	if got := Detect("en"); got != "en" {
		t.Fatalf("flag should override env, got %q", got)
	}
	if got := Detect("fr"); got != "en" {
		t.Fatalf("unsupported flag should fall back to en, got %q", got)
	}
}

func TestDetectEnvPrecedence(t *testing.T) {
	t.Setenv("LANG", "zh_CN.UTF-8")
	t.Setenv("LC_MESSAGES", "en_US.UTF-8")
	if got := Detect(""); got != "en" {
		t.Fatalf("LC_MESSAGES should beat LANG, got %q", got)
	}
	t.Setenv("LC_ALL", "zh_TW.UTF-8")
	if got := Detect(""); got != "zh" {
		t.Fatalf("LC_ALL should beat LC_MESSAGES, got %q", got)
	}
}

func TestDetectLocaleNormalization(t *testing.T) {
	cases := map[string]string{
		"":            "en",
		"zh":          "zh",
		"zh_CN.UTF-8": "zh",
		"zh_TW":       "zh",
		"en_US.UTF-8": "en",
		"fr_FR.UTF-8": "en",
		"de":          "en",
		"C":           "en",
		"POSIX":       "en",
	}
	for locale, want := range cases {
		if got := Detect(locale); got != want {
			t.Errorf("Detect(%q) = %q, want %q", locale, got, want)
		}
	}
}

func TestTranslate(t *testing.T) {
	en := New("en")
	zh := New("zh")
	if got := en.T("daemon_started", "http://x"); got != "daemon started at http://x" {
		t.Errorf("en daemon_started = %q", got)
	}
	if got := zh.T("daemon_started", "http://x"); got != "daemon 已在 http://x 启动" {
		t.Errorf("zh daemon_started = %q", got)
	}
	// Missing key returns the key itself (never empty/garbled).
	if got := zh.T("no_such_key"); got != "no_such_key" {
		t.Errorf("missing key = %q", got)
	}
	// Fallback language table is used for missing translations.
	if got := New("zz").T("pair_code", "abc"); got != "Pairing code: abc\nEnter it in the phone/browser (or scan the QR)" {
		t.Errorf("fallback pair_code = %q", got)
	}
}

package i18n

import "testing"

func TestDetectFlag(t *testing.T) {
	if got := Detect("en"); got != "en" {
		t.Fatalf("flag en should return en, got %q", got)
	}
	if got := Detect("zh"); got != "zh" {
		t.Fatalf("flag zh should return zh, got %q", got)
	}
	if got := Detect("zh_CN.UTF-8"); got != "zh" {
		t.Fatalf("flag zh_CN.UTF-8 should normalize to zh, got %q", got)
	}
	if got := Detect("fr"); got != "en" {
		t.Fatalf("unsupported flag should fall back to en, got %q", got)
	}
}

func TestDetectDefault(t *testing.T) {
	if got := Detect(""); got != "en" {
		t.Fatalf("empty flag should default to en, got %q", got)
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

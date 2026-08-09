package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	data := "# comment\nSMTP_HOST=mail.spacemail.com\nSMTP_PASS='p@ss*w*d'\nQUOTED=\"x\"\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SMTP_PASS", "existing")
	if err := loadEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("SMTP_HOST"); got != "mail.spacemail.com" {
		t.Fatalf("SMTP_HOST = %q", got)
	}
	// Existing variables must not be overridden.
	if got := os.Getenv("SMTP_PASS"); got != "existing" {
		t.Fatalf("SMTP_PASS overridden to %q", got)
	}
	if got := os.Getenv("QUOTED"); got != "x" {
		t.Fatalf("QUOTED = %q", got)
	}
	// Missing file is not an error.
	if err := loadEnvFile(filepath.Join(t.TempDir(), "nope")); err != nil {
		t.Fatal(err)
	}
}

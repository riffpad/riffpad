package config

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGeneratesAndPersistsLocalToken(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LocalToken == "" {
		t.Fatal("expected a local token on first load")
	}
	raw, err := hex.DecodeString(cfg.LocalToken)
	if err != nil || len(raw) != 32 {
		t.Fatalf("token should be 32 bytes hex, got %q", cfg.LocalToken)
	}
	// Persisted: a second load returns the same token (CLI/daemon agreement).
	cfg2, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.LocalToken != cfg.LocalToken {
		t.Fatal("local token must be stable across loads")
	}
	info, err := os.Stat(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config.json should be 0600, got %o", info.Mode().Perm())
	}
}

func TestLoadBackfillsLocalToken(t *testing.T) {
	dir := t.TempDir()
	// Pre-existing config from before local tokens existed.
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"port": 8787}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LocalToken == "" {
		t.Fatal("expected token backfill for legacy config")
	}
	cfg2, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.LocalToken != cfg.LocalToken {
		t.Fatal("backfilled token must persist")
	}
}

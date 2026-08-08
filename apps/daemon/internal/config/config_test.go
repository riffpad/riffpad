package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// captureHeal installs an OnHeal hook for the duration of a test.
func captureHeal(t *testing.T) *[]Heal {
	t.Helper()
	heals := &[]Heal{}
	prev := OnHeal
	OnHeal = func(h Heal) { *heals = append(*heals, h) }
	t.Cleanup(func() { OnHeal = prev })
	return heals
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Initial content must survive until a full replacement is in place.
	if err := os.WriteFile(path, []byte(`{"port": 1111}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte(`{"port": 2222}`), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"port": 2222}` {
		t.Fatalf("content %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm %o", info.Mode().Perm())
	}
	// No temp files left behind.
	leftovers, err := filepath.Glob(filepath.Join(dir, ".config.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("temp files left behind: %v", leftovers)
	}

	// A failed write (target dir removed) must return an error and leave no
	// partial target file behind.
	gone := filepath.Join(dir, "gone", "config.json")
	if err := WriteFileAtomic(gone, []byte(`{}`), 0o600); err == nil {
		t.Fatal("expected error writing into a missing directory")
	}
	if _, err := os.Stat(gone); !os.IsNotExist(err) {
		t.Fatalf("partial target file should not exist, stat err=%v", err)
	}
}

func TestLoadHealsCorruptedConfig(t *testing.T) {
	dir := t.TempDir()
	heals := captureHeal(t)
	path := filepath.Join(dir, "config.json")
	// Simulate a truncated write (power loss / kill -9 mid-write).
	if err := os.WriteFile(path, []byte(`{"port": 8787, "relayToken": "abc`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("corrupted config must self-heal, got %v", err)
	}
	if cfg.Port != defaultPort || cfg.LocalToken == "" {
		t.Fatalf("expected rebuilt defaults, got %+v", cfg)
	}
	// The corrupted content is preserved at config.json.bak.
	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal("expected config.json.bak backup")
	}
	if string(bak) != `{"port": 8787, "relayToken": "abc` {
		t.Fatalf("backup content %q", bak)
	}
	if len(*heals) != 1 || (*heals)[0].Kind != "config" || (*heals)[0].Backup != path+".bak" {
		t.Fatalf("expected one config heal notification, got %+v", *heals)
	}
	// The rebuilt config is valid and persists.
	cfg2, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.LocalToken != cfg.LocalToken {
		t.Fatal("rebuilt config must reload cleanly")
	}
}

func TestLoadOrCreateKeysHealsCorrupted(t *testing.T) {
	dir := t.TempDir()
	heals := captureHeal(t)
	path := filepath.Join(dir, "keys.json")
	if err := os.WriteFile(path, []byte(`{"x25519Private": "trunc`), 0o600); err != nil {
		t.Fatal(err)
	}
	k, err := LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatalf("corrupted keys must self-heal, got %v", err)
	}
	if k.X25519Private == "" || k.P256Private == "" || k.SessionEncKey == "" {
		t.Fatal("expected freshly generated keys")
	}
	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal("expected keys.json.bak backup")
	}
	if string(bak) != `{"x25519Private": "trunc` {
		t.Fatalf("backup content %q", bak)
	}
	if len(*heals) != 1 || (*heals)[0].Kind != "keys" {
		t.Fatalf("expected one keys heal notification, got %+v", *heals)
	}
	// Reloading returns the same regenerated keys (stable identity).
	k2, err := LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	if k2.X25519Public != k.X25519Public {
		t.Fatal("regenerated keys must persist across loads")
	}
}

// TestUpdateMergesConcurrentWriters simulates `riffpad login` and the running
// daemon writing different fields of config.json at the same time: with the
// lock + re-read merge, neither write is lost (#172).
func TestUpdateMergesConcurrentWriters(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); err != nil {
		t.Fatal(err)
	}
	const writers = 8
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			if err := Update(dir, func(c *Config) { c.RelayToken = fmt.Sprintf("tok-%d", i) }); err != nil {
				t.Error(err)
			}
		}(i)
		go func(i int) {
			defer wg.Done()
			if err := Update(dir, func(c *Config) { c.HostSecret = fmt.Sprintf("sec-%d", i) }); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Both fields must survive: a last-write-wins loser would have clobbered
	// one of them with its stale empty value.
	if !strings.HasPrefix(cfg.RelayToken, "tok-") {
		t.Fatalf("relay token lost: %q", cfg.RelayToken)
	}
	if !strings.HasPrefix(cfg.HostSecret, "sec-") {
		t.Fatalf("host secret lost: %q", cfg.HostSecret)
	}
	if cfg.LocalToken == "" {
		t.Fatal("local token must survive concurrent updates")
	}
}

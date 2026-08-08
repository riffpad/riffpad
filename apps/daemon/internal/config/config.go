package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/riffpad/riffpad/packages/protocol"
)

// Heal reports a corrupted state file that was backed up and rebuilt with
// defaults (#172).
type Heal struct {
	Path   string // the corrupted file
	Backup string // where the corrupted content was moved (<path>.bak)
	Kind   string // "config", "keys", or "devices"
}

// OnHeal, when non-nil, is called after a corrupted file is backed up and
// rebuilt. The CLI/daemon set it to surface the warning to the user.
var OnHeal func(h Heal)

// BackupCorrupted moves a corrupted file to <path>.bak (replacing any older
// backup) and reports the heal via OnHeal.
func BackupCorrupted(path, kind string) (string, error) {
	backup := path + ".bak"
	_ = os.Remove(backup) // Windows rename fails when the target exists
	if err := os.Rename(path, backup); err != nil {
		return "", err
	}
	if OnHeal != nil {
		OnHeal(Heal{Path: path, Backup: backup, Kind: kind})
	}
	return backup, nil
}

// WriteFileAtomic writes data to path atomically: a temp file in the same
// directory is fully written and fsynced, then renamed over path, so a crash
// or kill -9 mid-write can never leave a truncated file behind (#172).
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename succeeded
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	// Best effort: flush the directory entry so the rename survives a crash.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

const defaultPort = 8787

// Config is the daemon configuration persisted to config.json.
type Config struct {
	Port          int    `json:"port"`
	LocalToken    string `json:"localToken,omitempty"` // auth token for the local HTTP/WS API
	RelayURL      string `json:"relayUrl,omitempty"`
	HostID        string `json:"hostId,omitempty"`
	HostToken     string `json:"hostToken,omitempty"`
	HostSecret    string `json:"hostSecret,omitempty"`
	RelayToken    string `json:"relayToken,omitempty"`
	RelayUser     string `json:"relayUser,omitempty"`
	RelayPassword string `json:"-"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{Port: defaultPort}
}

// DefaultDataDir returns ~/.config/riffpad.
func DefaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "riffpad"), nil
}

// Load reads config.json from dir, creating defaults when missing. A
// corrupted config is backed up to config.json.bak and rebuilt with defaults
// instead of failing startup (#172).
func Load(dir string) (*Config, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")
	if data, err := os.ReadFile(path); err == nil {
		cfg := Default()
		if err := json.Unmarshal(data, cfg); err != nil {
			if _, berr := BackupCorrupted(path, "config"); berr != nil {
				return nil, fmt.Errorf("parse %s: %w (backup failed: %v)", path, err, berr)
			}
			// Fall through to rebuild with defaults.
		} else {
			if cfg.Port <= 0 {
				cfg.Port = defaultPort
			}
			applyEnvOverrides(cfg)
			if cfg.LocalToken == "" {
				cfg.LocalToken = NewLocalToken()
				if err := Save(dir, cfg); err != nil {
					return nil, err
				}
			}
			return cfg, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	cfg := Default()
	cfg.LocalToken = NewLocalToken()
	applyEnvOverrides(cfg)
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := WriteFileAtomic(path, raw, 0o600); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("RIFFPAD_LOCAL_TOKEN"); v != "" {
		cfg.LocalToken = v
	}
	if v := os.Getenv("RIFFPAD_RELAY_URL"); v != "" {
		cfg.RelayURL = v
	}
	if v := os.Getenv("RIFFPAD_HOST_ID"); v != "" {
		cfg.HostID = v
	}
	if v := os.Getenv("RIFFPAD_HOST_TOKEN"); v != "" {
		cfg.HostToken = v
	}
	if v := os.Getenv("RIFFPAD_HOST_SECRET"); v != "" {
		cfg.HostSecret = v
	}
	if v := os.Getenv("RIFFPAD_RELAY_USER"); v != "" {
		cfg.RelayUser = v
	}
	if v := os.Getenv("RIFFPAD_RELAY_PASSWORD"); v != "" {
		cfg.RelayPassword = v
	}
}

// Save persists the configuration back to config.json.
func Save(dir string, cfg *Config) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(filepath.Join(dir, "config.json"), raw, 0o600)
}

// Update mutates config.json under an exclusive file lock, re-reading the
// current on-disk state first so concurrent writers (the riffpad CLI and the
// running daemon) merge their changes instead of last-write-wins clobbering
// each other (#172).
func Update(dir string, mutate func(*Config)) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(dir, "config.json.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := lockFile(lock); err != nil {
		return err
	}
	defer func() { _ = unlockFile(lock) }()
	cfg, err := Load(dir)
	if err != nil {
		return err
	}
	mutate(cfg)
	return Save(dir, cfg)
}

// Keys holds the daemon's long-lived identity key pairs.
type Keys struct {
	X25519Private string `json:"x25519Private"`
	X25519Public  string `json:"x25519Public"`
	P256Private   string `json:"p256Private"`
	P256Public    string `json:"p256Public"`
	SessionEncKey string `json:"sessionEncKey,omitempty"` // local AES-256 key for persisted session history
}

// LoadOrCreateKeys loads keys.json, generating identity keys on first run. A
// corrupted keys file is backed up to keys.json.bak and regenerated (#172) —
// note that regeneration invalidates every paired device, so callers should
// surface the OnHeal warning prominently.
func LoadOrCreateKeys(dir string) (*Keys, error) {
	path := filepath.Join(dir, "keys.json")
	if data, err := os.ReadFile(path); err == nil {
		k := &Keys{}
		if err := json.Unmarshal(data, k); err != nil {
			if _, berr := BackupCorrupted(path, "keys"); berr != nil {
				return nil, fmt.Errorf("parse %s: %w (backup failed: %v)", path, err, berr)
			}
			// Fall through to regenerate.
		} else {
			if k.SessionEncKey == "" {
				k.SessionEncKey = newSessionEncKey()
				if err := saveKeys(path, k); err != nil {
					return nil, err
				}
			}
			return k, nil
		}
	}

	x, err := protocol.GenerateKeyPair(protocol.CurveX25519)
	if err != nil {
		return nil, err
	}
	p, err := protocol.GenerateKeyPair(protocol.CurveP256)
	if err != nil {
		return nil, err
	}
	k := &Keys{
		X25519Private: protocol.EncodeKey(x.PrivateKey()),
		X25519Public:  protocol.EncodeKey(x.PublicKey),
		P256Private:   protocol.EncodeKey(p.PrivateKey()),
		P256Public:    protocol.EncodeKey(p.PublicKey),
		SessionEncKey: newSessionEncKey(),
	}
	if err := saveKeys(path, k); err != nil {
		return nil, err
	}
	return k, nil
}

// NewLocalToken returns a fresh random local API token (hex, 256-bit).
func NewLocalToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func newSessionEncKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func saveKeys(path string, k *Keys) error {
	data, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, data, 0o600)
}

// Identity returns the server identity key pair for the given curve.
func (k *Keys) Identity(curve protocol.Curve) (*protocol.KeyPair, error) {
	switch curve {
	case protocol.CurveX25519:
		priv, err := protocol.DecodeKey(k.X25519Private)
		if err != nil {
			return nil, err
		}
		pub, err := protocol.DecodeKey(k.X25519Public)
		if err != nil {
			return nil, err
		}
		return protocol.RestoreKeyPair(curve, pub, priv)
	case protocol.CurveP256:
		priv, err := protocol.DecodeKey(k.P256Private)
		if err != nil {
			return nil, err
		}
		pub, err := protocol.DecodeKey(k.P256Public)
		if err != nil {
			return nil, err
		}
		return protocol.RestoreKeyPair(curve, pub, priv)
	default:
		return nil, errors.New("unsupported curve")
	}
}

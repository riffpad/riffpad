package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/riffpad/riffpad/packages/protocol"
)

const defaultPort = 8787

// Config is the daemon configuration persisted to config.json.
type Config struct {
	Port      int    `json:"port"`
	RelayURL  string `json:"relayUrl,omitempty"`
	HostID    string `json:"hostId,omitempty"`
	HostToken string `json:"hostToken,omitempty"`
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

// Load reads config.json from dir, creating defaults when missing.
func Load(dir string) (*Config, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if err == nil {
		cfg := Default()
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
		if cfg.Port <= 0 {
			cfg.Port = defaultPort
		}
		applyEnvOverrides(cfg)
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	cfg := Default()
	applyEnvOverrides(cfg)
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("RIFFPAD_RELAY_URL"); v != "" {
		cfg.RelayURL = v
	}
	if v := os.Getenv("RIFFPAD_HOST_ID"); v != "" {
		cfg.HostID = v
	}
	if v := os.Getenv("RIFFPAD_HOST_TOKEN"); v != "" {
		cfg.HostToken = v
	}
}

// Keys holds the daemon's long-lived identity key pairs.
type Keys struct {
	X25519Private string `json:"x25519Private"`
	X25519Public  string `json:"x25519Public"`
	P256Private   string `json:"p256Private"`
	P256Public    string `json:"p256Public"`
}

// LoadOrCreateKeys loads keys.json, generating identity keys on first run.
func LoadOrCreateKeys(dir string) (*Keys, error) {
	path := filepath.Join(dir, "keys.json")
	if data, err := os.ReadFile(path); err == nil {
		k := &Keys{}
		if err := json.Unmarshal(data, k); err != nil {
			return nil, err
		}
		return k, nil
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
	}
	raw, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return nil, err
	}
	return k, nil
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

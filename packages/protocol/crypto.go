package protocol

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Curve identifies the ECDH curve used by a device/session.
type Curve string

const (
	CurveX25519 Curve = "x25519" // native mobile/daemon
	CurveP256   Curve = "p256"   // web client (Web Crypto)
)

// KeyPair is a raw ECDH key pair.
type KeyPair struct {
	Curve     Curve
	PublicKey []byte
	private   []byte
}

// GenerateKeyPair creates a fresh key pair for the given curve.
func GenerateKeyPair(curve Curve) (*KeyPair, error) {
	switch curve {
	case CurveX25519:
		k, err := ecdh.X25519().GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		return &KeyPair{Curve: curve, PublicKey: k.PublicKey().Bytes(), private: k.Bytes()}, nil
	case CurveP256:
		k, err := ecdh.P256().GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		return &KeyPair{Curve: curve, PublicKey: k.PublicKey().Bytes(), private: k.Bytes()}, nil
	default:
		return nil, fmt.Errorf("unsupported curve %q", curve)
	}
}

// RestoreKeyPair reconstructs a private key pair from its raw bytes.
func RestoreKeyPair(curve Curve, publicKey, privateKey []byte) (*KeyPair, error) {
	if err := validateCurve(curve); err != nil {
		return nil, err
	}
	return &KeyPair{Curve: curve, PublicKey: append([]byte(nil), publicKey...), private: append([]byte(nil), privateKey...)}, nil
}

func (k *KeyPair) PrivateKey() []byte {
	return append([]byte(nil), k.private...)
}

func (k *KeyPair) ecdhKey() (*ecdh.PrivateKey, error) {
	var c ecdh.Curve
	switch k.Curve {
	case CurveX25519:
		c = ecdh.X25519()
	case CurveP256:
		c = ecdh.P256()
	default:
		return nil, fmt.Errorf("unsupported curve %q", k.Curve)
	}
	return c.NewPrivateKey(k.private)
}

// ECDH computes the shared secret between priv and peerPub.
func ECDH(priv *KeyPair, peerPub []byte) ([]byte, error) {
	k, err := priv.ecdhKey()
	if err != nil {
		return nil, err
	}
	var c ecdh.Curve
	switch priv.Curve {
	case CurveX25519:
		c = ecdh.X25519()
	case CurveP256:
		c = ecdh.P256()
	default:
		return nil, fmt.Errorf("unsupported curve %q", priv.Curve)
	}
	pub, err := c.NewPublicKey(peerPub)
	if err != nil {
		return nil, fmt.Errorf("peer public key: %w", err)
	}
	return k.ECDH(pub)
}

func validateCurve(curve Curve) error {
	switch curve {
	case CurveX25519, CurveP256:
		return nil
	default:
		return fmt.Errorf("unsupported curve %q", curve)
	}
}

// NewDeviceSecret derives the device-level shared secret between two paired
// identities: ECDH(identityA, identityB).
func NewDeviceSecret(identity *KeyPair, peerPublicKey []byte) ([]byte, error) {
	return ECDH(identity, peerPublicKey)
}

// DeriveSessionKey derives the per-session AES-256 key from:
//   HKDF-SHA256(ikm = ephemeral ECDH secret, salt = device secret,
//               info = "riffpad/session-v1/" + sessionID)
//
// Binding the device secret as salt authenticates the ephemeral exchange;
// using fresh ephemeral keys per session provides forward secrecy.
func DeriveSessionKey(deviceSecret, ephemeralSecret []byte, sessionID string) (*[32]byte, error) {
	r := hkdf.New(sha256.New, ephemeralSecret, deviceSecret, []byte("riffpad/session-v1/"+sessionID))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return (*[32]byte)(key), nil
}

// EncodeKey base64-encodes a raw key for storage/wire use.
func EncodeKey(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeKey base64-decodes a raw key.
func DecodeKey(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

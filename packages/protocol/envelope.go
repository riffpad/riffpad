package protocol

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// Envelope is the encrypted container forwarded by the relay. The relay may
// read routing fields (sessionId) but never the ciphertext contents.
type Envelope struct {
	V          int    `json:"v"`
	Kind       string `json:"kind"` // event | control
	SessionID  string `json:"sessionId,omitempty"`
	Nonce      string `json:"nonce,omitempty"`
	Ciphertext string `json:"ciphertext,omitempty"`
}

// Hello is the unencrypted session handshake message sent by the daemon so a
// client can derive the session key. It only carries an ephemeral public key.
type Hello struct {
	V           int    `json:"v"`
	Kind        string `json:"kind"` // hello
	SessionID   string `json:"sessionId"`
	ServerEphPub string `json:"serverEphPub"`
}

func (h Hello) Marshal() ([]byte, error) {
	return json.Marshal(h)
}

// WrapEnvelope encrypts plaintext with the session key and returns an Envelope.
func WrapEnvelope(sessionID string, plaintext []byte, key *[32]byte) (Envelope, error) {
	nonce, ct, err := Encrypt(key, plaintext, []byte(sessionID))
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{
		V:          1,
		Kind:       "event",
		SessionID:  sessionID,
		Nonce:      base64.RawURLEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawURLEncoding.EncodeToString(ct),
	}, nil
}

// Open decrypts an Envelope with the session key.
func (e Envelope) Open(key *[32]byte) ([]byte, error) {
	nonce, err := base64.RawURLEncoding.DecodeString(e.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	ct, err := base64.RawURLEncoding.DecodeString(e.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}
	return Decrypt(key, nonce, ct, []byte(e.SessionID))
}

// Encrypt seals plaintext with AES-256-GCM using a random 12-byte nonce.
func Encrypt(key *[32]byte, plaintext, aad []byte) (nonce, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	return nonce, gcm.Seal(nil, nonce, plaintext, aad), nil
}

// Decrypt opens an AES-256-GCM sealed message.
func Decrypt(key *[32]byte, nonce, ciphertext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("bad nonce size %d", len(nonce))
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

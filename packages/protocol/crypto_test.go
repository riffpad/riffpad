package protocol

import (
	"bytes"
	"testing"
)

func TestDeviceAndSessionKeyAgreement(t *testing.T) {
	for _, curve := range []Curve{CurveX25519, CurveP256} {
		t.Run(string(curve), func(t *testing.T) {
			server, err := GenerateKeyPair(curve)
			if err != nil {
				t.Fatal(err)
			}
			client, err := GenerateKeyPair(curve)
			if err != nil {
				t.Fatal(err)
			}

			secretA, err := NewDeviceSecret(server, client.PublicKey)
			if err != nil {
				t.Fatal(err)
			}
			secretB, err := NewDeviceSecret(client, server.PublicKey)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(secretA, secretB) {
				t.Fatal("device secrets do not match")
			}

			serverEph, _ := GenerateKeyPair(curve)
			clientEph, _ := GenerateKeyPair(curve)
			ephA, _ := ECDH(serverEph, clientEph.PublicKey)
			ephB, _ := ECDH(clientEph, serverEph.PublicKey)
			if !bytes.Equal(ephA, ephB) {
				t.Fatal("ephemeral secrets do not match")
			}

			keyA, err := DeriveSessionKey(secretA, ephA, "s1")
			if err != nil {
				t.Fatal(err)
			}
			keyB, err := DeriveSessionKey(secretB, ephB, "s1")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(keyA[:], keyB[:]) {
				t.Fatal("session keys do not match")
			}

			// Different session ID -> different key.
			keyOther, _ := DeriveSessionKey(secretA, ephA, "s2")
			if bytes.Equal(keyA[:], keyOther[:]) {
				t.Fatal("session keys must differ across sessions")
			}

			// Restore from raw private bytes.
			restored, err := RestoreKeyPair(curve, server.PublicKey, server.PrivateKey())
			if err != nil {
				t.Fatal(err)
			}
			secretC, _ := NewDeviceSecret(restored, client.PublicKey)
			if !bytes.Equal(secretA, secretC) {
				t.Fatal("restored key pair produced different secret")
			}
		})
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := &[32]byte{}
	for i := range key {
		key[i] = byte(i)
	}
	nonce, ct, err := Encrypt(key, []byte("hello"), []byte("aad"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decrypt(key, nonce, ct, []byte("aad"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("unexpected plaintext %q", got)
	}
	if _, err := Decrypt(key, nonce, ct, []byte("other-aad")); err == nil {
		t.Fatal("expected failure with wrong AAD")
	}
}

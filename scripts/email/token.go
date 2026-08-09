package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strings"
)

// normalizeEmail canonicalizes an address: lower-case, trimmed, no display
// name. The relay stores and matches the same form.
func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func validEmail(s string) bool {
	if s == "" || strings.ContainsAny(s, "\r\n ") {
		return false
	}
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}

func signEmail(email string) (string, error) {
	secret := os.Getenv("UNSUBSCRIBE_SECRET")
	if secret == "" {
		return "", fmt.Errorf("UNSUBSCRIBE_SECRET is not set")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(normalizeEmail(email)))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func validSignature(email, token string) bool {
	want, err := signEmail(email)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(want)) == 1
}

func unsubscribeURL(email string) (string, error) {
	email = normalizeEmail(email)
	if !validEmail(email) {
		return "", fmt.Errorf("invalid email %q", email)
	}
	tok, err := signEmail(email)
	if err != nil {
		return "", err
	}
	base := strings.TrimSuffix(envOr("UNSUBSCRIBE_BASE_URL", "https://riffpad.ai/unsubscribe"), "/")
	return base + "?email=" + url.QueryEscape(email) + "&token=" + tok, nil
}

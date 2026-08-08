package protocol

import (
	"encoding/json"
	"testing"
)

func TestNewEventRoundTrip(t *testing.T) {
	ev, err := NewEvent("s1", EventApprovalReq, ApprovalRequestPayload{
		RequestID: "req1",
		Action:    "file_delete",
		Summary:   "删除 src/old.ts",
		Options:   []string{"approve", "reject"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ev.SessionID != "s1" || ev.Type != EventApprovalReq {
		t.Fatalf("unexpected event: %+v", ev)
	}
	var p ApprovalRequestPayload
	if err := ev.DecodePayload(&p); err != nil {
		t.Fatal(err)
	}
	if p.RequestID != "req1" || p.Summary != "删除 src/old.ts" {
		t.Fatalf("unexpected payload: %+v", p)
	}
}

func TestNewEventRequiresTypeAndSession(t *testing.T) {
	if _, err := NewEvent("", EventNotify, NotifyPayload{}); err == nil {
		t.Fatal("expected error for empty sessionId")
	}
	if _, err := NewEvent("s1", "", NotifyPayload{}); err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestEnvelopeRoundTripAndTamper(t *testing.T) {
	key := &[32]byte{}
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte(`{"type":"agent_status","status":"running"}`)
	env, err := WrapEnvelope("s1", plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	if env.SessionID != "s1" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
	got, err := env.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("round trip mismatch: %s", got)
	}

	// Wrong key must fail.
	wrong := &[32]byte{}
	if _, err := env.Open(wrong); err == nil {
		t.Fatal("expected decrypt failure with wrong key")
	}

	// Tampered ciphertext must fail.
	ct, _ := decodeRaw(env.Ciphertext)
	ct[0] ^= 0xff
	env.Ciphertext = encodeRaw(ct)
	if _, err := env.Open(key); err == nil {
		t.Fatal("expected decrypt failure with tampered ciphertext")
	}
}

func TestEnvelopeJSONShape(t *testing.T) {
	key := &[32]byte{}
	env, err := WrapEnvelope("s1", []byte(`{}`), key)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"v", "kind", "sessionId", "nonce", "ciphertext"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing field %s in %s", k, b)
		}
	}
}

func decodeRaw(s string) ([]byte, error) { return DecodeKey(s) }
func encodeRaw(b []byte) string           { return EncodeKey(b) }

func TestIsCriticalEvent(t *testing.T) {
	critical := []string{EventApprovalReq, EventApprovalResp, EventSessionEnd}
	for _, typ := range critical {
		if !IsCriticalEvent(typ) {
			t.Fatalf("%s must be critical", typ)
		}
	}
	nonCritical := []string{EventAgentMessage, EventToolCall, EventNotify, EventSessionStart, ""}
	for _, typ := range nonCritical {
		if IsCriticalEvent(typ) {
			t.Fatalf("%s must not be critical", typ)
		}
	}
}

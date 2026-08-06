package daemon

import (
	"encoding/hex"
	"io"
	"log"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/packages/protocol"
)

func TestSessionEventEncryptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	ev, err := protocol.NewEvent("s1", protocol.EventAgentMessage,
		protocol.AgentMessagePayload{Text: "hello persisted"})
	if err != nil {
		t.Fatal(err)
	}
	if err := appendSessionEvent(dir, "s1", key, ev); err != nil {
		t.Fatal(err)
	}
	events, err := loadSessionEvents(dir, "s1", key)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != protocol.EventAgentMessage {
		t.Fatalf("unexpected events: %+v", events)
	}
	var p protocol.AgentMessagePayload
	if err := events[0].DecodePayload(&p); err != nil || p.Text != "hello persisted" {
		t.Fatalf("payload mismatch: %+v err=%v", p, err)
	}
	// A different key must not decrypt.
	other := make([]byte, 32)
	other[0] = 1
	if _, err := loadSessionEvents(dir, "s1", other); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreSessionsSkipsEnded(t *testing.T) {
	dir := t.TempDir()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := hex.DecodeString(keys.SessionEncKey)
	now := time.Now()
	live := &PersistedSession{ID: "live-1", Name: "n", CLI: "kimi", Cwd: "/tmp", Status: "running", CreatedAt: now}
	ended := &PersistedSession{ID: "ended-1", Name: "e", CLI: "codex", Cwd: "/tmp", Status: "done", Ended: true, CreatedAt: now}
	if err := persistSessionMeta(dir, live); err != nil {
		t.Fatal(err)
	}
	if err := persistSessionMeta(dir, ended); err != nil {
		t.Fatal(err)
	}
	ev, _ := protocol.NewEvent("live-1", protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: "x"})
	if err := appendSessionEvent(dir, "live-1", key, ev); err != nil {
		t.Fatal(err)
	}

	logger := log.New(io.Discard, "", 0)
	srv := New(config.Default(), keys, dir, logger, nil)
	srv.mu.Lock()
	_, hasLive := srv.sessions["live-1"]
	_, hasEnded := srv.sessions["ended-1"]
	srv.mu.Unlock()
	if !hasLive {
		t.Fatal("live session should be restored")
	}
	if hasEnded {
		t.Fatal("ended session should not be restored")
	}
}

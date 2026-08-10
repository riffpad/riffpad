package hub

import (
	"testing"
	"time"
)

func TestFindOrCreateGitHubUser(t *testing.T) {
	s, err := OpenStore(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	u1, err := s.FindOrCreateGitHubUser("12345", "octocat", "octo@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if u1.Username != "octocat" {
		t.Fatalf("unexpected username: %s", u1.Username)
	}
	if u1.Email != "octo@example.com" {
		t.Fatalf("unexpected email: %s", u1.Email)
	}
	if u1.PasswordHash != "" {
		t.Fatal("github user should be passwordless")
	}

	// Same GitHub identity must map to the same account.
	u2, err := s.FindOrCreateGitHubUser("12345", "octocat", "")
	if err != nil {
		t.Fatal(err)
	}
	if u2.ID != u1.ID {
		t.Fatalf("github identity created two users: %s != %s", u1.ID, u2.ID)
	}

	// A different GitHub identity with the same login gets a unique username.
	u3, err := s.FindOrCreateGitHubUser("99999", "octocat", "")
	if err != nil {
		t.Fatal(err)
	}
	if u3.ID == u1.ID {
		t.Fatal("different github identity must not share an account")
	}
	if u3.Username == u1.Username {
		t.Fatalf("username collision not resolved: %s", u3.Username)
	}
}

func TestGitHubUserLoginFailsWithoutPassword(t *testing.T) {
	s, err := OpenStore(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.FindOrCreateGitHubUser("12345", "octocat", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.VerifyLogin(u.Username, "anything"); err == nil {
		t.Fatal("passwordless github user should not accept password login")
	}
}

func TestUpsertSessionsKeepsAnnouncedLastSeenAt(t *testing.T) {
	s, err := OpenStore(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-30 * time.Minute)
	if err := s.UpsertSessions("host1", []SessionMeta{
		{ID: "s1", CLI: "claude", Status: "running", LastSeenAt: old},
		{ID: "s2", CLI: "claude", Status: "running"}, // no timestamp: legacy daemon
	}); err != nil {
		t.Fatal(err)
	}

	var got []SessionMeta
	if err := s.db.Where("host_id = ?", "host1").Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	byID := map[string]SessionMeta{}
	for _, m := range got {
		byID[m.ID] = m
	}
	if len(byID) != 2 {
		t.Fatalf("expected 2 session rows, got %d", len(byID))
	}
	if delta := byID["s1"].LastSeenAt.Sub(old); delta > time.Minute || delta < -time.Minute {
		t.Fatalf("s1 lastSeenAt overwritten: got %v want %v", byID["s1"].LastSeenAt, old)
	}
	if byID["s2"].LastSeenAt.IsZero() || time.Since(byID["s2"].LastSeenAt) > time.Minute {
		t.Fatalf("zero timestamp not stamped live: %v", byID["s2"].LastSeenAt)
	}
}

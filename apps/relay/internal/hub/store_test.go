package hub

import (
	"testing"
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

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnsubscribeTokenRoundTrip(t *testing.T) {
	t.Setenv("UNSUBSCRIBE_SECRET", "test-secret")
	u, err := unsubscribeURL("User@Example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u, "https://riffpad.ai/unsubscribe?email=user%40example.com&token=") {
		t.Fatalf("unexpected URL: %s", u)
	}
	tok := strings.TrimPrefix(u, "https://riffpad.ai/unsubscribe?email=user%40example.com&token=")
	if !validSignature("USER@example.com", tok) {
		t.Fatal("valid signature rejected")
	}
	if validSignature("other@example.com", tok) {
		t.Fatal("signature accepted for another email")
	}
	if validSignature("user@example.com", "deadbeef") {
		t.Fatal("forged signature accepted")
	}
}

func TestUnsubscribeURLRejectsInvalidEmail(t *testing.T) {
	t.Setenv("UNSUBSCRIBE_SECRET", "s")
	if _, err := unsubscribeURL("not-an-email"); err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestLoadRecipients(t *testing.T) {
	path := filepath.Join(t.TempDir(), "waitlist.csv")
	data := "Email,Name,Date\n  Alice@Example.com ,Alice,2026-08-01\nbob@example.com,Bob,2026-08-02\nbob@example.com,Dup,2026-08-03\nnope,\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	recs, err := loadRecipients(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("want 2 unique recipients, got %d", len(recs))
	}
	if recs[0].Email != "alice@example.com" || recs[0].Name != "Alice" {
		t.Fatalf("unexpected first recipient: %+v", recs[0])
	}
}

func TestRenderBodyAppendsUnsubscribeFooter(t *testing.T) {
	tmpl, err := loadTemplate(filepath.Join(t.TempDir(), "missing.txt"))
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	path := filepath.Join(t.TempDir(), "body.txt")
	if err := os.WriteFile(path, []byte("Hello {{.Email}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	tmpl, err = loadTemplate(path)
	if err != nil {
		t.Fatal(err)
	}
	body, err := renderBody(tmpl, recipient{Email: "a@b.c", Name: "Ann"}, "https://riffpad.ai/unsubscribe?email=a%40b.c&token=x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "Hello a@b.c") || !strings.Contains(body, "https://riffpad.ai/unsubscribe") {
		t.Fatalf("body missing content: %q", body)
	}
}

func TestFetchOptOuts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/waitlist/optouts" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-Admin-Key") != "k" {
			t.Errorf("missing admin key")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"emails":["a@b.c"," d@e.f "]}`))
	}))
	t.Setenv("RIFFPAD_API_URL", ts.URL)
	got, err := fetchOptOuts("k")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a@b.c" || got[1] != " d@e.f " {
		t.Fatalf("unexpected optouts: %v", got)
	}
}

func TestFetchWaitlistEmails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/waitlist/emails" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-Admin-Key") != "k" {
			t.Errorf("missing admin key")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"entries":[{"email":"a@b.c","createdAt":"2026-07-12T12:13:00Z"},{"email":"bad","createdAt":"2026-07-12T12:14:00Z"}]}`))
	}))
	t.Setenv("RIFFPAD_API_URL", ts.URL)
	recs, err := fetchWaitlist("k")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Email != "a@b.c" || recs[0].Date != "2026-07-12T12:13:00Z" {
		t.Fatalf("unexpected waitlist: %v", recs)
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := normalizeEmail("  Foo@Bar.COM "); got != "foo@bar.com" {
		t.Fatalf("normalize: %q", got)
	}
}

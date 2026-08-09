package hub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newWaitlistTestHub(t *testing.T) (*Hub, *httptest.Server) {
	t.Helper()
	t.Setenv("UNSUBSCRIBE_SECRET", "test-secret")
	t.Setenv("WAITLIST_ADMIN_KEY", "admin-key")
	t.Setenv("RIFFPAD_WEB_ORIGINS", "https://riffpad.ai,https://www.riffpad.ai")
	h, err := New(log.New(io.Discard, "", 0), t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(h.Handler())
	t.Cleanup(ts.Close)
	return h, ts
}

func TestWaitlistUnsubscribe(t *testing.T) {
	h, ts := newWaitlistTestHub(t)
	email := "User@Example.com"
	token, err := waitlistSign(email)
	if err != nil {
		t.Fatal(err)
	}

	post := func(origin string, body string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/waitlist/unsubscribe", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { resp.Body.Close() })
		return resp
	}

	body, _ := json.Marshal(map[string]string{"email": email, "token": token})
	resp := post("https://riffpad.ai", string(body))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid unsubscribe status %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "https://riffpad.ai" {
		t.Fatalf("missing CORS header: %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}

	// Idempotent: unsubscribing twice is still 200.
	resp = post("https://www.riffpad.ai", string(body))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("repeated unsubscribe status %d", resp.StatusCode)
	}

	// Disallowed origin must not get CORS headers.
	resp = post("https://evil.example", string(body))
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("CORS header leaked to disallowed origin")
	}

	// Bad token is rejected.
	resp = post("", `{"email":"a@b.c","token":"deadbeef"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("forged token status %d", resp.StatusCode)
	}

	// Invalid email is rejected.
	resp = post("", `{"email":"nope","token":"x"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid email status %d", resp.StatusCode)
	}

	// Preflight from an allowed origin passes.
	req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/api/waitlist/unsubscribe", nil)
	req.Header.Set("Origin", "https://riffpad.ai")
	req.Header.Set("Access-Control-Request-Method", "POST")
	pre, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer pre.Body.Close()
	if pre.StatusCode != http.StatusNoContent || pre.Header.Get("Access-Control-Allow-Origin") != "https://riffpad.ai" {
		t.Fatalf("preflight failed: status %d, ACAO %q", pre.StatusCode, pre.Header.Get("Access-Control-Allow-Origin"))
	}

	optedOut, err := h.store.EmailOptedOut(strings.ToLower(email))
	if err != nil {
		t.Fatal(err)
	}
	if !optedOut {
		t.Fatal("email not recorded as opted out")
	}
}

func TestWaitlistOptoutsAdminOnly(t *testing.T) {
	_, ts := newWaitlistTestHub(t)
	if _, err := waitlistSign("a@b.c"); err != nil {
		t.Fatal(err)
	}
	if err := postJSON(ts.URL+"/api/waitlist/unsubscribe", map[string]string{
		"email": "a@b.c", "token": mustToken(t, "a@b.c"),
	}); err != nil {
		t.Fatal(err)
	}

	get := func(key string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/waitlist/optouts", nil)
		if key != "" {
			req.Header.Set("X-Admin-Key", key)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { resp.Body.Close() })
		return resp
	}

	if resp := get(""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing key status %d", resp.StatusCode)
	}
	if resp := get("wrong"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong key status %d", resp.StatusCode)
	}
	resp := get("admin-key")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid key status %d", resp.StatusCode)
	}
	var out struct {
		Emails []string `json:"emails"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Emails) != 1 || out.Emails[0] != "a@b.c" {
		t.Fatalf("unexpected optouts: %v", out.Emails)
	}
}

func mustToken(t *testing.T, email string) string {
	t.Helper()
	tok, err := waitlistSign(email)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func postJSON(url string, v any) error {
	b, _ := json.Marshal(v)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unsubscribe status %d", resp.StatusCode)
	}
	return nil
}

package hub

import (
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
)

// TestRouteTable locks the relay's HTTP surface. The hub.go split (#294) moved
// every handler into its own file; this is the guard that the wiring in
// Handler() still exposes exactly the same routes.
//
// A route counts as registered when the request does not fall through to the
// ServeMux's own 404. Handlers that legitimately answer 404 for a bare request
// (unknown ids, auth failures) still prove registration as long as the body is
// not the mux's default — but the genuine default is exactly what an unmounted
// path returns, so the negative controls below pin it.
func TestRouteTable(t *testing.T) {
	_, ts := newTestHub(t)

	routed := []struct{ method, path string }{
		{http.MethodGet, "/"},
		{http.MethodGet, "/device"},
		{http.MethodGet, "/api/status"},
		{http.MethodPost, "/api/auth/register"},
		{http.MethodPost, "/api/auth/login"},
		{http.MethodPost, "/api/auth/logout"},
		{http.MethodGet, "/api/auth/me"},
		{http.MethodGet, "/api/auth/github/login"},
		{http.MethodGet, "/api/auth/github/callback"},
		{http.MethodPost, "/api/auth/oauth/device"},
		{http.MethodPost, "/api/auth/oauth/device/poll"},
		{http.MethodGet, "/api/auth/oauth/device/status"},
		{http.MethodGet, "/api/hosts"},
		{http.MethodPost, "/api/hosts/register"},
		{http.MethodGet, "/api/hosts/some-id"},
		{http.MethodPost, "/api/pairings"},
		{http.MethodPost, "/api/pair"},
		{http.MethodGet, "/api/devices"},
		{http.MethodDelete, "/api/devices/some-id"},
		{http.MethodGet, "/api/sessions"},
		{http.MethodPut, "/api/sessions/some-id/meta"},
		{http.MethodPost, "/api/waitlist/subscribe"},
		{http.MethodGet, "/api/waitlist/emails"},
		{http.MethodGet, "/api/waitlist/unsubscribe"},
		{http.MethodGet, "/api/waitlist/optouts"},
		{http.MethodGet, "/ws"},
		{http.MethodGet, "/ws/host"},
	}

	for _, rt := range routed {
		resp, body := call(t, ts.URL, rt.method, rt.path)
		if strings.TrimSpace(body) == muxNotFound {
			t.Errorf("%s %s is no longer routed by Handler()", rt.method, rt.path)
			continue
		}
		t.Logf("%s %s -> %d", rt.method, rt.path, resp)
	}

	// Negative controls: these must fall through to the mux default, otherwise
	// the check above proves nothing.
	for _, path := range []string{"/api/nope", "/nope/deep", "/api", "/apifoo"} {
		_, body := call(t, ts.URL, http.MethodGet, path)
		if strings.TrimSpace(body) != muxNotFound {
			t.Errorf("GET %s unexpectedly handled (body %q)", path, truncate(body, 60))
		}
	}
}

const muxNotFound = "404 page not found"

func call(t *testing.T, base, method, path string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, base+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(body)
}

// truncate is local to this test: the production helper lives in store.go.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// TestGitHubEndpointsFromEnv pins the contract the core-path browser test
// (#314) depends on: with the three GITHUB_*_URL variables set, the compiled
// relay drives the entire OAuth flow against a stub instead of github.com.
func TestGitHubEndpointsFromEnv(t *testing.T) {
	t.Setenv("GITHUB_AUTHORIZE_URL", "http://127.0.0.1:1/authorize")
	t.Setenv("GITHUB_TOKEN_URL", "http://127.0.0.1:2/token")
	t.Setenv("GITHUB_USER_URL", "http://127.0.0.1:3/user")

	h, err := New(log.New(io.Discard, "", 0), t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if h.githubAuthorizeURL != "http://127.0.0.1:1/authorize" {
		t.Errorf("authorize URL = %q", h.githubAuthorizeURL)
	}
	if h.githubTokenURL != "http://127.0.0.1:2/token" {
		t.Errorf("token URL = %q", h.githubTokenURL)
	}
	if h.githubUserURL != "http://127.0.0.1:3/user" {
		t.Errorf("user URL = %q", h.githubUserURL)
	}

	// Defaults stay production GitHub when nothing is set.
	t.Setenv("GITHUB_AUTHORIZE_URL", "")
	t.Setenv("GITHUB_TOKEN_URL", "")
	t.Setenv("GITHUB_USER_URL", "")
	h2, err := New(log.New(io.Discard, "", 0), t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if h2.githubAuthorizeURL != "https://github.com/login/oauth/authorize" ||
		h2.githubTokenURL != "https://github.com/login/oauth/access_token" ||
		h2.githubUserURL != "https://api.github.com/user" {
		t.Errorf("defaults changed: %q %q %q", h2.githubAuthorizeURL, h2.githubTokenURL, h2.githubUserURL)
	}
}

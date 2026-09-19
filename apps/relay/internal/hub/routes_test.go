package hub

import (
	"io"
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

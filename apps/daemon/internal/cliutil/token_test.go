package cliutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
)

// stubToken stubs the CLI's local token for the duration of a test.
func stubToken(t *testing.T, token string) {
	t.Helper()
	old := cliToken
	SetToken(token)
	t.Cleanup(func() { SetToken(old) })
}

func TestDaemonDoAttachesLocalToken(t *testing.T) {
	stubToken(t, "test-token")
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(daemon.LocalTokenHeader)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	resp, err := DaemonDo(nil, http.MethodGet, srv.URL+"/api/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got != "test-token" {
		t.Fatalf("expected %s header %q, got %q", daemon.LocalTokenHeader, "test-token", got)
	}
}

// The CLI health check must authenticate too: a daemon that requires the
// local token is "reachable" only when the CLI presents it.
func TestReachableWithTokenRequiringDaemon(t *testing.T) {
	stubToken(t, "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(daemon.LocalTokenHeader) != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	if !Reachable(srv.URL) {
		t.Fatal("reachable should succeed with the local token")
	}

	stubToken(t, "wrong-token")
	if Reachable(srv.URL) {
		t.Fatal("reachable should fail with the wrong token")
	}
}

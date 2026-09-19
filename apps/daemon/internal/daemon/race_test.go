package daemon

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
)

// TestSessionStateConcurrentStopAndPump pins the fix for #312: a session is
// created and stopped while its pump goroutine is still delivering the first
// events. The pump persists session state on every event, the HTTP handler
// flips status/ended — before sess.mu guarded those fields this was a data
// race (`go test -race` reported stopSession's write racing persistSession's
// read, failing TestDemoSessionFactoryWiring in CI).
//
// Repeated because the window is timing-dependent; run with -race for this to
// mean anything (CI does).
func TestSessionStateConcurrentStopAndPump(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, keys, dir, log.New(io.Discard, "", 0), nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	post := func(path string) *http.Response {
		req, err := http.NewRequest(http.MethodPost, ts.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(LocalTokenHeader, cfg.LocalToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	done := make(chan string, 64)
	for i := 0; i < 25; i++ {
		resp := authRequest(t, http.MethodPost, ts.URL+"/api/sessions", cfg.LocalToken,
			strings.NewReader(`{"cli":"demo","cwd":"/tmp"}`))
		var created struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if created.ID == "" {
			t.Fatal("demo session not created")
		}
		// Stop from another goroutine so the request races the pump, exactly
		// like a user hitting stop while the agent is still emitting events.
		go func(id string) {
			r := post("/api/sessions/" + id + "/stop")
			_, _ = io.Copy(io.Discard, r.Body)
			r.Body.Close()
			done <- id
		}(created.ID)
		// Also read the list: handleSessions walks the same fields.
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
		req.Header.Set(LocalTokenHeader, cfg.LocalToken)
		if r, err := http.DefaultClient.Do(req); err == nil {
			_, _ = io.Copy(io.Discard, r.Body)
			r.Body.Close()
		}
	}
	for i := 0; i < 25; i++ {
		<-done
	}
}

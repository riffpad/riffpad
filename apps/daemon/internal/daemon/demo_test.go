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

// TestDemoSessionFactoryWiring: cli=demo must produce a runnable session
// through the real HTTP create path, and stop cleanly.
func TestDemoSessionFactoryWiring(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, keys, dir, log.New(io.Discard, "", 0), nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := authRequest(t, http.MethodPost, ts.URL+"/api/sessions", cfg.LocalToken,
		strings.NewReader(`{"cli":"demo","cwd":"/tmp"}`))
	var sess struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || sess.ID == "" {
		t.Fatalf("demo session create failed: status %d id %q", resp.StatusCode, sess.ID)
	}

	stop := authRequest(t, http.MethodPost, ts.URL+"/api/sessions/"+sess.ID+"/stop", cfg.LocalToken, nil)
	stop.Body.Close()
	if stop.StatusCode != http.StatusOK {
		t.Fatalf("demo session stop failed: %d", stop.StatusCode)
	}
}

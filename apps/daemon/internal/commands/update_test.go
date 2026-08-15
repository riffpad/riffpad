package commands

// Characterization tests for the update path (updateCmd, verifyChecksum,
// compareVersions, updatePlatform), added before the #282 file split so the
// refactor cannot silently change behavior.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubUpdateServer points updateCmd's GitHub endpoints at a fake release
// server serving the given tag and asset bytes with a matching sha256sums
// entry. Cleanup restores the real endpoints.
func stubUpdateServer(t *testing.T, tag string, asset []byte) {
	t.Helper()
	sum := sha256.Sum256(asset)
	want := hex.EncodeToString(sum[:])
	srv := newReleaseServer(t, tag, func(w http.ResponseWriter, name string) {
		switch name {
		case "sha256sums.txt":
			fmt.Fprintf(w, "%s  %s\n", want, updateAssetName())
		case updateAssetName():
			_, _ = w.Write(asset)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	stubUpdateEndpoints(t, srv)
}

// newReleaseServer builds a fake GitHub release server; download routes hand
// off to serve for each asset name.
func newReleaseServer(t *testing.T, tag string, serve func(w http.ResponseWriter, name string)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/riffpad/riffpad/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": tag})
	})
	mux.HandleFunc("/releases/latest/download/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/releases/latest/download/")
		serve(w, name)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// stubUpdateEndpoints redirects the updater at srv for this test.
func stubUpdateEndpoints(t *testing.T, srv *httptest.Server) {
	t.Helper()
	oldAPI, oldDL := updateAPIBase, updateDownloadBase
	updateAPIBase = srv.URL + "/repos"
	updateDownloadBase = srv.URL + "/releases/latest/download/"
	t.Cleanup(func() { updateAPIBase, updateDownloadBase = oldAPI, oldDL })
}

// updateAssetName computes the release asset name updatePlatform() picks for
// the current platform.
func updateAssetName() string {
	osName, arch, _ := updatePlatform()
	return "riffpad-" + osName + "-" + arch
}

// withFakeSelfExecutable stubs osExecutableFn to point at a throwaway copy
// of the test binary, so updateCmd never replaces the real one.
func withFakeSelfExecutable(t *testing.T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "riffpad")
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatal(err)
	}
	old := osExecutableFn
	osExecutableFn = func() (string, error) { return dst, nil }
	t.Cleanup(func() { osExecutableFn = old })
	return dst
}

// TestUpdateCmdReplacesBinary walks the whole happy path: latest tag →
// download → checksum → atomic replace with a .riffpad.bak backup.
func TestUpdateCmdReplacesBinary(t *testing.T) {
	stubUpdateServer(t, "v9.9.9", []byte("new-binary-content"))
	exe := withFakeSelfExecutable(t)

	if err := UpdateCmd([]string{"--no-restart", "--force"}, t.TempDir()); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new-binary-content" {
		t.Fatalf("binary not replaced, got %q", data)
	}
	if _, err := os.Stat(exe + ".riffpad.bak"); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
}

// TestUpdateCmdUpToDateSkipsDownload: version "dev" parses as 0.0.0, equal
// to latest v0.0.0, so no download happens (the fake server serves no asset
// for that path; a download attempt would 404 and fail the test).
func TestUpdateCmdUpToDateSkipsDownload(t *testing.T) {
	stubUpdateServer(t, "v0.0.0", []byte("unused"))
	withFakeSelfExecutable(t)

	if err := UpdateCmd(nil, t.TempDir()); err != nil {
		t.Fatalf("up-to-date update failed: %v", err)
	}
}

// TestUpdateCmdChecksumMismatchAborts: a wrong sha256sums entry must abort
// before any replace or backup happens.
func TestUpdateCmdChecksumMismatchAborts(t *testing.T) {
	srv := newReleaseServer(t, "v9.9.9", func(w http.ResponseWriter, name string) {
		switch name {
		case "sha256sums.txt":
			fmt.Fprintf(w, "%s  %s\n", strings.Repeat("0", 64), updateAssetName())
		case updateAssetName():
			_, _ = w.Write([]byte("new-binary-content"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	stubUpdateEndpoints(t, srv)
	exe := withFakeSelfExecutable(t)

	if err := UpdateCmd([]string{"--force", "--no-restart"}, t.TempDir()); err == nil {
		t.Fatal("expected checksum mismatch to abort the update")
	}
	if data, err := os.ReadFile(exe); err != nil || string(data) == "new-binary-content" {
		t.Fatalf("binary must stay unchanged, err=%v", err)
	}
	if _, err := os.Stat(exe + ".riffpad.bak"); err == nil {
		t.Fatal("no backup must be created on checksum failure")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.2.3", "1.2.3", 0},
		{"1.2.3", "1.2.4", -1},
		{"1.2", "1.2.0", 0},
		{"dev", "v0.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"0.1.0-m0", "v0.1.0", 0},
		{"1.0.0+build", "1.0.0", 0},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestVerifyChecksumMatches(t *testing.T) {
	data := []byte("payload")
	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])
	path := filepath.Join(t.TempDir(), "asset")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", want, "riffpad-test-asset")
	}))
	t.Cleanup(srv.Close)

	if err := verifyChecksum(srv.URL, "riffpad-test-asset", path); err != nil {
		t.Fatalf("checksum should match: %v", err)
	}
}

func TestVerifyChecksumMissingEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "aaaa  other-asset\n")
	}))
	t.Cleanup(srv.Close)

	err := verifyChecksum(srv.URL, "riffpad-test-asset", path)
	if err == nil || !strings.Contains(err.Error(), "no entry") {
		t.Fatalf("expected missing-entry error, got %v", err)
	}
}

func TestVerifyChecksumMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  riffpad-test-asset\n", strings.Repeat("0", 64))
	}))
	t.Cleanup(srv.Close)

	if err := verifyChecksum(srv.URL, "riffpad-test-asset", path); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

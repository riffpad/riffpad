package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
	"github.com/riffpad/riffpad/apps/daemon/internal/version"
)

const updateRepo = "riffpad/riffpad"

// updateAPIBase and updateDownloadBase point the updater at GitHub; they are
// vars so tests can serve a fake release server.
var (
	updateAPIBase      = "https://api.github.com/repos"
	updateDownloadBase = "https://github.com/" + updateRepo + "/releases/latest/download/"
)

func UpdateCmd(args []string, dataDir string) error {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	force := fs.Bool("force", false, "reinstall even if already up to date")
	noRestart := fs.Bool("no-restart", false, "do not restart the daemon after updating")
	_ = fs.Parse(args)

	fmt.Println(t.T("update_current", version.Version))
	latest, err := latestReleaseTag()
	if err != nil {
		return err
	}
	fmt.Println(t.T("update_latest", latest))
	if !*force && compareVersions(version.Version, latest) >= 0 {
		fmt.Println(t.T("update_up_to_date"))
		return nil
	}

	osName, arch, err := updatePlatform()
	if err != nil {
		return err
	}
	asset := "riffpad-" + osName + "-" + arch
	base := updateDownloadBase
	fmt.Println(t.T("update_downloading", asset))

	exe, err := osExecutableFn()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".riffpad-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := downloadFile(base+asset, tmp); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := verifyChecksum(base+"sha256sums.txt", asset, tmpPath); err != nil {
		fmt.Println(t.T("update_checksum_failed"))
		return err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}

	backup := exe + ".riffpad.bak"
	if err := copyFile(exe, backup); err != nil {
		fmt.Println(t.T("update_backup_failed", err))
		return err
	}
	if err := os.Rename(tmpPath, exe); err != nil {
		fmt.Println(t.T("update_replace_failed", err))
		return err
	}
	fmt.Println(t.T("update_done", latest, backup))
	if !*noRestart && cliutil.Reachable(defaultDaemonBase()) {
		return daemonRestart(defaultDaemonBase(), dataDir)
	}
	fmt.Println(t.T("update_restart_hint"))
	return nil
}

func defaultDaemonBase() string {
	if b := os.Getenv("RIFFPAD_URL"); b != "" {
		return b
	}
	return "http://127.0.0.1:8787"
}

func latestReleaseTag() (string, error) {
	// http.DefaultClient has no timeout; a wedged network must fail the
	// update instead of hanging it forever (#174).
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(updateAPIBase + "/" + updateRepo + "/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", t.T("latest_release_failed", resp.Status))
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("%s", t.T("release_no_tag"))
	}
	return rel.TagName, nil
}

func updatePlatform() (string, string, error) {
	osName := strings.ToLower(runtime.GOOS)
	if osName != "linux" && osName != "darwin" {
		return "", "", fmt.Errorf("%s", t.T("update_platform_unsupported", runtime.GOOS))
	}
	arch := runtime.GOARCH
	if arch == "x86_64" {
		arch = "amd64"
	}
	if arch == "aarch64" {
		arch = "arm64"
	}
	if arch != "amd64" && arch != "arm64" {
		return "", "", fmt.Errorf("%s", t.T("update_arch_unsupported", runtime.GOARCH))
	}
	return osName, arch, nil
}

func downloadFile(url string, w io.Writer) error {
	// Binary downloads get a generous but bounded timeout (#174).
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", t.T("download_failed", url, resp.Status))
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func verifyChecksum(sumsURL, asset, path string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(sumsURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", t.T("download_failed", sumsURL, resp.Status))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	want := ""
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == asset {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("%s", t.T("checksum_missing", asset))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("SHA256 不匹配（期望 %s，实际 %s）", want, got)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// compareVersions compares semver-ish versions, ignoring "v" prefixes and
// prerelease/build suffixes (e.g. 0.1.0-m0 == v0.1.0).
func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	if i := strings.IndexAny(a, "-+"); i >= 0 {
		a = a[:i]
	}
	if i := strings.IndexAny(b, "-+"); i >= 0 {
		b = b[:i]
	}
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		av, bv := 0, 0
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

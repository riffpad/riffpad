// Command riffpad is the single Riffpad binary: user-facing CLI plus the
// hidden `_daemon` subcommand that runs the background daemon.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/cliutil"
	"github.com/riffpad/riffpad/apps/daemon/internal/commands"
	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/console"
	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
	"github.com/riffpad/riffpad/apps/daemon/internal/i18n"
	"github.com/riffpad/riffpad/apps/daemon/internal/logging"
	"github.com/riffpad/riffpad/apps/daemon/internal/version"
)

const updateRepo = "riffpad/riffpad"

// updateAPIBase and updateDownloadBase point the updater at GitHub; they are
// vars so tests can serve a fake release server.
var (
	updateAPIBase      = "https://api.github.com/repos"
	updateDownloadBase = "https://github.com/" + updateRepo + "/releases/latest/download/"
)

// t is the active language bundle, initialized in main from --lang.
var t = i18n.New(i18n.DefaultLang)

// localToken and daemonDo delegate to internal/cliutil, which owns the
// daemon's local API token resolution shared by all CLI commands.
func localToken() string { return cliutil.LocalToken() }
func daemonDo(client *http.Client, method, url string, body io.Reader) (*http.Response, error) {
	return cliutil.DaemonDo(client, method, url, body)
}

func main() {
	langFlag, args := extractLangFlag(os.Args[1:])
	t = i18n.New(i18n.Detect(langFlag))
	commands.SetBundle(t)
	console.SetBundle(t)
	os.Args = append([]string{os.Args[0]}, args...)
	// Corrupted state files are backed up and rebuilt automatically (#172);
	// warn instead of dying at startup. runDaemon overrides this to also log.
	config.OnHeal = func(h config.Heal) {
		fmt.Fprintln(os.Stderr, t.T("config_file_healed", h.Path, h.Backup))
		if h.Kind == "keys" {
			fmt.Fprintln(os.Stderr, t.T("keys_regenerated_warn"))
		}
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if os.Args[1] == "_daemon" {
		os.Exit(runDaemon(os.Args[2:]))
	}
	base := os.Getenv("RIFFPAD_URL")
	if base == "" {
		base = "http://127.0.0.1:8787"
	}
	dataDir := os.Getenv("RIFFPAD_DIR")
	if dataDir == "" {
		d, err := config.DefaultDataDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, t.T("resolve_data_dir", err))
			os.Exit(1)
		}
		dataDir = d
	}
	cliutil.SetDataDir(dataDir)

	var err error
	switch os.Args[1] {
	case "daemon":
		err = daemonCmd(os.Args[2:], base, dataDir)
	case "status":
		err = statusCmd(base)
	case "pair":
		err = withDaemon(func() error { return pairCmd(base, os.Args[2:]) }, base, dataDir)
	case "sessions":
		err = withDaemon(func() error { return sessionsCmd(base) }, base, dataDir)
	case "run":
		err = withDaemon(func() error { return runCmd(os.Args[2:], base) }, base, dataDir)
	case "logs":
		err = logsCmd(dataDir)
	case "attach":
		err = withDaemon(func() error { return attachCmd(base) }, base, dataDir)
	case "detach":
		err = detachCmd()
	case "auth":
		err = authCmd(dataDir)
	case "login":
		err = loginCmd(os.Args[2:], dataDir)
	case "logout":
		err = logoutCmd(dataDir)
	case "relay":
		err = relayCmd(os.Args[2:], dataDir)
	case "setup":
		err = setupCmd(os.Args[2:], dataDir)
	case "kill":
		err = killCmd(base)
	case "update":
		err = updateCmd(os.Args[2:], dataDir)
	case "version":
		fmt.Println("riffpad", version.Version)
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "riffpad:", err)
		os.Exit(1)
	}
}

// runDaemon is the hidden `riffpad _daemon` entry point used by daemon start
// and systemd. Keeping the daemon inside the same binary makes installation a
// single-file affair.
func runDaemon(args []string) int {
	fs := flag.NewFlagSet("_daemon", flag.ExitOnError)
	dataDir := fs.String("data-dir", "", "daemon data directory (default ~/.config/riffpad)")
	_ = fs.Parse(args)
	enrichDaemonPath()
	dir := *dataDir
	if dir == "" {
		var err error
		dir, err = config.DefaultDataDir()
		if err != nil {
			log.Printf("resolve data dir: %v", err)
			return 1
		}
	}
	logger, closer, err := logging.New(dir)
	if err != nil {
		log.Printf("init logging: %v", err)
		return 1
	}
	defer closer.Close()
	// Route self-heal warnings (#172) through the daemon log as well.
	config.OnHeal = func(h config.Heal) {
		logger.Printf("%s", t.T("config_file_healed", h.Path, h.Backup))
		if h.Kind == "keys" {
			logger.Printf("%s", t.T("keys_regenerated_warn"))
		}
	}

	cfg, err := config.Load(dir)
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}
	keys, err := config.LoadOrCreateKeys(dir)
	if err != nil {
		logger.Fatalf("load keys: %v", err)
	}
	srv := daemon.New(cfg, keys, dir, logger, daemon.DefaultFactory())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("server: %v", err)
		}
		stop()
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown: %v", err)
	}
	logger.Printf("daemon stopped")
	return 0
}

// enrichDaemonPath appends common user bin directories to PATH so the daemon
// can spawn coding CLIs (codex/kimi/claude) even when it was started by
// systemd with a minimal PATH. Only existing directories are added; the
// authoritative PATH captured by `riffpad setup` and per-command binary paths
// passed by `riffpad run` take precedence because they are earlier in PATH.
func enrichDaemonPath() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	extra := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".codex", "bin"),
		filepath.Join(home, ".kimi-code", "bin"),
		filepath.Join(home, ".opencode", "bin"),
		filepath.Join(home, ".cargo", "bin"),
		filepath.Join(home, "go", "bin"),
		filepath.Join(home, ".bun", "bin"),
		filepath.Join(home, ".npm-global", "bin"),
	}
	seen := map[string]bool{}
	parts := make([]string, 0, 16)
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p != "" && !seen[p] {
			seen[p] = true
			parts = append(parts, p)
		}
	}
	for _, d := range extra {
		if seen[d] {
			continue
		}
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			parts = append(parts, d)
			seen[d] = true
		}
	}
	os.Setenv("PATH", strings.Join(parts, string(os.PathListSeparator)))
}

func usage() {
	fmt.Fprintln(os.Stderr, t.T("usage"))
}

// extractLangFlag pulls a global --lang/-lang value out of args so it works
// before any subcommand, regardless of position.
func extractLangFlag(args []string) (lang string, rest []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--lang" || a == "-lang":
			if i+1 < len(args) {
				lang = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--lang="):
			lang = strings.TrimPrefix(a, "--lang=")
		case strings.HasPrefix(a, "-lang="):
			lang = strings.TrimPrefix(a, "-lang=")
		default:
			rest = append(rest, a)
		}
	}
	return lang, rest
}

func daemonCmd(args []string, base, dataDir string) error {
	return commands.DaemonCmd(args, base, dataDir)
}

func withDaemon(fn func() error, base, dataDir string) error {
	return commands.WithDaemon(fn, base, dataDir)
}

func statusCmd(base string) error { return commands.StatusCmd(base) }

func pairCmd(base string, args []string) error { return commands.PairCmd(base, args) }

func sessionsCmd(base string) error { return commands.SessionsCmd(base) }

func runCmd(args []string, base string) error { return commands.RunCmd(args, base) }

func logsCmd(dataDir string) error { return commands.LogsCmd(dataDir) }

func attachCmd(base string) error { return commands.AttachCmd(base) }

func detachCmd() error { return commands.DetachCmd() }

func setupCmd(args []string, dataDir string) error { return commands.SetupCmd(args, dataDir) }

func loginCmd(args []string, dataDir string) error { return commands.LoginCmd(args, dataDir) }

func authCmd(dataDir string) error { return commands.AuthCmd(dataDir) }

func logoutCmd(dataDir string) error { return commands.LogoutCmd(dataDir) }

func relayCmd(args []string, dataDir string) error { return commands.RelayCmd(args, dataDir) }

func killCmd(base string) error { return commands.KillCmd(base) }

func updateCmd(args []string, dataDir string) error { return commands.UpdateCmd(args, dataDir) }

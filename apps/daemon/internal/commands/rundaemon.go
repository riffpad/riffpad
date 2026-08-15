package commands

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
	"github.com/riffpad/riffpad/apps/daemon/internal/logging"
)

// RunDaemon is the hidden `riffpad _daemon` entry point used by daemon start
// and systemd. Keeping the daemon inside the same binary makes installation a
// single-file affair.
func RunDaemon(args []string) int {
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

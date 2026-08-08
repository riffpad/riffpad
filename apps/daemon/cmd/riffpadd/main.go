package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/config"
	"github.com/riffpad/riffpad/apps/daemon/internal/daemon"
	"github.com/riffpad/riffpad/apps/daemon/internal/logging"
)

func main() {
	dataDir := flag.String("data-dir", "", "daemon data directory (default ~/.config/riffpad)")
	flag.Parse()
	dir := *dataDir
	if dir == "" {
		var err error
		dir, err = config.DefaultDataDir()
		if err != nil {
			log.Fatalf("resolve data dir: %v", err)
		}
	}
	logger, closer, err := logging.New(dir)
	if err != nil {
		log.Fatalf("init logging: %v", err)
	}
	defer closer.Close()
	// Corrupted state files are backed up and rebuilt automatically (#172);
	// log a loud warning instead of dying at startup.
	config.OnHeal = func(h config.Heal) {
		logger.Printf("WARNING: %s was corrupted; rebuilt with defaults (backup: %s)", h.Path, h.Backup)
		if h.Kind == "keys" {
			logger.Printf("WARNING: identity keys regenerated; all paired devices are now invalid and must re-pair")
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
		// Server closed (e.g. via POST /api/shutdown): exit the wait loop.
		stop()
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown: %v", err)
	}
	logger.Printf("daemon stopped")
}

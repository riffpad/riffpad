// Command relay is the cloud WebSocket relay for Riffpad. It forwards
// encrypted envelopes between local daemons (hosts) and mobile/web viewers
// without decrypting or persisting content.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riffpad/riffpad/apps/relay/internal/hub"
)

func main() {
	port := flag.String("port", envOr("RELAY_PORT", "9090"), "listen port")
	regKey := flag.String("registration-key", envOr("REGISTRATION_KEY", ""), "registration key for new hosts (empty = open registration)")
	dataDir := flag.String("data-dir", envOr("RELAY_DATA_DIR", "./relay-data"), "persistent data directory")
	flag.Parse()

	logger := log.New(os.Stdout, "relay: ", log.LstdFlags|log.LUTC)
	h := hub.New(logger, *regKey, *dataDir)
	srv := &http.Server{Addr: ":" + *port, Handler: h.Handler()}
	logger.Printf("riffpad relay listening on :%s (registration key required: %v, data dir: %s)", *port, *regKey != "", *dataDir)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("server: %v", err)
		}
		stop()
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Printf("relay stopped")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Per-endpoint cutover proxy for the Python→Go backend migration.
//
// Reads a routes config (default ./routes.yaml), matches each incoming request
// against its rules, and reverse-proxies to the configured upstream. Anything
// that doesn't match an explicit rule falls back to the default upstream.
//
// Flip a single endpoint to the Go backend by editing routes.yaml and either
// SIGHUP'ing the process (reload) or restarting it.
//
// Wire format and behavior parity is enforced upstream by backend-bb-tests/.
// Run that suite against the Go backend before flipping any route.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "routes.yaml", "Path to the routes config file.")
	addr := flag.String("addr", ":8080", "Address to listen on, e.g. :8080.")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		logger.Error("load config", "path", *configPath, "err", err)
		return 2
	}
	logger.Info("loaded config",
		"path", *configPath,
		"default", cfg.Default,
		"rules", len(cfg.Rules),
	)

	handler, err := NewProxy(cfg, logger)
	if err != nil {
		logger.Error("build proxy", "err", err)
		return 2
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	logger.Info("proxy listening", "addr", *addr)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", "err", err)
			return 1
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown", "err", err)
			return 1
		}
	}
	return 0
}

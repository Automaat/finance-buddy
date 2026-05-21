// Finance Buddy Go backend.
//
// Endpoints are cut over from the Python backend one at a time, each gated
// on the backend-bb-tests/ parity suite. See migration/CUTOVER.md.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/db"
	"github.com/Automaat/finance-buddy/backend-go/internal/server"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := server.Config{
		Addr:        envOr("FB_ADDR", ":8000"),
		CORSOrigins: envOrPresent("CORS_ORIGINS", "http://localhost:3000"),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps := server.Deps{}
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		pool, err := db.New(ctx, dsn)
		if err != nil {
			logger.Error("open db pool", "err", err)
			return 2
		}
		defer pool.Close()
		deps.Pool = pool
		logger.Info("db pool ready")
	} else {
		logger.Warn("DATABASE_URL not set — DB-backed endpoints will 404")
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.New(cfg, logger, deps),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	logger.Info("backend-go listening", "addr", cfg.Addr)

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

// envOr returns os.Getenv(key) if non-empty, else fallback. Use for values
// like FB_ADDR where empty is meaningless and we want a default.
func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// envOrPresent returns the env value if the key is set (even if empty),
// else fallback. Use for values where an explicit empty string is a
// legitimate signal — e.g. CORS_ORIGINS="" matching Python's behaviour
// of trusting whatever Settings parsed from the environment.
func envOrPresent(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

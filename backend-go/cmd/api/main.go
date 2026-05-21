// Finance Buddy Go backend.
//
// Endpoints are cut over from the Python backend one at a time, each gated
// on the backend-bb-tests/ parity suite. See migration/CUTOVER.md.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/db"
	"github.com/Automaat/finance-buddy/backend-go/internal/server"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	os.Exit(run())
}

// healthcheck probes our own /health endpoint via HTTP.
//
// Designed for Docker HEALTHCHECK on the distroless image, which has no shell
// or curl. Reads FB_ADDR for the port (default :8000) and pings on localhost.
// Exits 0 on a 200 response, 1 otherwise.
func healthcheck() int {
	addr := envOr("FB_ADDR", ":8000")
	host, port, ok := strings.Cut(addr, ":")
	if !ok {
		port = addr
	}
	if host == "" {
		host = "127.0.0.1"
	}
	url := "http://" + net.JoinHostPort(host, port) + "/health"
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
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
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" || os.Getenv("PGHOST") != "" {
		// pgx's URL parser is strict — special chars in the password (@, :,
		// /, ?, #) must be percent-encoded. If callers prefer to skip that
		// hazard, they can leave DATABASE_URL empty and provide the libpq
		// env vars (PGHOST/PGPORT/PGUSER/PGPASSWORD/PGDATABASE); pgx picks
		// them up from an empty DSN.
		pool, err := db.New(ctx, dsn)
		if err != nil {
			logger.Error("open db pool", "err", err)
			return 2
		}
		defer pool.Close()
		deps.Pool = pool
		logger.Info("db pool ready")
	} else {
		logger.Warn("no DB config (DATABASE_URL or PGHOST) — DB-backed endpoints will 404")
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

// Package db owns the Postgres connection pool wiring and the baseline
// schema.
//
// Since the Python backend (which ran Alembic) was decommissioned, the Go
// backend owns the schema. schema.sql is the frozen baseline — a pg_dump of
// the final Alembic head. ApplySchema bootstraps a fresh database with it
// and no-ops when the schema already exists, so production is never touched.
// Future schema changes are made directly here.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New opens a pgx pool against the given DSN and verifies it with a ping.
//
// The returned pool must be closed by the caller (typically via defer in main).
// Defaults applied: 10 max conns, 5-min idle timeout, 1-min health-check
// period. All other pool settings (acquire timeout, statement caching, etc)
// are pgx defaults; override via DSN query string if needed.
func New(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse db dsn: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open db pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

// Package db owns the Postgres connection pool wiring.
//
// Alembic owns the schema (migration prep decision); this layer only opens
// connections and executes queries.
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

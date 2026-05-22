package db

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

// ApplySchema bootstraps an empty database with the baseline schema.
//
// It is idempotent by presence check: if the `accounts` table already
// exists the schema is assumed complete and nothing runs — so an existing
// production database is never re-DDL'd. Fresh databases (CI, dev,
// new installs) get the full schema in one transaction-less batch, exactly
// as pg_dump emits it.
func ApplySchema(ctx context.Context, pool *pgxpool.Pool) error {
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT to_regclass('public.accounts') IS NOT NULL`,
	).Scan(&exists); err != nil {
		return fmt.Errorf("schema presence check: %w", err)
	}
	if exists {
		return nil
	}
	// schema.sql is many statements; the default extended protocol would run
	// only the first. The simple protocol executes the whole batch.
	if _, err := pool.Exec(ctx, schemaSQL, pgx.QueryExecModeSimpleProtocol); err != nil {
		return fmt.Errorf("apply baseline schema: %w", err)
	}
	return nil
}

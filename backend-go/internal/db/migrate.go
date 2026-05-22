package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ownerTables are the tables whose `owner` string is being migrated to an
// `owner_user_id` foreign key as part of the personas->users merge.
var ownerTables = []string{
	"accounts", "transactions", "salary_records", "debt_payments",
	"bonus_events", "equity_grants", "retirement_limits", "snapshot_aggregates",
}

// Migrate applies the in-progress personas->users schema migration. It is
// idempotent and runs on every startup, after the users table exists.
//
// Phase B: add a nullable owner_user_id column to every owner-bearing table
// and backfill it from the owner string (matched against users.name). Rows
// whose owner has no matching user — notably "Shared" — keep owner_user_id
// NULL. Re-running the backfill each startup also catches rows written before
// the cutover. The owner string column stays in place until a later phase.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	for _, table := range ownerTables {
		if _, err := pool.Exec(ctx,
			`ALTER TABLE `+table+` ADD COLUMN IF NOT EXISTS owner_user_id integer`); err != nil {
			return fmt.Errorf("add owner_user_id to %s: %w", table, err)
		}
		if _, err := pool.Exec(ctx, `
			UPDATE `+table+` AS t
			SET owner_user_id = u.id
			FROM users u
			WHERE t.owner = u.name AND t.owner_user_id IS NULL`); err != nil {
			return fmt.Errorf("backfill owner_user_id on %s: %w", table, err)
		}
	}
	return nil
}

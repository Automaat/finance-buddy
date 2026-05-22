package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ownerTables are the tables whose `owner` string is being migrated to an
// `owner_user_id` reference as part of the personas->users merge.
var ownerTables = []string{
	"accounts", "transactions", "salary_records", "debt_payments",
	"bonus_events", "equity_grants", "retirement_limits", "snapshot_aggregates",
}

// Migrate applies the in-progress personas->users schema migration. It is
// idempotent and runs on every startup, after the users table exists.
//
// Phase B: every owner-bearing table gets a nullable owner_user_id column,
// kept in sync with the owner string (matched against users.name). It is a
// plain integer for now — the foreign key constraint and index are added in
// a later phase. Rows whose owner matches no user — notably "Shared" — keep
// owner_user_id NULL.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	for _, table := range ownerTables {
		if err := ensureOwnerUserIDColumn(ctx, pool, table); err != nil {
			return err
		}
		if err := syncOwnerUserID(ctx, pool, table); err != nil {
			return err
		}
	}
	return nil
}

// ensureOwnerUserIDColumn adds owner_user_id only when it is missing, so a
// no-op startup does not take the ALTER TABLE access-exclusive lock.
func ensureOwnerUserIDColumn(ctx context.Context, pool *pgxpool.Pool, table string) error {
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = $1
			  AND column_name = 'owner_user_id')`,
		table).Scan(&exists); err != nil {
		return fmt.Errorf("check owner_user_id on %s: %w", table, err)
	}
	if exists {
		return nil
	}
	if _, err := pool.Exec(ctx,
		`ALTER TABLE `+table+` ADD COLUMN owner_user_id integer`); err != nil {
		return fmt.Errorf("add owner_user_id to %s: %w", table, err)
	}
	return nil
}

// syncOwnerUserID re-points owner_user_id at the user whose name equals the
// row's current owner, and clears it when the owner matches no user. Running
// it every startup keeps the two columns consistent — e.g. after an owner
// rename — until the owner string column is dropped.
func syncOwnerUserID(ctx context.Context, pool *pgxpool.Pool, table string) error {
	if _, err := pool.Exec(ctx, `
		UPDATE `+table+` AS t
		SET owner_user_id = u.id
		FROM users u
		WHERE t.owner = u.name AND t.owner_user_id IS DISTINCT FROM u.id`); err != nil {
		return fmt.Errorf("sync owner_user_id on %s: %w", table, err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE `+table+` AS t
		SET owner_user_id = NULL
		WHERE t.owner_user_id IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.name = t.owner)`); err != nil {
		return fmt.Errorf("clear stale owner_user_id on %s: %w", table, err)
	}
	return nil
}

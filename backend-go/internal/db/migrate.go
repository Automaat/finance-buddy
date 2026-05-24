package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ownerTables are the eight tables carrying an owner_user_id reference. The
// legacy `owner` string column is dropped from each as the final step of the
// personas->users merge.
var ownerTables = []string{
	"accounts", "transactions", "salary_records", "debt_payments",
	"bonus_events", "equity_grants", "retirement_limits", "snapshot_aggregates",
}

// ownerFK names the owner_user_id->users(id) foreign key per table. The names
// match schema.sql so the existence check below treats a fresh database
// (already at the final shape) and a migrated one identically.
var ownerFK = map[string]string{
	"accounts":            "accounts_owner_user_id_fkey",
	"transactions":        "transactions_owner_user_id_fkey",
	"salary_records":      "salary_records_owner_user_id_fkey",
	"debt_payments":       "debt_payments_owner_user_id_fkey",
	"bonus_events":        "bonus_events_owner_user_id_fkey",
	"equity_grants":       "equity_grants_owner_user_id_fkey",
	"retirement_limits":   "retirement_limits_owner_user_id_fkey",
	"snapshot_aggregates": "snapshot_aggregates_owner_user_id_fkey",
}

// Migrate converges an existing database onto the final personas->users
// schema: it drops the legacy `owner` string column from every owner-bearing
// table, rebuilds the affected unique constraints on owner_user_id, adds the
// owner_user_id foreign keys, and drops the personas table.
//
// It runs on every startup after the users table exists and is fully
// idempotent — every step is guarded by a presence check, so it is a no-op
// against a fresh database (where schema.sql already created the final shape)
// and safe to re-run.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if err := dropOwnerDependentObjects(ctx, pool); err != nil {
		return err
	}
	for _, table := range ownerTables {
		if err := dropOwnerColumn(ctx, pool, table); err != nil {
			return err
		}
	}
	if err := rebuildOwnerUserIDConstraints(ctx, pool); err != nil {
		return err
	}
	for _, table := range ownerTables {
		if err := addOwnerUserIDForeignKey(ctx, pool, table); err != nil {
			return err
		}
	}
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS personas`); err != nil {
		return fmt.Errorf("drop personas table: %w", err)
	}
	if err := addAppConfigWithdrawalRate(ctx, pool); err != nil {
		return err
	}
	if err := createRecurringTransactionsTable(ctx, pool); err != nil {
		return err
	}
	if err := createHoldingsTables(ctx, pool); err != nil {
		return err
	}
	return nil
}

// createHoldingsTables creates the securities / lots / price_quotes tables
// (issue #400) on existing databases. Fresh installs get them from schema.sql.
// All statements are idempotent.
func createHoldingsTables(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS securities (
			id serial PRIMARY KEY,
			symbol varchar(32) NOT NULL,
			isin varchar(12),
			name varchar(200) NOT NULL,
			asset_type varchar(16) NOT NULL,
			currency varchar(3) NOT NULL DEFAULT 'PLN',
			created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_securities_symbol ON securities (symbol)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_securities_isin ON securities (isin) WHERE isin IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS lots (
			id serial PRIMARY KEY,
			account_id integer NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			security_id integer NOT NULL REFERENCES securities(id) ON DELETE RESTRICT,
			side varchar(8) NOT NULL,
			quantity numeric(20,8) NOT NULL,
			price numeric(20,6) NOT NULL,
			fee numeric(15,2) NOT NULL DEFAULT 0,
			date ` + "DATE" + ` NOT NULL,
			created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
		)`,
		`CREATE INDEX IF NOT EXISTS ix_lots_security_date ON lots (security_id, date)`,
		`CREATE INDEX IF NOT EXISTS ix_lots_account ON lots (account_id)`,
		`CREATE TABLE IF NOT EXISTS price_quotes (
			id serial PRIMARY KEY,
			security_id integer NOT NULL REFERENCES securities(id) ON DELETE CASCADE,
			date ` + "DATE" + ` NOT NULL,
			price numeric(20,6) NOT NULL,
			source varchar(40) NOT NULL DEFAULT 'manual',
			created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_price_quotes_security_date ON price_quotes (security_id, date)`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("create holdings tables: %w", err)
		}
	}
	return nil
}

// createRecurringTransactionsTable creates the recurring_transactions table
// for issue #384 on existing databases. New installs get it from schema.sql.
// Idempotent via IF NOT EXISTS.
func createRecurringTransactionsTable(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS recurring_transactions (
			id serial PRIMARY KEY,
			account_id integer NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			amount numeric(15,2) NOT NULL,
			owner_user_id integer REFERENCES users(id) ON DELETE RESTRICT,
			transaction_type varchar(20),
			category varchar(50),
			description varchar(200) NOT NULL DEFAULT '',
			frequency varchar(16) NOT NULL,
			day_of_month integer,
			start_date date NOT NULL,
			end_date date,
			active boolean NOT NULL DEFAULT true,
			skipped_dates date[] NOT NULL DEFAULT ARRAY[]::date[],
			last_run_date date,
			created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
			updated_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
		)`,
		`ALTER TABLE recurring_transactions ADD COLUMN IF NOT EXISTS category varchar(50)`,
		`CREATE INDEX IF NOT EXISTS ix_recurring_transactions_active_account
		   ON recurring_transactions (active, account_id)`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("create recurring_transactions: %w", err)
		}
	}
	return nil
}

// addAppConfigWithdrawalRate adds the withdrawal_rate column to app_config
// for FIRE number computation (issue #376). 0.04 (4 percent) is the
// Trinity-study default; the column is created with that default so any
// existing app_config row is backfilled without a separate UPDATE.
func addAppConfigWithdrawalRate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		ALTER TABLE app_config
		ADD COLUMN IF NOT EXISTS withdrawal_rate numeric(5,4) NOT NULL DEFAULT 0.04`); err != nil {
		return fmt.Errorf("add withdrawal_rate to app_config: %w", err)
	}
	return nil
}

// dropOwnerDependentObjects removes the index and unique constraints keyed on
// the legacy `owner` column so the column can be dropped.
func dropOwnerDependentObjects(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`DROP INDEX IF EXISTS ix_accounts_owner`,
		`ALTER TABLE retirement_limits DROP CONSTRAINT IF EXISTS uq_year_wrapper_owner`,
		`ALTER TABLE snapshot_aggregates DROP CONSTRAINT IF EXISTS uix_snapshot_agg_snapshot_owner`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("drop owner-dependent object: %w", err)
		}
	}
	return nil
}

// dropOwnerColumn runs a final owner->owner_user_id backfill while the legacy
// column still exists, then drops it. The backfill is belt-and-suspenders for
// any row missed by the phase-B sync; it must run before the DROP COLUMN.
func dropOwnerColumn(ctx context.Context, pool *pgxpool.Pool, table string) error {
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = $1
			  AND column_name = 'owner')`,
		table).Scan(&exists); err != nil {
		return fmt.Errorf("check owner column on %s: %w", table, err)
	}
	if !exists {
		return nil
	}
	// Guard against a database that still has `owner` but never reached
	// phase B — the backfill below would otherwise fail on a missing column.
	if _, err := pool.Exec(ctx,
		`ALTER TABLE `+table+` ADD COLUMN IF NOT EXISTS owner_user_id integer`); err != nil {
		return fmt.Errorf("ensure owner_user_id on %s: %w", table, err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE `+table+` AS t
		SET owner_user_id = u.id
		FROM users u
		WHERE t.owner = u.name AND t.owner_user_id IS DISTINCT FROM u.id`); err != nil {
		return fmt.Errorf("final owner_user_id backfill on %s: %w", table, err)
	}
	if _, err := pool.Exec(ctx,
		`ALTER TABLE `+table+` DROP COLUMN IF EXISTS owner`); err != nil {
		return fmt.Errorf("drop owner column on %s: %w", table, err)
	}
	return nil
}

// rebuildOwnerUserIDConstraints adds the unique constraints keyed on
// owner_user_id, only when missing. NULLS NOT DISTINCT (PostgreSQL 18) keeps
// the jointly-owned (NULL owner_user_id) bucket unique.
func rebuildOwnerUserIDConstraints(ctx context.Context, pool *pgxpool.Pool) error {
	constraints := []struct {
		name, stmt string
	}{
		{
			"uq_year_wrapper_owner",
			`ALTER TABLE retirement_limits
			 ADD CONSTRAINT uq_year_wrapper_owner
			 UNIQUE NULLS NOT DISTINCT (year, account_wrapper, owner_user_id)`,
		},
		{
			"uix_snapshot_agg_snapshot_owner",
			`ALTER TABLE snapshot_aggregates
			 ADD CONSTRAINT uix_snapshot_agg_snapshot_owner
			 UNIQUE NULLS NOT DISTINCT (snapshot_id, owner_user_id)`,
		},
	}
	for _, c := range constraints {
		present, err := constraintExists(ctx, pool, c.name)
		if err != nil {
			return err
		}
		if present {
			continue
		}
		if _, err := pool.Exec(ctx, c.stmt); err != nil {
			return fmt.Errorf("add constraint %s: %w", c.name, err)
		}
	}
	return nil
}

// addOwnerUserIDForeignKey adds the owner_user_id->users(id) foreign key for a
// table, only when it is missing.
func addOwnerUserIDForeignKey(ctx context.Context, pool *pgxpool.Pool, table string) error {
	name := ownerFK[table]
	present, err := constraintExists(ctx, pool, name)
	if err != nil {
		return err
	}
	if present {
		return nil
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE `+table+`
		ADD CONSTRAINT `+name+`
		FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT`); err != nil {
		return fmt.Errorf("add foreign key %s: %w", name, err)
	}
	return nil
}

// constraintExists reports whether a named constraint is present in the public
// schema.
func constraintExists(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint c
			JOIN pg_namespace n ON n.oid = c.connamespace
			WHERE n.nspname = 'public' AND c.conname = $1)`,
		name).Scan(&exists); err != nil {
		return false, fmt.Errorf("check constraint %s: %w", name, err)
	}
	return exists, nil
}

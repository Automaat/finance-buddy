package db

import (
	"context"
	"fmt"
	"strings"

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

const migrationSQLSnippetMaxLen = 96

type migrationFunc func(context.Context, *pgxpool.Pool) error

type columnDef struct {
	name       string
	definition string
}

var schemaExtensionMigrations = []migrationFunc{
	addAppConfigWithdrawalRate,
	addAppConfigCoastFIRE,
	addAppConfigBaristaFIRE,
	addAppConfigFIREBands,
	addAppConfigMonthlySavings,
	createRecurringTransactionsTable,
	createHoldingsTables,
	createSimulationScenariosTable,
	addAccountsExcludedFromFire,
	addAccountsInterestRatePct,
	dropAppConfigLegacyPPKRates,
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
	for _, migrate := range schemaExtensionMigrations {
		if err := migrate(ctx, pool); err != nil {
			return err
		}
	}
	return nil
}

func execMigrationSQL(ctx context.Context, pool *pgxpool.Pool, label string, stmts ...string) error {
	for idx, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("%s statement %d %q: %w",
				label, idx+1, migrationSQLSnippet(stmt), err)
		}
	}
	return nil
}

func migrationSQLSnippet(stmt string) string {
	snippet := strings.Join(strings.Fields(stmt), " ")
	if snippet == "" {
		return "<empty>"
	}
	if len(snippet) <= migrationSQLSnippetMaxLen {
		return snippet
	}
	return snippet[:migrationSQLSnippetMaxLen] + "..."
}

func addColumnsIfMissing(
	ctx context.Context,
	pool *pgxpool.Pool,
	label string,
	table string,
	columns ...columnDef,
) error {
	stmts := make([]string, 0, len(columns))
	for _, col := range columns {
		stmts = append(stmts, addColumnIfMissingSQL(table, col))
	}
	return execMigrationSQL(ctx, pool, label, stmts...)
}

func addColumnIfMissingSQL(table string, col columnDef) string {
	return fmt.Sprintf("ALTER TABLE %s\nADD COLUMN IF NOT EXISTS %s %s",
		table, col.name, col.definition)
}

// addAccountsInterestRatePct adds the optional nominal-yield field for
// interest-bearing accounts (issue #573). Nullable on purpose — non-cash
// accounts and accounts whose rate the user hasn't recorded leave it NULL,
// in which case the real-yield widget hides for that row.
func addAccountsInterestRatePct(ctx context.Context, pool *pgxpool.Pool) error {
	return addColumnsIfMissing(ctx, pool, "add interest_rate_pct to accounts", "accounts",
		columnDef{name: "interest_rate_pct", definition: "numeric(6,4)"},
	)
}

// addAccountsExcludedFromFire adds the per-account opt-out flag for FIRE
// math. A primary residence ("lived-in flat") inflates net worth without
// ever being drawn down for retirement income, so counting it toward the
// FIRE number distorts every downstream metric. Default false preserves
// current behavior for every existing account; the user opts in per
// account from the edit modal.
func addAccountsExcludedFromFire(ctx context.Context, pool *pgxpool.Pool) error {
	return addColumnsIfMissing(ctx, pool, "add excluded_from_fire to accounts", "accounts",
		columnDef{name: "excluded_from_fire", definition: "boolean NOT NULL DEFAULT false"},
	)
}

// createSimulationScenariosTable creates the simulation_scenarios table for
// issue #547 on existing databases. New installs get it from schema.sql.
// inputs_json is opaque to the backend — the simulations form serializes
// its current state into it, so adding fields to the form doesn't require
// a schema change.
func createSimulationScenariosTable(ctx context.Context, pool *pgxpool.Pool) error {
	return execMigrationSQL(ctx, pool, "create simulation_scenarios",
		`CREATE TABLE IF NOT EXISTS simulation_scenarios (
			id serial PRIMARY KEY,
			name varchar(200) NOT NULL,
			kind varchar(40) NOT NULL,
			inputs_json jsonb NOT NULL,
			created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
			updated_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
		)`,
		`CREATE INDEX IF NOT EXISTS ix_simulation_scenarios_kind_updated_at
			ON simulation_scenarios (kind, updated_at DESC)`,
	)
}

// addAppConfigMonthlySavings adds the monthly_savings input feeding the
// projected-FI-date metric (issue #551). Nullable — when unset, the FI
// projection tile shows an empty state asking the user to configure it.
func addAppConfigMonthlySavings(ctx context.Context, pool *pgxpool.Pool) error {
	return addColumnsIfMissing(ctx, pool, "add monthly_savings to app_config", "app_config",
		columnDef{name: "monthly_savings", definition: "numeric(15,2)"},
	)
}

// addAppConfigFIREBands adds the Lean and Fat FIRE monthly-expense bands
// to app_config (issue #550). Both nullable — when unset the band tile is
// hidden and only the existing Base FIRE number (monthly_expenses) shows.
func addAppConfigFIREBands(ctx context.Context, pool *pgxpool.Pool) error {
	return addColumnsIfMissing(ctx, pool, "add fire bands to app_config", "app_config",
		columnDef{name: "lean_monthly_expenses", definition: "numeric(15,2)"},
		columnDef{name: "fat_monthly_expenses", definition: "numeric(15,2)"},
	)
}

// addAppConfigBaristaFIRE adds the Barista FIRE input to app_config (issue
// #552): an optional `barista_monthly_income`. Nullable on purpose — when
// unset the Barista FIRE tile is hidden, matching the Coast FIRE convention.
func addAppConfigBaristaFIRE(ctx context.Context, pool *pgxpool.Pool) error {
	return addColumnsIfMissing(ctx, pool, "add barista_monthly_income to app_config", "app_config",
		columnDef{name: "barista_monthly_income", definition: "numeric(15,2)"},
	)
}

// addAppConfigCoastFIRE adds the Coast FIRE inputs to app_config (issue #548):
// `coast_fire_target_age` (optional — Coast FIRE tile hides when nil) and
// `expected_return_rate` (defaults to 0.07, a conservative real-return
// assumption matching the retirement-savings projection on the settings page).
func addAppConfigCoastFIRE(ctx context.Context, pool *pgxpool.Pool) error {
	return addColumnsIfMissing(ctx, pool, "add coast fire columns to app_config", "app_config",
		columnDef{name: "coast_fire_target_age", definition: "integer"},
		columnDef{name: "expected_return_rate", definition: "numeric(5,4) NOT NULL DEFAULT 0.07"},
	)
}

// createHoldingsTables creates the securities / lots / price_quotes tables
// (issue #400) on existing databases. Fresh installs get them from schema.sql.
// All statements are idempotent.
func createHoldingsTables(ctx context.Context, pool *pgxpool.Pool) error {
	return execMigrationSQL(ctx, pool, "create holdings tables",
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
			date DATE NOT NULL,
			created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
		)`,
		`CREATE INDEX IF NOT EXISTS ix_lots_security_date ON lots (security_id, date)`,
		`CREATE INDEX IF NOT EXISTS ix_lots_account ON lots (account_id)`,
		`CREATE TABLE IF NOT EXISTS price_quotes (
			id serial PRIMARY KEY,
			security_id integer NOT NULL REFERENCES securities(id) ON DELETE CASCADE,
			date DATE NOT NULL,
			price numeric(20,6) NOT NULL,
			source varchar(40) NOT NULL DEFAULT 'manual',
			created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_price_quotes_security_date ON price_quotes (security_id, date)`,
	)
}

// createRecurringTransactionsTable creates the recurring_transactions table
// for issue #384 on existing databases. New installs get it from schema.sql.
// Idempotent via IF NOT EXISTS.
func createRecurringTransactionsTable(ctx context.Context, pool *pgxpool.Pool) error {
	return execMigrationSQL(ctx, pool, "create recurring_transactions",
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
	)
}

// dropAppConfigLegacyPPKRates drops four NOT NULL Python-era columns
// (ppk_{employee,employer}_rate_{marcin,ewa}) from app_config. PPK rates now
// live per-user in the users table, but these columns were never dropped on
// existing databases — so every PUT /api/config fails with a NOT NULL
// violation because the Go INSERT doesn't list them.
func dropAppConfigLegacyPPKRates(ctx context.Context, pool *pgxpool.Pool) error {
	return execMigrationSQL(ctx, pool, "drop legacy ppk rate columns from app_config",
		`ALTER TABLE app_config
			DROP COLUMN IF EXISTS ppk_employee_rate_marcin,
			DROP COLUMN IF EXISTS ppk_employer_rate_marcin,
			DROP COLUMN IF EXISTS ppk_employee_rate_ewa,
			DROP COLUMN IF EXISTS ppk_employer_rate_ewa`,
	)
}

// addAppConfigWithdrawalRate adds the withdrawal_rate column to app_config
// for FIRE number computation (issue #376). 0.04 (4 percent) is the
// Trinity-study default; the column is created with that default so any
// existing app_config row is backfilled without a separate UPDATE.
func addAppConfigWithdrawalRate(ctx context.Context, pool *pgxpool.Pool) error {
	return addColumnsIfMissing(ctx, pool, "add withdrawal_rate to app_config", "app_config",
		columnDef{name: "withdrawal_rate", definition: "numeric(5,4) NOT NULL DEFAULT 0.04"},
	)
}

// dropOwnerDependentObjects removes the index and unique constraints keyed on
// the legacy `owner` column so the column can be dropped.
func dropOwnerDependentObjects(ctx context.Context, pool *pgxpool.Pool) error {
	return execMigrationSQL(ctx, pool, "drop owner-dependent object",
		`DROP INDEX IF EXISTS ix_accounts_owner`,
		`ALTER TABLE retirement_limits DROP CONSTRAINT IF EXISTS uq_year_wrapper_owner`,
		`ALTER TABLE snapshot_aggregates DROP CONSTRAINT IF EXISTS uix_snapshot_agg_snapshot_owner`,
	)
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

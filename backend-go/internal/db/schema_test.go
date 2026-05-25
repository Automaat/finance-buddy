package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSchemaSQLEmbedded(t *testing.T) {
	if len(schemaSQL) == 0 {
		t.Fatal("schemaSQL is empty — embed directive broken")
	}
}

func TestSchemaSQLContainsCoreTables(t *testing.T) {
	required := []string{
		"CREATE TABLE",
		"accounts",
		"users",
		"snapshots",
		"transactions",
		"snapshot_aggregates",
	}
	for _, frag := range required {
		if !strings.Contains(schemaSQL, frag) {
			t.Errorf("schema.sql missing fragment %q", frag)
		}
	}
}

// integrationPool returns a real pool when TEST_DATABASE_URL is set,
// otherwise the calling test is skipped. Each call wipes the `public`
// schema, so callers MUST NOT run in parallel with each other or with
// any other test that shares the DSN. The bb-tests-go CI job is the
// authoritative integration oracle for schema bootstrap; this hook lets
// developers exercise the same code path with `go test` against a local
// throwaway Postgres.
func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	pool, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to TEST_DATABASE_URL: %v", err)
	}
	t.Cleanup(pool.Close)
	for _, stmt := range []string{
		"DROP SCHEMA public CASCADE",
		"CREATE SCHEMA public",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("reset test schema (%s): %v", stmt, err)
		}
	}
	return pool
}

func TestApplySchemaCreatesAccountsTable(t *testing.T) {
	pool := integrationPool(t)
	ctx := t.Context()
	if err := ApplySchema(ctx, pool); err != nil {
		t.Fatalf("ApplySchema on empty db: %v", err)
	}
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT to_regclass('public.accounts') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatalf("regclass check: %v", err)
	}
	if !exists {
		t.Fatal("accounts table not created")
	}
}

// TestApplySchemaIsIdempotent proves the presence-check short-circuit by
// seeding a sentinel row after the first apply: schema.sql starts with
// `CREATE TABLE`, so a non-short-circuited second apply would error out on
// the already-present `accounts` table. Reaching the sentinel lookup proves
// the no-op branch fired; the row check guards against future schema.sql
// rewrites that prepend DROPs or otherwise survive a re-run.
func TestApplySchemaIsIdempotent(t *testing.T) {
	pool := integrationPool(t)
	ctx := t.Context()
	if err := ApplySchema(ctx, pool); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (username, password_hash) VALUES ('sentinel', 'x')`); err != nil {
		t.Fatalf("seed sentinel: %v", err)
	}
	if err := ApplySchema(ctx, pool); err != nil {
		t.Fatalf("second apply (should no-op): %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM users WHERE username = 'sentinel'`).Scan(&count); err != nil {
		t.Fatalf("sentinel lookup: %v", err)
	}
	if count != 1 {
		t.Fatalf("sentinel row wiped — second ApplySchema did not no-op (count=%d)", count)
	}
}

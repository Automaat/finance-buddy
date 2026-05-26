package db

import (
	"slices"
	"strings"
	"testing"
)

func TestOwnerTablesHaveForeignKeyNames(t *testing.T) {
	for _, table := range ownerTables {
		name, ok := ownerFK[table]
		if !ok {
			t.Errorf("ownerFK missing entry for table %q", table)
			continue
		}
		if name == "" {
			t.Errorf("ownerFK[%q] is empty", table)
		}
	}
	if len(ownerTables) != len(ownerFK) {
		t.Errorf("ownerTables (%d) and ownerFK (%d) length mismatch",
			len(ownerTables), len(ownerFK))
	}
	for tbl := range ownerFK {
		if !slices.Contains(ownerTables, tbl) {
			t.Errorf("ownerFK has %q but ownerTables does not", tbl)
		}
	}
}

func TestMigrationSQLSnippet(t *testing.T) {
	t.Run("compacts whitespace", func(t *testing.T) {
		got := migrationSQLSnippet(`
			ALTER TABLE app_config
			ADD COLUMN IF NOT EXISTS monthly_savings numeric(15,2)
		`)
		want := "ALTER TABLE app_config ADD COLUMN IF NOT EXISTS monthly_savings numeric(15,2)"
		if got != want {
			t.Fatalf("snippet = %q, want %q", got, want)
		}
	})

	t.Run("marks empty statement", func(t *testing.T) {
		got := migrationSQLSnippet(" \n\t ")
		if got != "<empty>" {
			t.Fatalf("snippet = %q, want <empty>", got)
		}
	})

	t.Run("truncates long statement", func(t *testing.T) {
		got := migrationSQLSnippet(strings.Repeat("x", migrationSQLSnippetMaxLen+1))
		want := strings.Repeat("x", migrationSQLSnippetMaxLen) + "..."
		if got != want {
			t.Fatalf("snippet = %q, want %q", got, want)
		}
	})
}

func TestAddColumnIfMissingSQL(t *testing.T) {
	got := addColumnIfMissingSQL("app_config", columnDef{
		name:       "expected_return_rate",
		definition: "numeric(5,4) NOT NULL DEFAULT 0.07",
	})
	want := "ALTER TABLE app_config\nADD COLUMN IF NOT EXISTS expected_return_rate numeric(5,4) NOT NULL DEFAULT 0.07"
	if got != want {
		t.Fatalf("statement = %q, want %q", got, want)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	pool := integrationPool(t)
	ctx := t.Context()
	if err := ApplySchema(ctx, pool); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("second migrate (should no-op): %v", err)
	}
}

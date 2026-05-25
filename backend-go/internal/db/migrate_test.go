package db

import (
	"context"
	"slices"
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

func TestMigrateIsIdempotent(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
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

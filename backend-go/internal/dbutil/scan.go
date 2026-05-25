// Package dbutil holds small pgx helpers shared across stores. The goal
// is not abstraction — it's killing the 4-line "if ErrNoRows → sentinel /
// else wrap" pattern that repeats in every Get/scan call site.
package dbutil

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// MapErr converts a pgx error into the canonical pair every store uses:
// pgx.ErrNoRows → notFound (the domain's sentinel), everything else →
// wrapPrefix-prefixed wrap. Returns nil when err is nil so the call can
// stand at the end of a Scan branch unconditionally.
func MapErr(err, notFound error, wrapPrefix string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return fmt.Errorf("%s: %w", wrapPrefix, err)
}

// CollectRows scans every row and preserves the store-specific scan/iterate
// error prefixes used by existing handlers.
func CollectRows[T any](rows pgx.Rows, scan func(pgx.Row) (T, error), scanPrefix, iterPrefix string) ([]T, error) {
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", scanPrefix, err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", iterPrefix, err)
	}
	return out, nil
}

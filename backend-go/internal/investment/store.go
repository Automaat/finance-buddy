// Package investment implements /api/investment — stock + bond ROI
// aggregates computed across snapshots.
package investment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Store is the persistence boundary.
type Store struct{ pool *pgxpool.Pool }

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// CategoryStats is the computed aggregate for one investment category.
type CategoryStats struct {
	TotalValue       decimal.Decimal
	TotalContributed decimal.Decimal
	HasAccounts      bool
}

// StatsForCategory replicates backend/app/services/investment.get_category_stats:
//   - total_contributed: sum of active transactions for the category's
//     active accounts, capped at the latest snapshot date when snapshots
//     exist (uncapped otherwise).
//   - total_value: sum of snapshot_values at that latest snapshot date.
//
// Single round-trip: the cat_accounts CTE drives the has-accounts flag, the
// transactions sum and the value sum. HasAccounts=false makes the handler
// emit the all-zero response Python produces for an empty category.
//
// The "uncapped when no snapshot" behavior is the `latest.d IS NULL OR
// t.date <= latest.d` predicate — when no snapshot row exists, latest.d is
// NULL and the date bound drops out.
func (s *Store) StatsForCategory(ctx context.Context, category string) (CategoryStats, error) {
	row := s.pool.QueryRow(ctx, `
		WITH cat_accounts AS (
			SELECT id FROM accounts WHERE category = $1 AND is_active = true
		),
		latest AS (
			SELECT MAX(s.date) AS d
			FROM snapshots s
			JOIN snapshot_values sv ON sv.snapshot_id = s.id
			WHERE sv.account_id IN (SELECT id FROM cat_accounts)
		)
		SELECT
			EXISTS (SELECT 1 FROM cat_accounts) AS has_accounts,
			(SELECT COALESCE(SUM(t.amount), 0)
			 FROM transactions t, latest
			 WHERE t.account_id IN (SELECT id FROM cat_accounts)
			   AND t.is_active = true
			   AND (latest.d IS NULL OR t.date <= latest.d)
			) AS total_contributed,
			(SELECT COALESCE(SUM(sv.value), 0)
			 FROM snapshot_values sv
			 JOIN snapshots s ON s.id = sv.snapshot_id, latest
			 WHERE sv.account_id IN (SELECT id FROM cat_accounts)
			   AND latest.d IS NOT NULL
			   AND s.date = latest.d
			) AS total_value`,
		category,
	)
	var stats CategoryStats
	if err := row.Scan(&stats.HasAccounts, &stats.TotalContributed, &stats.TotalValue); err != nil {
		return CategoryStats{}, fmt.Errorf("category stats: %w", err)
	}
	return stats, nil
}

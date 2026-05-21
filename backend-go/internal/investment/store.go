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
// Returns HasAccounts=false when the category has no active accounts so the
// handler can emit the all-zero response Python produces.
func (s *Store) StatsForCategory(ctx context.Context, category string) (CategoryStats, error) {
	var accountCount int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM accounts WHERE category = $1 AND is_active = true`,
		category,
	).Scan(&accountCount); err != nil {
		return CategoryStats{}, fmt.Errorf("count category accounts: %w", err)
	}
	if accountCount == 0 {
		return CategoryStats{HasAccounts: false}, nil
	}

	// total_contributed + total_value in a single query. The latest snapshot
	// date is the MAX(s.date) over snapshots that hold a value for any active
	// account in this category; when no snapshot exists the date subquery is
	// NULL and the transactions sum is left uncapped (COALESCE on the bound).
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
	stats.HasAccounts = true
	if err := row.Scan(&stats.TotalContributed, &stats.TotalValue); err != nil {
		return CategoryStats{}, fmt.Errorf("category stats: %w", err)
	}
	return stats, nil
}

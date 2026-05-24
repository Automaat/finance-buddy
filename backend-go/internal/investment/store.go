// Package investment implements /api/investment — stock + bond ROI
// aggregates computed across snapshots.
package investment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

// TransactionDated is a dated transaction in the cash-flow series used by the
// returns endpoint. amount preserves the sign convention from the table
// (positive = deposit, negative = withdrawal).
type TransactionDated struct {
	Date   time.Time
	Amount decimal.Decimal
}

// ScopeFilter narrows the account set the returns endpoint operates over.
// Exactly one of AccountID / Category / Wrapper / All should be set; All=true
// means the whole household.
type ScopeFilter struct {
	AccountID *int
	Category  *string
	Wrapper   *string
	All       bool
}

// PeriodWindow returns (since, asOf). A nil since means "since the first cash
// flow in scope" (= "All").
type PeriodWindow struct {
	Since *time.Time
	AsOf  time.Time
}

// ScopedFlows pulls all active transactions in scope between `since` (open)
// and `asOf` (inclusive). Returns chronologically sorted.
func (s *Store) ScopedFlows(ctx context.Context, scope ScopeFilter, w PeriodWindow) ([]TransactionDated, error) {
	args := []any{w.AsOf}
	clauses := []string{"a.is_active = true", "t.is_active = true", "t.date <= $1"}
	if w.Since != nil {
		args = append(args, *w.Since)
		clauses = append(clauses, fmt.Sprintf("t.date >= $%d", len(args)))
	}
	switch {
	case scope.AccountID != nil:
		args = append(args, *scope.AccountID)
		clauses = append(clauses, fmt.Sprintf("a.id = $%d", len(args)))
	case scope.Category != nil:
		args = append(args, *scope.Category)
		clauses = append(clauses, fmt.Sprintf("a.category = $%d", len(args)))
	case scope.Wrapper != nil:
		args = append(args, *scope.Wrapper)
		clauses = append(clauses, fmt.Sprintf("a.account_wrapper = $%d", len(args)))
	}
	where := ""
	for i, c := range clauses {
		if i == 0 {
			where = "WHERE " + c
		} else {
			where += " AND " + c
		}
	}
	q := `
		SELECT t.date, t.amount
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		` + where + `
		ORDER BY t.date ASC, t.id ASC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("scoped flows: %w", err)
	}
	defer rows.Close()
	out := []TransactionDated{}
	for rows.Next() {
		var td TransactionDated
		if err := rows.Scan(&td.Date, &td.Amount); err != nil {
			return nil, fmt.Errorf("scan flow: %w", err)
		}
		out = append(out, td)
	}
	return out, rows.Err()
}

// LatestValueInScope returns the sum of snapshot_values at the latest
// snapshot date on or before `asOf` for accounts matching scope. Returns
// (zero, false) when no qualifying snapshot exists.
func (s *Store) LatestValueInScope(ctx context.Context, scope ScopeFilter, asOf time.Time) (decimal.Decimal, time.Time, bool, error) {
	args := []any{asOf}
	scopeJoin := ""
	switch {
	case scope.AccountID != nil:
		args = append(args, *scope.AccountID)
		scopeJoin = fmt.Sprintf("AND sv.account_id = $%d", len(args))
	case scope.Category != nil:
		args = append(args, *scope.Category)
		scopeJoin = fmt.Sprintf("AND a.category = $%d", len(args))
	case scope.Wrapper != nil:
		args = append(args, *scope.Wrapper)
		scopeJoin = fmt.Sprintf("AND a.account_wrapper = $%d", len(args))
	}
	row := s.pool.QueryRow(ctx, `
		WITH latest AS (
			SELECT MAX(s.date) AS d
			FROM snapshots s
			JOIN snapshot_values sv ON sv.snapshot_id = s.id
			JOIN accounts a ON a.id = sv.account_id
			WHERE s.date <= $1
			  AND a.is_active = true
			  `+scopeJoin+`
		)
		SELECT COALESCE(SUM(sv.value), 0), latest.d
		FROM snapshot_values sv
		JOIN snapshots s ON s.id = sv.snapshot_id
		JOIN accounts a ON a.id = sv.account_id, latest
		WHERE s.date = latest.d
		  AND a.is_active = true
		  `+scopeJoin+`
		GROUP BY latest.d`,
		args...)
	var value decimal.Decimal
	var date time.Time
	switch err := row.Scan(&value, &date); {
	case errors.Is(err, pgx.ErrNoRows):
		return decimal.Zero, time.Time{}, false, nil
	case err != nil:
		return decimal.Zero, time.Time{}, false, fmt.Errorf("latest value in scope: %w", err)
	}
	return value, date, true, nil
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

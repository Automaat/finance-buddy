// Package dashboard implements GET /api/dashboard — the aggregate-backed
// hot path with a raw-fallback, ported from backend/app/services/dashboard/.
package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Store holds every DB read the dashboard needs.
type Store struct{ pool *pgxpool.Pool }

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// AggregateRow mirrors a snapshot_aggregates row. OwnerUserID is nil for
// the jointly-owned ("Shared") bucket.
type AggregateRow struct {
	SnapshotID       int
	Month            time.Time
	OwnerUserID      *int
	TotalAssets      decimal.Decimal
	TotalLiabilities decimal.Decimal
	NetWorth         decimal.Decimal
	Allocation       []AllocationJSONItem
}

// AllocationJSONItem is one entry of allocation_json.
type AllocationJSONItem struct {
	Category string  `json:"category"`
	Value    float64 `json:"value"`
}

// Account mirrors the columns the dashboard reads from accounts.
// OwnerUserID is nil for jointly-owned accounts. IsActive lets the
// dashboard preserve historical snapshot values from soft-deleted
// accounts while keeping "current state" reads filtered.
type Account struct {
	ID             int
	Name           string
	Type           string
	Category       string
	OwnerUserID    *int
	AccountWrapper *string
	Purpose        string
	SquareMeters   *decimal.Decimal
	IsActive       bool
}

// SnapshotValue is one snapshot_values row.
type SnapshotValue struct {
	SnapshotID int
	AccountID  *int
	AssetID    *int
	Value      decimal.Decimal
}

// Transaction is the subset of transactions the dashboard reads.
type Transaction struct {
	AccountID       int
	Amount          decimal.Decimal
	Date            time.Time
	TransactionType *string
}

// AppConfig mirrors the app_config row.
type AppConfig struct {
	MonthlyExpenses        decimal.Decimal
	MonthlyMortgagePayment decimal.Decimal
	AllocationStocks       int
	AllocationBonds        int
	AllocationGold         int
	WithdrawalRate         decimal.Decimal
	BirthDate              time.Time
	CoastFIRETargetAge     *int
	ExpectedReturnRate     decimal.Decimal
}

// SnapshotMeta is id+date.
type SnapshotMeta struct {
	ID   int
	Date time.Time
}

// LoadAggregateRows returns every snapshot_aggregates row.
func (s *Store) LoadAggregateRows(ctx context.Context) ([]AggregateRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT snapshot_id, month, owner_user_id, total_assets, total_liabilities,
		       net_worth, allocation_json
		FROM snapshot_aggregates`,
	)
	if err != nil {
		return nil, fmt.Errorf("load aggregates: %w", err)
	}
	defer rows.Close()
	out := []AggregateRow{}
	for rows.Next() {
		var r AggregateRow
		var allocJSON []byte
		if err := rows.Scan(
			&r.SnapshotID, &r.Month, &r.OwnerUserID,
			&r.TotalAssets, &r.TotalLiabilities, &r.NetWorth, &allocJSON,
		); err != nil {
			return nil, fmt.Errorf("scan aggregate: %w", err)
		}
		if len(allocJSON) > 0 && string(allocJSON) != "null" {
			if err := json.Unmarshal(allocJSON, &r.Allocation); err != nil {
				return nil, fmt.Errorf("decode allocation_json: %w", err)
			}
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate aggregates: %w", err)
	}
	return out, nil
}

// SnapshotDates returns a map of snapshot id -> date for the given ids.
func (s *Store) SnapshotDates(ctx context.Context, ids []int) (map[int]time.Time, error) {
	out := map[int]time.Time{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT id, date FROM snapshots WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("snapshot dates: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var d time.Time
		if err := rows.Scan(&id, &d); err != nil {
			return nil, fmt.Errorf("scan snapshot date: %w", err)
		}
		out[id] = d
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot dates: %w", err)
	}
	return out, nil
}

// AllSnapshots returns id+date for every snapshot ordered by date.
func (s *Store) AllSnapshots(ctx context.Context) ([]SnapshotMeta, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, date FROM snapshots ORDER BY date`)
	if err != nil {
		return nil, fmt.Errorf("all snapshots: %w", err)
	}
	defer rows.Close()
	out := []SnapshotMeta{}
	for rows.Next() {
		var m SnapshotMeta
		if err := rows.Scan(&m.ID, &m.Date); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return out, nil
}

// AllAccounts returns every account (active + soft-deleted). Historical
// snapshot values that reference a soft-deleted account need the original
// type/category to keep their contribution to net worth correct; callers
// that want only the active set filter on Account.IsActive.
func (s *Store) AllAccounts(ctx context.Context) ([]Account, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, type, category, owner_user_id, account_wrapper,
		       purpose, square_meters, is_active
		FROM accounts`,
	)
	if err != nil {
		return nil, fmt.Errorf("all accounts: %w", err)
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		var a Account
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Type, &a.Category, &a.OwnerUserID,
			&a.AccountWrapper, &a.Purpose, &a.SquareMeters, &a.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}
	return out, nil
}

// AllAssetIDs returns the set of asset ids (active + soft-deleted), used
// to mark historical snapshot rows that point at the assets table even
// when the underlying asset is soft-deleted.
func (s *Store) AllAssetIDs(ctx context.Context) (map[int]struct{}, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM assets`)
	if err != nil {
		return nil, fmt.Errorf("all assets: %w", err)
	}
	defer rows.Close()
	out := map[int]struct{}{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan asset id: %w", err)
		}
		out[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate asset ids: %w", err)
	}
	return out, nil
}

// SnapshotValuesFor returns the snapshot_values for one snapshot.
func (s *Store) SnapshotValuesFor(ctx context.Context, snapshotID int) ([]SnapshotValue, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT snapshot_id, account_id, asset_id, value
		FROM snapshot_values WHERE snapshot_id = $1`,
		snapshotID,
	)
	if err != nil {
		return nil, fmt.Errorf("snapshot values: %w", err)
	}
	defer rows.Close()
	return collectSnapshotValues(rows)
}

// AllSnapshotValues returns every snapshot_values row.
func (s *Store) AllSnapshotValues(ctx context.Context) ([]SnapshotValue, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT snapshot_id, account_id, asset_id, value FROM snapshot_values`,
	)
	if err != nil {
		return nil, fmt.Errorf("snapshot values: %w", err)
	}
	defer rows.Close()
	return collectSnapshotValues(rows)
}

func collectSnapshotValues(rows pgx.Rows) ([]SnapshotValue, error) {
	out := []SnapshotValue{}
	for rows.Next() {
		var v SnapshotValue
		if err := rows.Scan(&v.SnapshotID, &v.AccountID, &v.AssetID, &v.Value); err != nil {
			return nil, fmt.Errorf("scan snapshot value: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot values: %w", err)
	}
	return out, nil
}

// ActiveTransactions returns every active transaction.
func (s *Store) ActiveTransactions(ctx context.Context) ([]Transaction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT account_id, amount, date, transaction_type
		FROM transactions WHERE is_active = true`,
	)
	if err != nil {
		return nil, fmt.Errorf("active transactions: %w", err)
	}
	defer rows.Close()
	out := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.AccountID, &t.Amount, &t.Date, &t.TransactionType); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return out, nil
}

// LoadAppConfig returns the app_config id=1 row. The bool is false when no
// config row exists (a normal branch — the dashboard emits zeroed metric
// cards in that case).
func (s *Store) LoadAppConfig(ctx context.Context) (AppConfig, bool, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT monthly_expenses, monthly_mortgage_payment,
		       allocation_stocks, allocation_bonds, allocation_gold,
		       withdrawal_rate, birth_date,
		       coast_fire_target_age, expected_return_rate
		FROM app_config WHERE id = 1`,
	)
	var c AppConfig
	if err := row.Scan(
		&c.MonthlyExpenses, &c.MonthlyMortgagePayment,
		&c.AllocationStocks, &c.AllocationBonds, &c.AllocationGold,
		&c.WithdrawalRate, &c.BirthDate,
		&c.CoastFIRETargetAge, &c.ExpectedReturnRate,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AppConfig{}, false, nil
		}
		return AppConfig{}, false, fmt.Errorf("app config: %w", err)
	}
	return c, true, nil
}

// RecentSalaries returns the gross_amount of the latest n active salary
// records, most-recent first.
func (s *Store) RecentSalaries(ctx context.Context, n int) ([]decimal.Decimal, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT gross_amount FROM salary_records
		WHERE is_active = true ORDER BY date DESC LIMIT $1`,
		n,
	)
	if err != nil {
		return nil, fmt.Errorf("recent salaries: %w", err)
	}
	defer rows.Close()
	out := []decimal.Decimal{}
	for rows.Next() {
		var v decimal.Decimal
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan salary: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate salaries: %w", err)
	}
	return out, nil
}

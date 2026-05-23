package aggregates

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

// errSnapshotGone signals "snapshot row no longer exists" — callers treat
// that as a successful no-op (matches Python's `if snapshot is None: return`).
// Unexported sentinel so we don't return (nil, nil).
var errSnapshotGone = errors.New("snapshot no longer exists")

// Store does the recompute_for_snapshot[s] writeback against snapshot_aggregates.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// RecomputeForSnapshot deletes and reinserts aggregate rows for one snapshot.
// Idempotent. Runs in the caller's transaction so accounts.Update can chain
// account writes + aggregate recompute atomically.
//
// When the snapshot has already been deleted (e.g. concurrent delete + edit),
// returns nil and writes nothing.
func (s *Store) RecomputeForSnapshot(ctx context.Context, tx pgx.Tx, snapshotID int) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM snapshot_aggregates WHERE snapshot_id = $1`, snapshotID,
	); err != nil {
		return fmt.Errorf("delete snapshot_aggregates: %w", err)
	}
	snap, err := loadSnapshot(ctx, tx, snapshotID)
	if err != nil {
		if errors.Is(err, errSnapshotGone) {
			return nil
		}
		return err
	}
	values, err := loadSnapshotValues(ctx, tx, snapshotID)
	if err != nil {
		return err
	}
	accountIDs := []int{}
	assetIDs := []int{}
	for _, v := range values {
		if v.AccountID != nil {
			accountIDs = append(accountIDs, *v.AccountID)
		}
		if v.AssetID != nil {
			assetIDs = append(assetIDs, *v.AssetID)
		}
	}
	accounts, err := loadAccountsByID(ctx, tx, accountIDs)
	if err != nil {
		return err
	}
	knownAssets, err := loadAssetIDs(ctx, tx, assetIDs)
	if err != nil {
		return err
	}
	rows := ComputeAggregates(snap, values, accounts, knownAssets)
	now := time.Now().UTC()
	for _, r := range rows {
		alloc := make([]map[string]any, 0, len(r.Allocation))
		for _, a := range r.Allocation {
			f, _ := a.Value.Float64()
			alloc = append(alloc, map[string]any{"category": a.Category, "value": f})
		}
		allocJSON, err := json.Marshal(alloc)
		if err != nil {
			return fmt.Errorf("encode allocation_json: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO snapshot_aggregates (
				snapshot_id, month, owner_user_id, total_assets,
				total_liabilities, net_worth, allocation_json, computed_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8
			)`,
			r.SnapshotID, r.Month, r.OwnerUserID,
			r.TotalAssets, r.TotalLiabilities, r.NetWorth, allocJSON, now,
		); err != nil {
			return fmt.Errorf("insert snapshot_aggregates: %w", err)
		}
	}
	return nil
}

// RecomputeForSnapshots loops over a set of snapshot IDs.
func (s *Store) RecomputeForSnapshots(ctx context.Context, tx pgx.Tx, snapshotIDs []int) error {
	for _, sid := range snapshotIDs {
		if err := s.RecomputeForSnapshot(ctx, tx, sid); err != nil {
			return err
		}
	}
	return nil
}

// AffectedSnapshotIDsForAccount returns the snapshot IDs that hold a value
// for this account — accounts.Update calls this before deciding to recompute.
func (s *Store) AffectedSnapshotIDsForAccount(ctx context.Context, tx pgx.Tx, accountID int) ([]int, error) {
	rows, err := tx.Query(ctx,
		`SELECT DISTINCT snapshot_id FROM snapshot_values WHERE account_id = $1`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("affected snapshots for account: %w", err)
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var sid int
		if err := rows.Scan(&sid); err != nil {
			return nil, fmt.Errorf("scan snapshot id: %w", err)
		}
		out = append(out, sid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot ids: %w", err)
	}
	return out, nil
}

// AffectedSnapshotIDsForAsset returns the snapshot IDs that hold a value
// for this asset.
func (s *Store) AffectedSnapshotIDsForAsset(ctx context.Context, tx pgx.Tx, assetID int) ([]int, error) {
	rows, err := tx.Query(ctx,
		`SELECT DISTINCT snapshot_id FROM snapshot_values WHERE asset_id = $1`,
		assetID,
	)
	if err != nil {
		return nil, fmt.Errorf("affected snapshots for asset: %w", err)
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var sid int
		if err := rows.Scan(&sid); err != nil {
			return nil, fmt.Errorf("scan snapshot id: %w", err)
		}
		out = append(out, sid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot ids: %w", err)
	}
	return out, nil
}

func loadSnapshot(ctx context.Context, tx pgx.Tx, id int) (SnapshotInput, error) {
	row := tx.QueryRow(ctx, `SELECT id, date FROM snapshots WHERE id = $1`, id)
	var s SnapshotInput
	if err := row.Scan(&s.ID, &s.Date); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SnapshotInput{}, errSnapshotGone
		}
		return SnapshotInput{}, fmt.Errorf("load snapshot: %w", err)
	}
	return s, nil
}

func loadSnapshotValues(ctx context.Context, tx pgx.Tx, snapshotID int) ([]SnapshotValueInput, error) {
	rows, err := tx.Query(ctx, `
		SELECT account_id, asset_id, value
		FROM snapshot_values WHERE snapshot_id = $1`,
		snapshotID,
	)
	if err != nil {
		return nil, fmt.Errorf("load snapshot_values: %w", err)
	}
	defer rows.Close()
	out := []SnapshotValueInput{}
	for rows.Next() {
		var v SnapshotValueInput
		var account, asset *int
		var value decimal.Decimal
		if err := rows.Scan(&account, &asset, &value); err != nil {
			return nil, fmt.Errorf("scan snapshot_value: %w", err)
		}
		v.AccountID = account
		v.AssetID = asset
		v.Value = value
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot_values: %w", err)
	}
	return out, nil
}

// loadAccountsByID loads every account referenced by the snapshots being
// recomputed, regardless of is_active. Aggregates are the historical
// reporting view (issue #394): soft-deleting an account must not retroactively
// drop its contribution from old snapshots.
func loadAccountsByID(ctx context.Context, tx pgx.Tx, ids []int) ([]AccountInput, error) {
	if len(ids) == 0 {
		return []AccountInput{}, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id, owner_user_id, type, category
		FROM accounts WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("load accounts for aggregates: %w", err)
	}
	defer rows.Close()
	out := []AccountInput{}
	for rows.Next() {
		var a AccountInput
		if err := rows.Scan(&a.ID, &a.OwnerUserID, &a.Type, &a.Category); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}
	return out, nil
}

// loadAssetIDs loads every asset id referenced by the snapshots being
// recomputed, regardless of is_active. See loadAccountsByID for the why.
func loadAssetIDs(ctx context.Context, tx pgx.Tx, ids []int) (map[int]struct{}, error) {
	if len(ids) == 0 {
		return map[int]struct{}{}, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id FROM assets WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("load assets for aggregates: %w", err)
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

// Pool exposes the pgx pool for callers that need to BeginTx outside the
// aggregates package (accounts/assets/snapshots use this to wrap their
// mutations + recompute in one tx).
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

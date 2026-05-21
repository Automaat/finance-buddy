// Package assets implements /api/assets — CRUD with cascade-on-delete:
// soft-delete triggers snapshot_aggregates recompute for snapshots that
// reference the asset (inactive asset contributes 0).
package assets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/aggregates"
)

// Asset mirrors backend/app/models/asset.Asset.
type Asset struct {
	ID        int
	Name      string
	IsActive  bool
	CreatedAt time.Time
}

// Sentinel errors mapped to HTTP status codes.
var (
	ErrNotFound      = errors.New("asset not found")
	ErrDuplicateName = errors.New("asset name already exists")
)

// Store is the persistence boundary.
type Store struct {
	pool       *pgxpool.Pool
	aggregates *aggregates.Store
}

// NewStore wires the pool + aggregates store (cascade-on-delete uses it).
func NewStore(pool *pgxpool.Pool, aggs *aggregates.Store) *Store {
	return &Store{pool: pool, aggregates: aggs}
}

const selectColumns = `id, name, is_active, created_at`

// List returns active assets.
func (s *Store) List(ctx context.Context) ([]Asset, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+selectColumns+` FROM assets WHERE is_active = true ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()
	out := []Asset{}
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		out = append(out, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assets: %w", err)
	}
	return out, nil
}

// LatestValuesByAsset returns the most recent snapshot value per asset.
func (s *Store) LatestValuesByAsset(ctx context.Context, assetIDs []int) (map[int]decimal.Decimal, error) {
	if len(assetIDs) == 0 {
		return map[int]decimal.Decimal{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (sv.asset_id) sv.asset_id, sv.value
		FROM snapshot_values sv
		JOIN snapshots s ON s.id = sv.snapshot_id
		WHERE sv.asset_id = ANY($1)
		ORDER BY sv.asset_id, s.date DESC`,
		assetIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("latest asset values: %w", err)
	}
	defer rows.Close()
	out := map[int]decimal.Decimal{}
	for rows.Next() {
		var id int
		var v decimal.Decimal
		if err := rows.Scan(&id, &v); err != nil {
			return nil, fmt.Errorf("scan latest asset value: %w", err)
		}
		out[id] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest asset values: %w", err)
	}
	return out, nil
}

// LatestValueForAsset returns the asset's value in the latest snapshot, or
// zero. Mirrors Python: take the most recent snapshot by date, then the
// matching SnapshotValue.
func (s *Store) LatestValueForAsset(ctx context.Context, assetID int) (decimal.Decimal, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT sv.value
		FROM snapshot_values sv
		JOIN snapshots s ON s.id = sv.snapshot_id
		WHERE sv.asset_id = $1
		  AND s.id = (SELECT id FROM snapshots ORDER BY date DESC LIMIT 1)
		LIMIT 1`,
		assetID,
	)
	var v decimal.Decimal
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return decimal.Zero, nil
		}
		return decimal.Zero, fmt.Errorf("latest asset value: %w", err)
	}
	return v, nil
}

// Create inserts an asset row. Duplicate name -> ErrDuplicateName.
func (s *Store) Create(ctx context.Context, name string) (*Asset, error) {
	if err := s.checkDuplicateName(ctx, name, 0); err != nil {
		return nil, err
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO assets (name, is_active, created_at)
		VALUES ($1, true, $2)
		RETURNING `+selectColumns,
		name, time.Now().UTC(),
	)
	return scanAsset(row)
}

// Update applies the (only) field: name. No cascade — assets don't carry
// owner/category, so aggregates are unaffected by a rename.
func (s *Store) Update(ctx context.Context, id int, name *string) (*Asset, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin update tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	current, err := lockAsset(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if name != nil && *name != current.Name {
		if err := s.checkDuplicateNameTx(ctx, tx, *name, id); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `UPDATE assets SET name = $1 WHERE id = $2`, *name, id); err != nil {
			return nil, fmt.Errorf("update asset: %w", err)
		}
		current.Name = *name
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update tx: %w", err)
	}
	committed = true
	return current, nil
}

// Delete soft-deletes and cascades into aggregate recompute.
func (s *Store) Delete(ctx context.Context, id int) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin delete tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	current, err := lockAsset(ctx, tx, id)
	if err != nil {
		return err
	}
	if !current.IsActive {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit idempotent delete: %w", err)
		}
		committed = true
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE assets SET is_active = false WHERE id = $1`, id); err != nil {
		return fmt.Errorf("soft-delete asset: %w", err)
	}
	ids, err := s.aggregates.AffectedSnapshotIDsForAsset(ctx, tx, id)
	if err != nil {
		return err
	}
	if err := s.aggregates.RecomputeForSnapshots(ctx, tx, ids); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete tx: %w", err)
	}
	committed = true
	return nil
}

func (s *Store) checkDuplicateName(ctx context.Context, name string, excludeID int) error {
	q := `SELECT id FROM assets WHERE name = $1 AND is_active = true`
	args := []any{name}
	if excludeID > 0 {
		q += ` AND id <> $2`
		args = append(args, excludeID)
	}
	var existing int
	err := s.pool.QueryRow(ctx, q+` LIMIT 1`, args...).Scan(&existing)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check duplicate asset name: %w", err)
	}
	return ErrDuplicateName
}

func (s *Store) checkDuplicateNameTx(ctx context.Context, tx pgx.Tx, name string, excludeID int) error {
	var existing int
	err := tx.QueryRow(ctx, `
		SELECT id FROM assets
		WHERE name = $1 AND is_active = true AND id <> $2
		LIMIT 1`,
		name, excludeID,
	).Scan(&existing)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check duplicate asset name (tx): %w", err)
	}
	return ErrDuplicateName
}

func lockAsset(ctx context.Context, tx pgx.Tx, id int) (*Asset, error) {
	row := tx.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM assets WHERE id = $1 FOR UPDATE`, id,
	)
	a, err := scanAsset(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock asset: %w", err)
	}
	return a, nil
}

func scanAsset(row pgx.Row) (*Asset, error) {
	var a Asset
	if err := row.Scan(&a.ID, &a.Name, &a.IsActive, &a.CreatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

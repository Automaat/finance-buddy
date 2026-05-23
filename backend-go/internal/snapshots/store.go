// Package snapshots implements /api/snapshots — atomic
// create/update of a snapshot + its values + cascade
// snapshot_aggregates recompute, all in one transaction.
package snapshots

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/aggregates"
)

// Snapshot mirrors backend/app/models/snapshot.Snapshot.
type Snapshot struct {
	ID        int
	Date      time.Time
	Notes     *string
	CreatedAt time.Time
}

// Value is one row from snapshot_values + the related asset/account name
// resolved via LEFT JOIN.
type Value struct {
	ID          int
	AssetID     *int
	AssetName   *string
	AccountID   *int
	AccountName *string
	Value       decimal.Decimal
}

// ListItem is what GET /api/snapshots returns per row.
type ListItem struct {
	ID            int
	Date          time.Time
	Notes         *string
	TotalNetWorth float64
}

// ValueInput is one POST/PUT body entry. AssetID and AccountID are
// mutually-exclusive — the validator enforces "exactly one".
type ValueInput struct {
	AssetID   *int
	AccountID *int
	Value     decimal.Decimal
}

// Sentinel errors mapped to HTTP status codes by the handler.
var (
	ErrNotFound            = errors.New("snapshot not found")
	ErrDuplicateDate       = errors.New("snapshot date conflicts")
	ErrDuplicateAssetIDs   = errors.New("duplicate asset ids")
	ErrDuplicateAccountIDs = errors.New("duplicate account ids")
)

// MissingReferencesError carries the missing IDs back to the handler so the
// 404 detail string can spell them out (matches Python's "Assets not found:
// {1, 2}" message).
type MissingReferencesError struct {
	MissingAssets   []int
	MissingAccounts []int
}

func (e *MissingReferencesError) Error() string {
	return "missing references"
}

// Store is the persistence boundary.
type Store struct {
	pool       *pgxpool.Pool
	aggregates *aggregates.Store
}

// NewStore wires the pool + aggregates store (used for recompute).
func NewStore(pool *pgxpool.Pool, aggs *aggregates.Store) *Store {
	return &Store{pool: pool, aggregates: aggs}
}

// List returns all snapshots ordered by date desc, each with its total_net_worth.
// Soft-deleted accounts/assets still contribute to historical snapshots they
// belong to — see issue #394. The active flag governs current-management
// views, not historical totals.
func (s *Store) List(ctx context.Context) ([]ListItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.date, s.notes,
			COALESCE(SUM(
				CASE
					WHEN a.id IS NOT NULL THEN sv.value
					WHEN acc.id IS NOT NULL AND acc.type = 'asset' THEN sv.value
					WHEN acc.id IS NOT NULL AND acc.type = 'liability' THEN -sv.value
					ELSE 0
				END
			), 0) AS total_net_worth
		FROM snapshots s
		LEFT JOIN snapshot_values sv ON sv.snapshot_id = s.id
		LEFT JOIN assets a ON a.id = sv.asset_id
		LEFT JOIN accounts acc ON acc.id = sv.account_id
		GROUP BY s.id, s.date, s.notes
		ORDER BY s.date DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()
	out := []ListItem{}
	for rows.Next() {
		var item ListItem
		var total decimal.Decimal
		if err := rows.Scan(&item.ID, &item.Date, &item.Notes, &total); err != nil {
			return nil, fmt.Errorf("scan snapshot list item: %w", err)
		}
		item.TotalNetWorth, _ = total.Float64()
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot list: %w", err)
	}
	return out, nil
}

// Get returns one snapshot + all its values with names joined in.
func (s *Store) Get(ctx context.Context, id int) (*Snapshot, []Value, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, date, notes, created_at FROM snapshots WHERE id = $1`, id,
	)
	var snap Snapshot
	if err := row.Scan(&snap.ID, &snap.Date, &snap.Notes, &snap.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("get snapshot: %w", err)
	}
	values, err := s.loadValues(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return &snap, values, nil
}

func (s *Store) loadValues(ctx context.Context, snapshotID int) ([]Value, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sv.id, sv.asset_id, a.name, sv.account_id, acc.name, sv.value
		FROM snapshot_values sv
		LEFT JOIN assets a ON a.id = sv.asset_id
		LEFT JOIN accounts acc ON acc.id = sv.account_id
		WHERE sv.snapshot_id = $1
		ORDER BY sv.id`,
		snapshotID,
	)
	if err != nil {
		return nil, fmt.Errorf("load snapshot values: %w", err)
	}
	defer rows.Close()
	out := []Value{}
	for rows.Next() {
		var v Value
		if err := rows.Scan(&v.ID, &v.AssetID, &v.AssetName, &v.AccountID, &v.AccountName, &v.Value); err != nil {
			return nil, fmt.Errorf("scan snapshot value: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot values: %w", err)
	}
	return out, nil
}

// Create inserts a snapshot + all values + runs the aggregate recompute,
// all atomically. ErrDuplicateDate if the date is already taken,
// MissingReferencesError if any referenced asset/account doesn't exist or is
// inactive.
func (s *Store) Create(ctx context.Context, date time.Time, notes *string, values []ValueInput) (*Snapshot, []Value, error) {
	if err := validateValueIDs(values); err != nil {
		return nil, nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin create tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := checkDuplicateDate(ctx, tx, date, 0); err != nil {
		return nil, nil, err
	}
	if err := checkReferencesExistActive(ctx, tx, values); err != nil {
		return nil, nil, err
	}
	var snap Snapshot
	err = tx.QueryRow(ctx, `
		INSERT INTO snapshots (date, notes, created_at)
		VALUES ($1, $2, $3)
		RETURNING id, date, notes, created_at`,
		date, notes, time.Now().UTC(),
	).Scan(&snap.ID, &snap.Date, &snap.Notes, &snap.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, nil, ErrDuplicateDate
		}
		return nil, nil, fmt.Errorf("insert snapshot: %w", err)
	}
	if err := insertValues(ctx, tx, snap.ID, values); err != nil {
		return nil, nil, err
	}
	if err := s.aggregates.RecomputeForSnapshot(ctx, tx, snap.ID); err != nil {
		return nil, nil, err
	}
	stored, err := loadValuesTx(ctx, tx, snap.ID)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit create tx: %w", err)
	}
	committed = true
	return &snap, stored, nil
}

// UpdatePatch is the sparse update set. ValuesSet=true distinguishes
// "values omitted" (keep) from "values present" (replace).
type UpdatePatch struct {
	Date      *time.Time
	NotesSet  bool
	Notes     *string
	ValuesSet bool
	Values    []ValueInput
}

// Update mutates the snapshot, optionally replaces values, and runs the
// aggregate recompute — all in one transaction.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*Snapshot, []Value, error) {
	if p.ValuesSet {
		if err := validateValueIDs(p.Values); err != nil {
			return nil, nil, err
		}
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin update tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	snap, err := lockSnapshot(ctx, tx, id)
	if err != nil {
		return nil, nil, err
	}
	if p.Date != nil {
		if err := checkDuplicateDate(ctx, tx, *p.Date, id); err != nil {
			return nil, nil, err
		}
		snap.Date = *p.Date
	}
	if p.NotesSet {
		snap.Notes = p.Notes
	}
	if _, err := tx.Exec(ctx,
		`UPDATE snapshots SET date = $1, notes = $2 WHERE id = $3`,
		snap.Date, snap.Notes, id,
	); err != nil {
		if isUniqueViolation(err) {
			return nil, nil, ErrDuplicateDate
		}
		return nil, nil, fmt.Errorf("update snapshot: %w", err)
	}
	if p.ValuesSet {
		if err := checkReferencesExist(ctx, tx, p.Values); err != nil {
			return nil, nil, err
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM snapshot_values WHERE snapshot_id = $1`, id,
		); err != nil {
			return nil, nil, fmt.Errorf("delete snapshot values: %w", err)
		}
		if err := insertValues(ctx, tx, id, p.Values); err != nil {
			return nil, nil, err
		}
	}
	if err := s.aggregates.RecomputeForSnapshot(ctx, tx, id); err != nil {
		return nil, nil, err
	}
	stored, err := loadValuesTx(ctx, tx, id)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit update tx: %w", err)
	}
	committed = true
	return snap, stored, nil
}

// --- helpers ---

func validateValueIDs(values []ValueInput) error {
	seenAssets := map[int]struct{}{}
	seenAccounts := map[int]struct{}{}
	for _, v := range values {
		if v.AssetID != nil {
			if _, dup := seenAssets[*v.AssetID]; dup {
				return ErrDuplicateAssetIDs
			}
			seenAssets[*v.AssetID] = struct{}{}
		}
		if v.AccountID != nil {
			if _, dup := seenAccounts[*v.AccountID]; dup {
				return ErrDuplicateAccountIDs
			}
			seenAccounts[*v.AccountID] = struct{}{}
		}
	}
	return nil
}

func checkDuplicateDate(ctx context.Context, tx pgx.Tx, date time.Time, excludeID int) error {
	q := `SELECT id FROM snapshots WHERE date = $1`
	args := []any{date}
	if excludeID > 0 {
		q += ` AND id <> $2`
		args = append(args, excludeID)
	}
	var existing int
	err := tx.QueryRow(ctx, q+` LIMIT 1`, args...).Scan(&existing)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check duplicate date: %w", err)
	}
	return ErrDuplicateDate
}

// checkReferencesExistActive enforces is_active=true (used on POST, matches
// Python's `Asset.is_active.is_(True)`).
func checkReferencesExistActive(ctx context.Context, tx pgx.Tx, values []ValueInput) error {
	return checkReferences(ctx, tx, values, true)
}

// checkReferencesExist ignores is_active (used on PUT, matches Python's
// PUT path which doesn't filter by is_active when validating IDs).
func checkReferencesExist(ctx context.Context, tx pgx.Tx, values []ValueInput) error {
	return checkReferences(ctx, tx, values, false)
}

func checkReferences(ctx context.Context, tx pgx.Tx, values []ValueInput, activeOnly bool) error {
	assetIDs := []int{}
	accountIDs := []int{}
	for _, v := range values {
		if v.AssetID != nil {
			assetIDs = append(assetIDs, *v.AssetID)
		}
		if v.AccountID != nil {
			accountIDs = append(accountIDs, *v.AccountID)
		}
	}
	missingAssets, err := missingFrom(ctx, tx, "assets", assetIDs, activeOnly)
	if err != nil {
		return err
	}
	missingAccounts, err := missingFrom(ctx, tx, "accounts", accountIDs, activeOnly)
	if err != nil {
		return err
	}
	if len(missingAssets) > 0 || len(missingAccounts) > 0 {
		sort.Ints(missingAssets)
		sort.Ints(missingAccounts)
		return &MissingReferencesError{
			MissingAssets:   missingAssets,
			MissingAccounts: missingAccounts,
		}
	}
	return nil
}

func missingFrom(ctx context.Context, tx pgx.Tx, table string, ids []int, activeOnly bool) ([]int, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := fmt.Sprintf(`SELECT id FROM %s WHERE id = ANY($1)`, table)
	if activeOnly {
		q += ` AND is_active = true`
	}
	rows, err := tx.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("check refs %s: %w", table, err)
	}
	defer rows.Close()
	found := map[int]struct{}{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan ref %s: %w", table, err)
		}
		found[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate refs %s: %w", table, err)
	}
	missing := []int{}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing, nil
}

// insertValues batches all inserts through a single pgx.Batch so a snapshot
// with N values pays one round-trip instead of N.
func insertValues(ctx context.Context, tx pgx.Tx, snapshotID int, values []ValueInput) error {
	if len(values) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, v := range values {
		batch.Queue(`
			INSERT INTO snapshot_values (snapshot_id, asset_id, account_id, value)
			VALUES ($1, $2, $3, $4)`,
			snapshotID, v.AssetID, v.AccountID, v.Value,
		)
	}
	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for i := range values {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert snapshot value[%d]: %w", i, err)
		}
	}
	return nil
}

// isUniqueViolation checks whether err is a Postgres 23505. Used to convert
// a racy duplicate-date INSERT into the same 400 the pre-check returns.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr == nil {
		return false
	}
	return pgErr.Code == pgerrcode.UniqueViolation
}

func lockSnapshot(ctx context.Context, tx pgx.Tx, id int) (*Snapshot, error) {
	row := tx.QueryRow(ctx,
		`SELECT id, date, notes, created_at FROM snapshots WHERE id = $1 FOR UPDATE`, id,
	)
	var s Snapshot
	if err := row.Scan(&s.ID, &s.Date, &s.Notes, &s.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock snapshot: %w", err)
	}
	return &s, nil
}

func loadValuesTx(ctx context.Context, tx pgx.Tx, snapshotID int) ([]Value, error) {
	rows, err := tx.Query(ctx, `
		SELECT sv.id, sv.asset_id, a.name, sv.account_id, acc.name, sv.value
		FROM snapshot_values sv
		LEFT JOIN assets a ON a.id = sv.asset_id
		LEFT JOIN accounts acc ON acc.id = sv.account_id
		WHERE sv.snapshot_id = $1
		ORDER BY sv.id`,
		snapshotID,
	)
	if err != nil {
		return nil, fmt.Errorf("load snapshot values: %w", err)
	}
	defer rows.Close()
	out := []Value{}
	for rows.Next() {
		var v Value
		if err := rows.Scan(&v.ID, &v.AssetID, &v.AssetName, &v.AccountID, &v.AccountName, &v.Value); err != nil {
			return nil, fmt.Errorf("scan snapshot value: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot values: %w", err)
	}
	return out, nil
}

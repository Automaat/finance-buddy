// Package accounts implements /api/accounts — CRUD with cascade-on-edit:
// owner/category changes trigger snapshot_aggregates recompute for every
// snapshot that references this account.
package accounts

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

// Account mirrors backend/app/models/account.Account. OwnerUserID is the
// owning household member; nil means jointly owned ("Shared").
type Account struct {
	ID                    int
	Name                  string
	Type                  string
	Category              string
	OwnerUserID           *int
	Currency              string
	AccountWrapper        *string
	Purpose               string
	SquareMeters          *decimal.Decimal
	IsActive              bool
	ReceivesContributions bool
	ExcludedFromFire      bool
	CreatedAt             time.Time
}

// Sentinel errors mapped to HTTP status codes by the handler.
var (
	ErrNotFound      = errors.New("account not found")
	ErrDuplicateName = errors.New("account name already exists")
)

// Store is the persistence boundary.
type Store struct {
	pool       *pgxpool.Pool
	aggregates *aggregates.Store
}

// NewStore wires the pool + aggregates store (used for cascade-on-edit).
func NewStore(pool *pgxpool.Pool, aggs *aggregates.Store) *Store {
	return &Store{pool: pool, aggregates: aggs}
}

const selectColumns = `
	id, name, type, category, owner_user_id, currency, account_wrapper, purpose,
	square_meters, is_active, receives_contributions, excluded_from_fire, created_at
`

// List returns active accounts.
func (s *Store) List(ctx context.Context) ([]Account, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+selectColumns+` FROM accounts WHERE is_active = true ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		out = append(out, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}
	return out, nil
}

// Get returns the account regardless of is_active, mirroring Python's
// get_or_404 helper. Returns ErrNotFound when the row doesn't exist; callers
// that need an active row check IsActive themselves.
func (s *Store) Get(ctx context.Context, id int) (*Account, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM accounts WHERE id = $1`, id,
	)
	a, err := scanAccount(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get account: %w", err)
	}
	return a, nil
}

// LatestValuesByAccount returns the most recent active snapshot value per
// account in one query — used to fill AccountResponse.current_value.
func (s *Store) LatestValuesByAccount(ctx context.Context, accountIDs []int) (map[int]decimal.Decimal, error) {
	if len(accountIDs) == 0 {
		return map[int]decimal.Decimal{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (sv.account_id) sv.account_id, sv.value
		FROM snapshot_values sv
		JOIN snapshots s ON s.id = sv.snapshot_id
		WHERE sv.account_id = ANY($1)
		ORDER BY sv.account_id, s.date DESC`,
		accountIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("latest values: %w", err)
	}
	defer rows.Close()
	out := map[int]decimal.Decimal{}
	for rows.Next() {
		var id int
		var v decimal.Decimal
		if err := rows.Scan(&id, &v); err != nil {
			return nil, fmt.Errorf("scan latest value: %w", err)
		}
		out[id] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest values: %w", err)
	}
	return out, nil
}

// LatestValueForAccount returns the most recent active snapshot value for
// one account, or zero when none exists (matches Python's 0.0 fallback).
func (s *Store) LatestValueForAccount(ctx context.Context, accountID int) (decimal.Decimal, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT sv.value
		FROM snapshot_values sv
		JOIN snapshots s ON s.id = sv.snapshot_id
		WHERE sv.account_id = $1
		ORDER BY s.date DESC LIMIT 1`,
		accountID,
	)
	var v decimal.Decimal
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return decimal.Zero, nil
		}
		return decimal.Zero, fmt.Errorf("latest value: %w", err)
	}
	return v, nil
}

// Create inserts an account. Returns ErrDuplicateName for a name collision.
func (s *Store) Create(ctx context.Context, a *Account) (*Account, error) {
	if err := s.checkDuplicateName(ctx, a.Name, 0); err != nil {
		return nil, err
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO accounts (
			name, type, category, owner_user_id, currency, account_wrapper,
			purpose, square_meters, is_active, receives_contributions,
			excluded_from_fire, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, true, $9, $10, $11
		)
		RETURNING `+selectColumns,
		a.Name, a.Type, a.Category, a.OwnerUserID, a.Currency, a.AccountWrapper, a.Purpose,
		a.SquareMeters, a.ReceivesContributions, a.ExcludedFromFire, time.Now().UTC(),
	)
	return scanAccount(row)
}

// UpdatePatch is the sparse update set with explicit "field was set" booleans
// for the two nullable fields whose null-vs-omit semantics differ.
type UpdatePatch struct {
	Name                  *string
	Category              *string
	OwnerUserIDSet        bool
	OwnerUserID           *int
	Currency              *string
	AccountWrapperSet     bool
	AccountWrapper        *string
	Purpose               *string
	SquareMetersSet       bool
	SquareMeters          *decimal.Decimal
	ReceivesContributions *bool
	ExcludedFromFire      *bool
}

// Update applies the patch. Owner/category changes cascade into a
// snapshot_aggregates recompute for every snapshot referencing the account.
// The whole op runs in one transaction.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*Account, error) {
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
	current, err := lockAccount(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if p.Name != nil && *p.Name != current.Name {
		if err := s.checkDuplicateNameTx(ctx, tx, *p.Name, id); err != nil {
			return nil, err
		}
	}
	// Capture pre-patch values to decide whether the request meaningfully
	// changed owner or category — matches Python's old_owner/old_category.
	oldOwnerUserID := current.OwnerUserID
	oldCategory := current.Category
	applyPatch(current, p)
	needsRecompute := (p.OwnerUserIDSet && !intPtrEqual(p.OwnerUserID, oldOwnerUserID)) ||
		(p.Category != nil && *p.Category != oldCategory)
	if err := s.updateRow(ctx, tx, id, current); err != nil {
		return nil, err
	}
	if needsRecompute {
		ids, err := s.aggregates.AffectedSnapshotIDsForAccount(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		if err := s.aggregates.RecomputeForSnapshots(ctx, tx, ids); err != nil {
			return nil, err
		}
	}
	updated, err := loadAccount(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update tx: %w", err)
	}
	committed = true
	return updated, nil
}

// Delete soft-deletes + cascades into aggregate recompute.
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
	current, err := lockAccount(ctx, tx, id)
	if err != nil {
		return err
	}
	if !current.IsActive {
		// Idempotent — matches Python's `if not is_active: return`.
		// Commit the empty tx so the FOR UPDATE row lock releases promptly,
		// and mark committed so the deferred rollback is a no-op.
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit idempotent delete: %w", err)
		}
		committed = true
		return nil
	}
	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET is_active = false WHERE id = $1`, id,
	); err != nil {
		return fmt.Errorf("soft-delete account: %w", err)
	}
	ids, err := s.aggregates.AffectedSnapshotIDsForAccount(ctx, tx, id)
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
	var existing int
	args := []any{name}
	q := `SELECT id FROM accounts WHERE name = $1 AND is_active = true`
	if excludeID > 0 {
		q += ` AND id <> $2`
		args = append(args, excludeID)
	}
	err := s.pool.QueryRow(ctx, q+` LIMIT 1`, args...).Scan(&existing)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check duplicate name: %w", err)
	}
	return ErrDuplicateName
}

func (s *Store) checkDuplicateNameTx(ctx context.Context, tx pgx.Tx, name string, excludeID int) error {
	var existing int
	err := tx.QueryRow(ctx, `
		SELECT id FROM accounts
		WHERE name = $1 AND is_active = true AND id <> $2
		LIMIT 1`,
		name, excludeID,
	).Scan(&existing)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check duplicate name (tx): %w", err)
	}
	return ErrDuplicateName
}

func (s *Store) updateRow(ctx context.Context, tx pgx.Tx, id int, a *Account) error {
	_, err := tx.Exec(ctx, `
		UPDATE accounts SET
			name = $1, type = $2, category = $3,
			owner_user_id = $4, currency = $5,
			account_wrapper = $6, purpose = $7, square_meters = $8,
			receives_contributions = $9, excluded_from_fire = $10
		WHERE id = $11`,
		a.Name, a.Type, a.Category, a.OwnerUserID, a.Currency,
		a.AccountWrapper, a.Purpose, a.SquareMeters,
		a.ReceivesContributions, a.ExcludedFromFire, id,
	)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	return nil
}

func lockAccount(ctx context.Context, tx pgx.Tx, id int) (*Account, error) {
	row := tx.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM accounts WHERE id = $1 FOR UPDATE`, id,
	)
	a, err := scanAccount(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock account: %w", err)
	}
	return a, nil
}

func loadAccount(ctx context.Context, tx pgx.Tx, id int) (*Account, error) {
	row := tx.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM accounts WHERE id = $1`, id,
	)
	a, err := scanAccount(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("load account: %w", err)
	}
	return a, nil
}

func applyPatch(a *Account, p UpdatePatch) {
	if p.Name != nil {
		a.Name = *p.Name
	}
	if p.Category != nil {
		a.Category = *p.Category
	}
	if p.OwnerUserIDSet {
		a.OwnerUserID = p.OwnerUserID
	}
	if p.Currency != nil {
		a.Currency = *p.Currency
	}
	if p.Purpose != nil {
		a.Purpose = *p.Purpose
	}
	if p.ReceivesContributions != nil {
		a.ReceivesContributions = *p.ReceivesContributions
	}
	if p.AccountWrapperSet {
		a.AccountWrapper = p.AccountWrapper
	}
	if p.SquareMetersSet {
		a.SquareMeters = p.SquareMeters
	}
	if p.ExcludedFromFire != nil {
		a.ExcludedFromFire = *p.ExcludedFromFire
	}
}

func scanAccount(row pgx.Row) (*Account, error) {
	var a Account
	if err := row.Scan(
		&a.ID, &a.Name, &a.Type, &a.Category, &a.OwnerUserID, &a.Currency,
		&a.AccountWrapper, &a.Purpose, &a.SquareMeters,
		&a.IsActive, &a.ReceivesContributions, &a.ExcludedFromFire, &a.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &a, nil
}

// intPtrEqual reports whether two optional ints are equal (both nil counts).
func intPtrEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

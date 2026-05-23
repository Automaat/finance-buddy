// Package allocation implements the /api/allocation/targets endpoints +
// the actual-vs-target drift compute that powers the dashboard widget.
//
// A target is one (owner_user_id, category) pair carrying a target_pct.
// owner_user_id NULL = household-level target (jointly owned). Per-owner
// targets must sum to exactly 100 before they are usable for drift; the
// store accepts partial sets so the UI can edit incrementally and the
// bulk Replace endpoint enforces the sum.
package allocation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Target is one allocation_targets row.
type Target struct {
	ID          int
	Category    string
	OwnerUserID *int
	TargetPct   decimal.Decimal
	CreatedAt   time.Time
}

// ErrNotFound is returned when no row matches the supplied id.
var ErrNotFound = errors.New("allocation target not found")

// OwnerMissingError is returned when create/update references a
// non-existent owner_user_id; mapped to a 404.
type OwnerMissingError struct {
	OwnerUserID int
}

func (e *OwnerMissingError) Error() string {
	return fmt.Sprintf("user with id %d not found", e.OwnerUserID)
}

// DuplicateScopeError is returned when (owner_user_id, category) collides
// with an existing row; mapped to a 409.
type DuplicateScopeError struct {
	Category    string
	OwnerUserID *int
}

func (e *DuplicateScopeError) Error() string {
	return fmt.Sprintf("target for category %q in this scope already exists", e.Category)
}

// Store is the persistence boundary for allocation_targets.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `id, category, owner_user_id, target_pct, created_at`

// EnsureSchema creates the allocation_targets table if missing. Idempotent
// so it can run on every backend start — schema.sql only seeds empty
// databases, so existing prod installs need the additive DDL here too.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS allocation_targets (
			id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			category varchar(50) NOT NULL,
			owner_user_id integer,
			target_pct numeric(5,2) NOT NULL,
			created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			CONSTRAINT allocation_targets_pct_range CHECK (target_pct >= 0 AND target_pct <= 100),
			CONSTRAINT allocation_targets_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
		)`); err != nil {
		return fmt.Errorf("ensure allocation_targets table: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS allocation_targets_scope_category_uniq
			ON allocation_targets (COALESCE(owner_user_id, -1), category)`); err != nil {
		return fmt.Errorf("ensure allocation_targets uniq index: %w", err)
	}
	return nil
}

// List returns every target ordered by (owner_user_id NULLS FIRST, category).
func (s *Store) List(ctx context.Context) ([]Target, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+selectColumns+` FROM allocation_targets
		 ORDER BY owner_user_id NULLS FIRST, category`,
	)
	if err != nil {
		return nil, fmt.Errorf("list allocation targets: %w", err)
	}
	defer rows.Close()
	out := []Target{}
	for rows.Next() {
		t, err := scanTarget(rows)
		if err != nil {
			return nil, fmt.Errorf("scan allocation target: %w", err)
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate allocation targets: %w", err)
	}
	return out, nil
}

// Get returns one target by id; ErrNotFound when missing.
func (s *Store) Get(ctx context.Context, id int) (*Target, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM allocation_targets WHERE id = $1`, id,
	)
	t, err := scanTarget(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get allocation target: %w", err)
	}
	return t, nil
}

// Create inserts a new target. Returns OwnerMissingError when owner_user_id
// references a non-existent user, DuplicateScopeError when the (owner,
// category) pair is already taken.
func (s *Store) Create(ctx context.Context, t *Target) (*Target, error) {
	if err := s.validateOwner(ctx, t.OwnerUserID); err != nil {
		return nil, err
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO allocation_targets (category, owner_user_id, target_pct, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING `+selectColumns,
		t.Category, t.OwnerUserID, t.TargetPct, time.Now().UTC(),
	)
	created, err := scanTarget(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, &DuplicateScopeError{Category: t.Category, OwnerUserID: t.OwnerUserID}
		}
		if isForeignKeyViolation(err) && t.OwnerUserID != nil {
			// Race: user was deleted between validateOwner and the INSERT.
			return nil, &OwnerMissingError{OwnerUserID: *t.OwnerUserID}
		}
		return nil, fmt.Errorf("insert allocation target: %w", err)
	}
	return created, nil
}

// UpdatePatch is a sparse update; only set fields apply. Scope (category +
// owner_user_id) is immutable — moving a target between scopes is a
// delete + create.
type UpdatePatch struct {
	TargetPct *decimal.Decimal
}

// Update applies the patch and returns the refreshed target.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*Target, error) {
	if p.TargetPct == nil {
		return s.Get(ctx, id)
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE allocation_targets SET target_pct = $1
		WHERE id = $2
		RETURNING `+selectColumns,
		*p.TargetPct, id,
	)
	updated, err := scanTarget(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update allocation target: %w", err)
	}
	return updated, nil
}

// Delete removes the target by id.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM allocation_targets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete allocation target: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Replace atomically replaces every target for one owner scope (NULL = all
// household-level rows). The caller validates sum-to-100 before calling.
// A per-scope pg_advisory_xact_lock serializes concurrent replaces so two
// in-flight payloads cannot leave the scope in a torn state.
func (s *Store) Replace(ctx context.Context, ownerUserID *int, targets []Target) ([]Target, error) {
	if err := s.validateOwner(ctx, ownerUserID); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin replace tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	scopeLockKey := -1
	if ownerUserID != nil {
		scopeLockKey = *ownerUserID
	}
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('alloc_scope_' || $1::text))`, scopeLockKey,
	); err != nil {
		return nil, fmt.Errorf("acquire scope lock: %w", err)
	}
	if ownerUserID == nil {
		if _, err := tx.Exec(ctx, `DELETE FROM allocation_targets WHERE owner_user_id IS NULL`); err != nil {
			return nil, fmt.Errorf("clear household targets: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `DELETE FROM allocation_targets WHERE owner_user_id = $1`, *ownerUserID); err != nil {
			return nil, fmt.Errorf("clear owner targets: %w", err)
		}
	}
	now := time.Now().UTC()
	out := make([]Target, 0, len(targets))
	for _, t := range targets {
		row := tx.QueryRow(ctx, `
			INSERT INTO allocation_targets (category, owner_user_id, target_pct, created_at)
			VALUES ($1, $2, $3, $4)
			RETURNING `+selectColumns,
			t.Category, ownerUserID, t.TargetPct, now,
		)
		inserted, err := scanTarget(row)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, &DuplicateScopeError{Category: t.Category, OwnerUserID: ownerUserID}
			}
			return nil, fmt.Errorf("insert replacement target: %w", err)
		}
		out = append(out, *inserted)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit replace tx: %w", err)
	}
	committed = true
	return out, nil
}

func (s *Store) validateOwner(ctx context.Context, ownerUserID *int) error {
	if ownerUserID == nil {
		return nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, *ownerUserID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check user: %w", err)
	}
	if !exists {
		return &OwnerMissingError{OwnerUserID: *ownerUserID}
	}
	return nil
}

// isUniqueViolation reports whether err is a Postgres 23505. Concurrent
// inserts can race past the pre-check, so the caller converts the
// resulting error into a 409.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr == nil {
		return false
	}
	return pgErr.Code == pgerrcode.UniqueViolation
}

// isForeignKeyViolation reports whether err is a Postgres 23503.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr == nil {
		return false
	}
	return pgErr.Code == pgerrcode.ForeignKeyViolation
}

func scanTarget(row pgx.Row) (*Target, error) {
	var t Target
	if err := row.Scan(&t.ID, &t.Category, &t.OwnerUserID, &t.TargetPct, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

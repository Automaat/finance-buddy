package recurring

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Store is the persistence boundary for recurring transactions.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const recurringCols = `
	id, account_id, amount, owner_user_id, transaction_type, category, description,
	frequency, day_of_month, start_date, end_date, active, skipped_dates,
	last_run_date, created_at, updated_at
`

func scanRow(row pgx.Row) (Recurring, error) {
	var r Recurring
	var freq string
	if err := row.Scan(
		&r.ID, &r.AccountID, &r.Amount, &r.OwnerUserID, &r.TransactionType, &r.Category,
		&r.Description, &freq, &r.DayOfMonth, &r.StartDate, &r.EndDate, &r.Active,
		&r.SkippedDates, &r.LastRunDate, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return Recurring{}, err
	}
	r.Frequency = Frequency(freq)
	return r, nil
}

// List returns all recurring templates, ordered by creation.
func (s *Store) List(ctx context.Context) ([]Recurring, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+recurringCols+`
		FROM recurring_transactions
		ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list recurring: %w", err)
	}
	defer rows.Close()
	out := []Recurring{}
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recurring: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListActive returns active templates only — used by the scheduler.
func (s *Store) ListActive(ctx context.Context) ([]Recurring, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+recurringCols+`
		FROM recurring_transactions
		WHERE active = true
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list active recurring: %w", err)
	}
	defer rows.Close()
	out := []Recurring{}
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan active recurring: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Get returns one template by ID, ErrNotFound when missing.
func (s *Store) Get(ctx context.Context, id int) (Recurring, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+recurringCols+` FROM recurring_transactions WHERE id = $1`, id)
	r, err := scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Recurring{}, ErrNotFound
	}
	if err != nil {
		return Recurring{}, fmt.Errorf("get recurring %d: %w", id, err)
	}
	return r, nil
}

// CreateInput is the validated payload for Create.
type CreateInput struct {
	AccountID       int
	Amount          decimal.Decimal
	OwnerUserID     *int
	TransactionType *string
	Category        *string
	Description     string
	Frequency       Frequency
	DayOfMonth      *int
	StartDate       time.Time
	EndDate         *time.Time
	Active          bool
}

// Create inserts a new template, returning the persisted row.
func (s *Store) Create(ctx context.Context, in CreateInput) (Recurring, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO recurring_transactions
			(account_id, amount, owner_user_id, transaction_type, category, description,
			 frequency, day_of_month, start_date, end_date, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+recurringCols,
		in.AccountID, in.Amount, in.OwnerUserID, in.TransactionType, in.Category, in.Description,
		string(in.Frequency), in.DayOfMonth, in.StartDate, in.EndDate, in.Active,
	)
	return scanRow(row)
}

// UpdateInput is the validated payload for Update; same shape as CreateInput.
type UpdateInput = CreateInput

// Update overwrites a template. ErrNotFound when id is unknown.
func (s *Store) Update(ctx context.Context, id int, in UpdateInput) (Recurring, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE recurring_transactions
		SET account_id = $1, amount = $2, owner_user_id = $3, transaction_type = $4,
		    category = $5, description = $6, frequency = $7, day_of_month = $8,
		    start_date = $9, end_date = $10, active = $11,
		    updated_at = (now() at time zone 'utc')
		WHERE id = $12
		RETURNING `+recurringCols,
		in.AccountID, in.Amount, in.OwnerUserID, in.TransactionType, in.Category,
		in.Description, string(in.Frequency), in.DayOfMonth, in.StartDate, in.EndDate,
		in.Active, id,
	)
	r, err := scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Recurring{}, ErrNotFound
	}
	if err != nil {
		return Recurring{}, fmt.Errorf("update recurring %d: %w", id, err)
	}
	return r, nil
}

// Delete removes the template. ErrNotFound when id is unknown.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM recurring_transactions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete recurring %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AppendSkip adds a date to skipped_dates (dedup at the SQL level).
func (s *Store) AppendSkip(ctx context.Context, id int, date time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE recurring_transactions
		SET skipped_dates = (
		    SELECT array_agg(DISTINCT d ORDER BY d)
		    FROM unnest(skipped_dates || $2::date) AS d
		),
		updated_at = (now() at time zone 'utc')
		WHERE id = $1`, id, date)
	if err != nil {
		return fmt.Errorf("append skip on %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RemoveSkip drops `date` from skipped_dates if present.
func (s *Store) RemoveSkip(ctx context.Context, id int, date time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE recurring_transactions
		SET skipped_dates = array_remove(skipped_dates, $2::date),
		    updated_at = (now() at time zone 'utc')
		WHERE id = $1`, id, date)
	if err != nil {
		return fmt.Errorf("remove skip on %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AccountExists reports whether the given account id refers to an active
// account; used to reject creates pointing at deleted accounts.
func (s *Store) AccountExists(ctx context.Context, id int) (bool, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND is_active = true)`,
		id,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("account presence check: %w", err)
	}
	return exists, nil
}

// MintOccurrence inserts a concrete transaction for `r` on `date` and records
// the run, atomically and under a per-template advisory lock. The lock
// serializes a concurrent scheduler tick and a user clicking "Run now" so the
// same (recurring_id, date) cannot mint two transactions. Returns the new
// transaction id.
func (s *Store) MintOccurrence(ctx context.Context, r Recurring, date time.Time) (int, error) {
	var id int
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, lockErr := tx.Exec(ctx,
			`SELECT pg_advisory_xact_lock(hashtext($1))`,
			fmt.Sprintf("recurring_%d", r.ID),
		); lockErr != nil {
			return fmt.Errorf("advisory lock: %w", lockErr)
		}
		var freshLast *time.Time
		if err := tx.QueryRow(ctx,
			`SELECT last_run_date FROM recurring_transactions WHERE id = $1`, r.ID,
		).Scan(&freshLast); err != nil {
			return fmt.Errorf("reload last_run_date: %w", err)
		}
		if freshLast != nil && !freshLast.Before(date) {
			return errAlreadyMinted
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO transactions (account_id, amount, date, owner_user_id, transaction_type, is_active, created_at)
			VALUES ($1, $2, $3, $4, $5, true, (now() at time zone 'utc'))
			RETURNING id`,
			r.AccountID, r.Amount, date, r.OwnerUserID, r.TransactionType,
		).Scan(&id); err != nil {
			return fmt.Errorf("insert recurring-generated transaction: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE recurring_transactions
			SET last_run_date = $2, updated_at = (now() at time zone 'utc')
			WHERE id = $1`, r.ID, date); err != nil {
			return fmt.Errorf("update last_run_date: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// errAlreadyMinted signals that a concurrent caller already minted this or a
// later occurrence; mapped to nothing at the handler — Run Now silently
// no-ops to keep the UI idempotent.
var errAlreadyMinted = errors.New("recurring: occurrence already minted")

// IsAlreadyMinted reports whether err is the already-minted sentinel.
func IsAlreadyMinted(err error) bool {
	return errors.Is(err, errAlreadyMinted)
}

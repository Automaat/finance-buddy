// Package personas implements the /api/personas endpoints.
//
// PUT cascades an owner rename to 5 owner-bearing tables (accounts,
// transactions, salary_records, debt_payments, retirement_limits) inside one
// transaction. DELETE checks each of those tables for references and returns
// a 409 with the counts if any persist.
package personas

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

// Persona mirrors backend/app/models/persona.Persona.
type Persona struct {
	ID              int             `json:"id"`
	Name            string          `json:"name"`
	PPKEmployeeRate decimal.Decimal `json:"ppk_employee_rate"`
	PPKEmployerRate decimal.Decimal `json:"ppk_employer_rate"`
	CreatedAt       time.Time       `json:"created_at"`
}

// Sentinel errors so handlers can map to HTTP status codes without sniffing
// pg error text.
var (
	ErrNotFound     = errors.New("persona not found")
	ErrNameConflict = errors.New("persona name already exists")
)

// ConflictError reports the rows referencing a persona that block deletion.
// The .Details slice carries the same per-table phrasing the Python backend
// emits ("3 account(s)" etc).
type ConflictError struct {
	Name    string
	Details []string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("cannot delete persona %q: referenced by %v", e.Name, e.Details)
}

// Store is the persistence boundary for personas + owner-rename cascade.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `id, name, ppk_employee_rate, ppk_employer_rate, created_at`

// rollbackOnExit defers a Rollback whose error we deliberately swallow:
// Rollback after Commit is a no-op (returns ErrTxClosed), and the path that
// matters — failure before Commit — is what we want this guard for.
func rollbackOnExit(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

// List returns all personas ordered by name (matches Python's ORDER BY name).
func (s *Store) List(ctx context.Context) ([]Persona, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+selectColumns+` FROM personas ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("select personas: %w", err)
	}
	defer rows.Close()
	out := []Persona{}
	for rows.Next() {
		var p Persona
		if err := rows.Scan(&p.ID, &p.Name, &p.PPKEmployeeRate, &p.PPKEmployerRate, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan persona: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate personas: %w", err)
	}
	return out, nil
}

// Create inserts a new persona; ErrNameConflict on duplicate name.
func (s *Store) Create(ctx context.Context, name string, employee, employer decimal.Decimal) (*Persona, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO personas (name, ppk_employee_rate, ppk_employer_rate, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING `+selectColumns,
		name, employee, employer, time.Now().UTC(),
	)
	p, err := scanPersona(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrNameConflict
		}
		return nil, fmt.Errorf("insert persona: %w", err)
	}
	return p, nil
}

// Update applies a partial update. If newName is provided and differs, cascades
// the owner rename to the 5 owner-bearing tables in the same transaction.
func (s *Store) Update(
	ctx context.Context,
	id int,
	newName *string,
	newEmployee, newEmployer *decimal.Decimal,
) (*Persona, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer rollbackOnExit(ctx, tx)

	var current Persona
	err = tx.QueryRow(ctx, `SELECT `+selectColumns+` FROM personas WHERE id = $1 FOR UPDATE`, id).
		Scan(&current.ID, &current.Name, &current.PPKEmployeeRate, &current.PPKEmployerRate, &current.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select persona: %w", err)
	}

	oldName := current.Name
	if newName != nil {
		current.Name = *newName
	}
	if newEmployee != nil {
		current.PPKEmployeeRate = *newEmployee
	}
	if newEmployer != nil {
		current.PPKEmployerRate = *newEmployer
	}

	if _, err := tx.Exec(ctx, `
		UPDATE personas
		SET name = $1, ppk_employee_rate = $2, ppk_employer_rate = $3
		WHERE id = $4`,
		current.Name, current.PPKEmployeeRate, current.PPKEmployerRate, id,
	); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrNameConflict
		}
		return nil, fmt.Errorf("update persona: %w", err)
	}

	if newName != nil && *newName != oldName {
		// Cascade the owner rename to the 5 owner-bearing tables. Python does
		// these as separate UPDATEs; mirror exactly so any partial-success
		// behavior (there is none — single tx) is identical.
		for _, table := range [...]string{
			"accounts", "transactions", "salary_records", "debt_payments", "retirement_limits",
		} {
			if _, err := tx.Exec(ctx,
				`UPDATE `+table+` SET owner = $1 WHERE owner = $2`,
				current.Name, oldName,
			); err != nil {
				return nil, fmt.Errorf("cascade rename %s: %w", table, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &current, nil
}

// Delete removes a persona after verifying no owner-bearing rows reference it.
// On reference conflict returns *ConflictError with per-table counts.
func (s *Store) Delete(ctx context.Context, id int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer rollbackOnExit(ctx, tx)

	var name string
	if err := tx.QueryRow(ctx, `SELECT name FROM personas WHERE id = $1 FOR UPDATE`, id).Scan(&name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("select persona: %w", err)
	}

	var details []string
	for _, c := range [...]struct {
		table, label string
	}{
		{"accounts", "account(s)"},
		{"transactions", "transaction(s)"},
		{"salary_records", "salary record(s)"},
		{"debt_payments", "debt payment(s)"},
		{"retirement_limits", "retirement limit(s)"},
	} {
		var count int
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM `+c.table+` WHERE owner = $1`, name,
		).Scan(&count); err != nil {
			return fmt.Errorf("count %s: %w", c.table, err)
		}
		if count > 0 {
			details = append(details, fmt.Sprintf("%d %s", count, c.label))
		}
	}
	if len(details) > 0 {
		return &ConflictError{Name: name, Details: details}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM personas WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete persona: %w", err)
	}
	return tx.Commit(ctx)
}

func scanPersona(row pgx.Row) (*Persona, error) {
	var p Persona
	if err := row.Scan(&p.ID, &p.Name, &p.PPKEmployeeRate, &p.PPKEmployerRate, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr == nil {
		return false
	}
	return pgErr.Code == pgerrcode.UniqueViolation
}

// Package goals implements the /api/goals endpoints.
//
// Hard-delete semantics: DELETE removes the row, not a soft-delete flag.
// PUT distinguishes "field omitted" from "field explicitly set to null" so
// nullable foreign keys (account_id, category) behave like the Python PATCH
// using model_fields_set.
package goals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/dbutil"
)

// Goal mirrors backend/app/models/goal.Goal.
type Goal struct {
	ID                  int
	Name                string
	TargetAmount        decimal.Decimal
	TargetDate          time.Time
	CurrentAmount       decimal.Decimal
	MonthlyContribution decimal.Decimal
	IsCompleted         bool
	AccountID           *int
	Category            *string
	CreatedAt           time.Time
}

// ErrNotFound is returned when no row matches the supplied id.
var ErrNotFound = errors.New("goal not found")

// AccountMissingError is returned when create/update references a non-existent
// account_id; mapped to a 404 with a Python-shaped detail.
type AccountMissingError struct {
	AccountID int
}

func (e *AccountMissingError) Error() string {
	return fmt.Sprintf("account with id %d not found", e.AccountID)
}

// Store is the persistence boundary for goals + account-name lookup.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `
	id, name, target_amount, target_date, current_amount,
	monthly_contribution, is_completed, account_id, category, created_at
`

// List returns every goal ordered by target_date plus the account_name lookup
// map (same shape Python's _get_account_names returns).
func (s *Store) List(ctx context.Context) ([]Goal, map[int]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+selectColumns+` FROM goals ORDER BY target_date`,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("select goals: %w", err)
	}
	defer rows.Close()
	out := []Goal{}
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan goal: %w", err)
		}
		out = append(out, *g)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate goals: %w", err)
	}
	ids := []int{}
	for i := range out {
		if out[i].AccountID != nil {
			ids = append(ids, *out[i].AccountID)
		}
	}
	names, err := s.accountNames(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return out, names, nil
}

// Get returns a goal by id and the optional account name (empty if no account
// linked).
func (s *Store) Get(ctx context.Context, id int) (*Goal, string, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM goals WHERE id = $1`, id,
	)
	g, err := scanGoal(row)
	if err != nil {
		return nil, "", dbutil.MapErr(err, ErrNotFound, "select goal")
	}
	name, err := s.accountName(ctx, g.AccountID)
	if err != nil {
		return nil, "", err
	}
	return g, name, nil
}

// Create inserts a new goal. Returns AccountMissingError if account_id is set
// and the referenced account doesn't exist.
func (s *Store) Create(ctx context.Context, g *Goal) (*Goal, string, error) {
	if err := s.validateAccount(ctx, g.AccountID); err != nil {
		return nil, "", err
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO goals (
			name, target_amount, target_date, current_amount,
			monthly_contribution, is_completed, account_id, category, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+selectColumns,
		g.Name, g.TargetAmount, g.TargetDate, g.CurrentAmount,
		g.MonthlyContribution, g.IsCompleted, g.AccountID, g.Category,
		time.Now().UTC(),
	)
	created, err := scanGoal(row)
	if err != nil {
		return nil, "", fmt.Errorf("insert goal: %w", err)
	}
	name, err := s.accountName(ctx, created.AccountID)
	if err != nil {
		return nil, "", err
	}
	return created, name, nil
}

// UpdatePatch applies a sparse update. Only fields present in `set` are
// touched. account_id and category can be explicitly set to nil to clear them
// (caller signals this by including the key in `set` with a nil value).
type UpdatePatch struct {
	Name                *string
	TargetAmount        *decimal.Decimal
	TargetDate          *time.Time
	CurrentAmount       *decimal.Decimal
	MonthlyContribution *decimal.Decimal
	IsCompleted         *bool
	AccountIDSet        bool
	AccountID           *int
	CategorySet         bool
	Category            *string
}

// Update applies the patch and returns the refreshed goal + account name.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*Goal, string, error) {
	if p.AccountIDSet {
		if err := s.validateAccount(ctx, p.AccountID); err != nil {
			return nil, "", err
		}
	}
	g, _, err := s.Get(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if p.Name != nil {
		g.Name = *p.Name
	}
	if p.TargetAmount != nil {
		g.TargetAmount = *p.TargetAmount
	}
	if p.TargetDate != nil {
		g.TargetDate = *p.TargetDate
	}
	if p.CurrentAmount != nil {
		g.CurrentAmount = *p.CurrentAmount
	}
	if p.MonthlyContribution != nil {
		g.MonthlyContribution = *p.MonthlyContribution
	}
	if p.IsCompleted != nil {
		g.IsCompleted = *p.IsCompleted
	}
	if p.AccountIDSet {
		g.AccountID = p.AccountID
	}
	if p.CategorySet {
		g.Category = p.Category
	}

	row := s.pool.QueryRow(ctx, `
		UPDATE goals SET
			name = $1, target_amount = $2, target_date = $3, current_amount = $4,
			monthly_contribution = $5, is_completed = $6, account_id = $7, category = $8
		WHERE id = $9
		RETURNING `+selectColumns,
		g.Name, g.TargetAmount, g.TargetDate, g.CurrentAmount,
		g.MonthlyContribution, g.IsCompleted, g.AccountID, g.Category, id,
	)
	updated, err := scanGoal(row)
	if err != nil {
		// pgx.ErrNoRows here means the row was concurrently deleted between
		// the Get above and this UPDATE; surface as a 404 instead of 200ing
		// with stale in-memory state.
		return nil, "", dbutil.MapErr(err, ErrNotFound, "update goal")
	}
	name, err := s.accountName(ctx, updated.AccountID)
	if err != nil {
		return nil, "", err
	}
	return updated, name, nil
}

// Delete removes the goal (hard delete — Python does the same).
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM goals WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete goal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) validateAccount(ctx context.Context, accountID *int) error {
	if accountID == nil {
		return nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1)`, *accountID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check account: %w", err)
	}
	if !exists {
		return &AccountMissingError{AccountID: *accountID}
	}
	return nil
}

func (s *Store) accountNames(ctx context.Context, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, name FROM accounts WHERE id = ANY($1)`, ids,
	)
	if err != nil {
		return nil, fmt.Errorf("select account names: %w", err)
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan account name: %w", err)
		}
		out[id] = name
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account names: %w", err)
	}
	return out, nil
}

func (s *Store) accountName(ctx context.Context, accountID *int) (string, error) {
	if accountID == nil {
		return "", nil
	}
	var name string
	err := s.pool.QueryRow(ctx,
		`SELECT name FROM accounts WHERE id = $1`, *accountID,
	).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("select account name: %w", err)
	}
	return name, nil
}

func scanGoal(row pgx.Row) (*Goal, error) {
	var g Goal
	if err := row.Scan(
		&g.ID, &g.Name, &g.TargetAmount, &g.TargetDate, &g.CurrentAmount,
		&g.MonthlyContribution, &g.IsCompleted, &g.AccountID, &g.Category, &g.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &g, nil
}

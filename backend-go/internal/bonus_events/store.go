// Package bonusevents implements the /api/bonuses endpoints.
//
// Mirrors Python's bonus_events service: soft-delete via is_active filter,
// optional list filters (owner / date range / company), FX-derived amount_pln
// + fx_rate on every response. FX lookups go through internal/fx, which both
// reads from and writes to the fx_rates cache on miss.
package bonusevents

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// BonusEvent mirrors backend/app/models/bonus_event.BonusEvent.
// OwnerUserID is the owning household member; nil means jointly owned.
type BonusEvent struct {
	ID           int
	Date         time.Time
	Amount       decimal.Decimal
	Currency     string
	Type         string
	Company      string
	OwnerUserID  *int
	ContractType string
	Notes        *string
	IsActive     bool
	CreatedAt    time.Time
}

// ErrNotFound covers both "row doesn't exist" and "row exists but inactive"
// — the public contract is identical (a 404).
var ErrNotFound = errors.New("bonus event not found")

// ListFilter constrains the active rows returned by List.
type ListFilter struct {
	OwnerUserID *int
	DateFrom    *time.Time
	DateTo      *time.Time
	Company     *string
}

// Store is the persistence boundary.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `
	id, date, amount, currency, type, company, owner_user_id,
	contract_type, notes, is_active, created_at
`

// List returns active bonuses ordered by date desc and the distinct
// active-company list, matching Python's two-query shape.
func (s *Store) List(ctx context.Context, f ListFilter) ([]BonusEvent, []string, error) {
	conds := []string{"is_active = true"}
	args := []any{}
	if f.OwnerUserID != nil {
		args = append(args, *f.OwnerUserID)
		conds = append(conds, fmt.Sprintf("owner_user_id = $%d", len(args)))
	}
	if f.DateFrom != nil {
		args = append(args, *f.DateFrom)
		conds = append(conds, fmt.Sprintf("date >= $%d", len(args)))
	}
	if f.DateTo != nil {
		args = append(args, *f.DateTo)
		conds = append(conds, fmt.Sprintf("date <= $%d", len(args)))
	}
	if f.Company != nil {
		args = append(args, *f.Company)
		conds = append(conds, fmt.Sprintf("company = $%d", len(args)))
	}
	query := `SELECT ` + selectColumns + ` FROM bonus_events WHERE ` +
		strings.Join(conds, " AND ") + ` ORDER BY date DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("select bonuses: %w", err)
	}
	defer rows.Close()
	out := []BonusEvent{}
	for rows.Next() {
		b, err := scanBonus(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan bonus: %w", err)
		}
		out = append(out, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate bonuses: %w", err)
	}
	companies, err := s.activeCompanies(ctx)
	if err != nil {
		return nil, nil, err
	}
	return out, companies, nil
}

// Get returns the active bonus or ErrNotFound for missing/soft-deleted rows.
func (s *Store) Get(ctx context.Context, id int) (*BonusEvent, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM bonus_events WHERE id = $1`, id,
	)
	b, err := scanBonus(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select bonus: %w", err)
	}
	if !b.IsActive {
		return nil, ErrNotFound
	}
	return b, nil
}

// Create inserts a row.
func (s *Store) Create(ctx context.Context, b *BonusEvent) (*BonusEvent, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO bonus_events (
			date, amount, currency, type, company, owner_user_id,
			contract_type, notes, is_active, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, true, $9
		)
		RETURNING `+selectColumns,
		b.Date, b.Amount, b.Currency, b.Type, b.Company, b.OwnerUserID,
		b.ContractType, b.Notes, time.Now().UTC(),
	)
	return scanBonus(row)
}

// UpdatePatch is a sparse update.
type UpdatePatch struct {
	Date           *time.Time
	Amount         *decimal.Decimal
	Currency       *string
	Type           *string
	Company        *string
	OwnerUserIDSet bool
	OwnerUserID    *int
	ContractType   *string
	Notes          *string
}

// Update applies the patch and returns the refreshed bonus.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*BonusEvent, error) {
	b, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Date != nil {
		b.Date = *p.Date
	}
	if p.Amount != nil {
		b.Amount = *p.Amount
	}
	if p.Currency != nil {
		b.Currency = *p.Currency
	}
	if p.Type != nil {
		b.Type = *p.Type
	}
	if p.Company != nil {
		b.Company = *p.Company
	}
	if p.OwnerUserIDSet {
		b.OwnerUserID = p.OwnerUserID
	}
	if p.ContractType != nil {
		b.ContractType = *p.ContractType
	}
	if p.Notes != nil {
		b.Notes = p.Notes
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE bonus_events SET
			date = $1, amount = $2, currency = $3, type = $4, company = $5,
			owner_user_id = $6, contract_type = $7, notes = $8
		WHERE id = $9 AND is_active = true
		RETURNING `+selectColumns,
		b.Date, b.Amount, b.Currency, b.Type, b.Company, b.OwnerUserID,
		b.ContractType, b.Notes, id,
	)
	updated, err := scanBonus(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update bonus: %w", err)
	}
	return updated, nil
}

// Delete soft-deletes the row.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE bonus_events SET is_active = false WHERE id = $1 AND is_active = true`,
		id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete bonus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) activeCompanies(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT company FROM bonus_events
		 WHERE is_active = true ORDER BY company`,
	)
	if err != nil {
		return nil, fmt.Errorf("select active companies: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate companies: %w", err)
	}
	return out, nil
}

func scanBonus(row pgx.Row) (*BonusEvent, error) {
	var b BonusEvent
	if err := row.Scan(
		&b.ID, &b.Date, &b.Amount, &b.Currency, &b.Type, &b.Company,
		&b.OwnerUserID, &b.ContractType, &b.Notes, &b.IsActive, &b.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &b, nil
}

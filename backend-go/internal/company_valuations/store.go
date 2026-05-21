// Package companyvaluations implements the /api/company-valuations endpoints.
//
// All listing reads filter by is_active to honor the soft-delete contract.
// Update applies a range-integrity check (fmv_low <= fmv_per_share <= fmv_high)
// after the partial merge — same place the Python service catches it.
package companyvaluations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Valuation mirrors backend/app/models/company_valuation.CompanyValuation.
type Valuation struct {
	ID                     int
	Company                string
	Date                   time.Time
	Currency               string
	FMVPerShare            decimal.Decimal
	FMVLow                 *decimal.Decimal
	FMVHigh                *decimal.Decimal
	Source                 string
	CommonStockDiscountPct *decimal.Decimal
	Notes                  *string
	IsActive               bool
	CreatedAt              time.Time
}

// Sentinel errors mapped to HTTP status codes by the handler.
var (
	ErrNotFound  = errors.New("company valuation not found")
	ErrRangeLow  = errors.New("fmv_low cannot exceed fmv_per_share")
	ErrRangeHigh = errors.New("fmv_high cannot be below fmv_per_share")
	// ErrNoValuation signals "company has no active valuation" — a normal
	// outcome for GetLatestForCompany that consumers (equity grants) treat
	// as "skip paper-value computation", not as a failure.
	ErrNoValuation = errors.New("no valuation for company")
)

// Store is the persistence boundary.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `
	id, company, date, currency, fmv_per_share, fmv_low, fmv_high,
	source, common_stock_discount_pct, notes, is_active, created_at
`

// ListFilter constrains the active valuations returned by List.
type ListFilter struct {
	Company *string
}

// List returns valuations ordered by date desc + the distinct active-company
// list (sorted alphabetically) — same shape Python emits.
func (s *Store) List(ctx context.Context, f ListFilter) ([]Valuation, []string, error) {
	query := `SELECT ` + selectColumns + ` FROM company_valuations WHERE is_active = true`
	args := []any{}
	if f.Company != nil {
		query += ` AND company = $1`
		args = append(args, *f.Company)
	}
	query += ` ORDER BY date DESC`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("select valuations: %w", err)
	}
	defer rows.Close()
	out := []Valuation{}
	for rows.Next() {
		v, err := scanValuation(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan valuation: %w", err)
		}
		out = append(out, *v)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate valuations: %w", err)
	}
	companies, err := s.activeCompanies(ctx)
	if err != nil {
		return nil, nil, err
	}
	return out, companies, nil
}

// Get returns the active valuation or ErrNotFound for missing/soft-deleted rows.
func (s *Store) Get(ctx context.Context, id int) (*Valuation, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM company_valuations WHERE id = $1`, id,
	)
	v, err := scanValuation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select valuation: %w", err)
	}
	if !v.IsActive {
		return nil, ErrNotFound
	}
	return v, nil
}

// Create inserts a row.
func (s *Store) Create(ctx context.Context, v *Valuation) (*Valuation, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO company_valuations (
			company, date, currency, fmv_per_share, fmv_low, fmv_high,
			source, common_stock_discount_pct, notes, is_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10)
		RETURNING `+selectColumns,
		v.Company, v.Date, v.Currency, v.FMVPerShare, v.FMVLow, v.FMVHigh,
		v.Source, v.CommonStockDiscountPct, v.Notes, time.Now().UTC(),
	)
	created, err := scanValuation(row)
	if err != nil {
		return nil, fmt.Errorf("insert valuation: %w", err)
	}
	return created, nil
}

// UpdatePatch is a sparse update — every nil field means "leave alone".
type UpdatePatch struct {
	Company                *string
	Date                   *time.Time
	Currency               *string
	FMVPerShare            *decimal.Decimal
	FMVLow                 *decimal.Decimal
	FMVHigh                *decimal.Decimal
	Source                 *string
	CommonStockDiscountPct *decimal.Decimal
	Notes                  *string
}

// Update applies the patch and returns the refreshed row. Re-checks range
// integrity after merging so update-time violations (e.g. raising fmv_low
// above existing fmv_per_share) still 422 cleanly.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*Valuation, error) {
	v, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Company != nil {
		v.Company = *p.Company
	}
	if p.Date != nil {
		v.Date = *p.Date
	}
	if p.Currency != nil {
		v.Currency = *p.Currency
	}
	if p.FMVPerShare != nil {
		v.FMVPerShare = *p.FMVPerShare
	}
	if p.FMVLow != nil {
		v.FMVLow = p.FMVLow
	}
	if p.FMVHigh != nil {
		v.FMVHigh = p.FMVHigh
	}
	if p.Source != nil {
		v.Source = *p.Source
	}
	if p.CommonStockDiscountPct != nil {
		v.CommonStockDiscountPct = p.CommonStockDiscountPct
	}
	if p.Notes != nil {
		v.Notes = p.Notes
	}
	if v.FMVLow != nil && v.FMVLow.GreaterThan(v.FMVPerShare) {
		return nil, ErrRangeLow
	}
	if v.FMVHigh != nil && v.FMVHigh.LessThan(v.FMVPerShare) {
		return nil, ErrRangeHigh
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE company_valuations SET
			company = $1, date = $2, currency = $3, fmv_per_share = $4,
			fmv_low = $5, fmv_high = $6, source = $7,
			common_stock_discount_pct = $8, notes = $9
		WHERE id = $10 AND is_active = true
		RETURNING `+selectColumns,
		v.Company, v.Date, v.Currency, v.FMVPerShare, v.FMVLow, v.FMVHigh,
		v.Source, v.CommonStockDiscountPct, v.Notes, id,
	)
	updated, err := scanValuation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update valuation: %w", err)
	}
	return updated, nil
}

// GetLatestForCompany returns the most recent active valuation for a company
// on/before onDate. Returns ErrNoValuation when no valuation exists — callers
// (equity grants) treat that as a normal "skip paper-value computation"
// branch rather than a failure.
func (s *Store) GetLatestForCompany(ctx context.Context, company string, onDate *time.Time) (*Valuation, error) {
	query := `SELECT ` + selectColumns + ` FROM company_valuations
		WHERE company = $1 AND is_active = true`
	args := []any{company}
	if onDate != nil {
		query += ` AND date <= $2`
		args = append(args, *onDate)
	}
	query += ` ORDER BY date DESC LIMIT 1`
	row := s.pool.QueryRow(ctx, query, args...)
	v, err := scanValuation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoValuation
		}
		return nil, fmt.Errorf("select latest valuation: %w", err)
	}
	return v, nil
}

// Delete soft-deletes the row (matches Python's soft_delete helper).
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE company_valuations SET is_active = false WHERE id = $1 AND is_active = true`,
		id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete valuation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) activeCompanies(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT company FROM company_valuations
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

func scanValuation(row pgx.Row) (*Valuation, error) {
	var v Valuation
	if err := row.Scan(
		&v.ID, &v.Company, &v.Date, &v.Currency, &v.FMVPerShare,
		&v.FMVLow, &v.FMVHigh, &v.Source, &v.CommonStockDiscountPct,
		&v.Notes, &v.IsActive, &v.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &v, nil
}

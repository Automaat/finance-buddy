// Package salaries implements /api/salaries.
//
// Mirrors backend/app/services/salary_records.py: full CRUD with soft-delete,
// optional filters (owner, date_from, date_to, company), a current-salary
// snapshot per persona, and an inflation_context per persona computed via
// the CPI index.
package salaries

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

// SalaryRecord mirrors backend/app/models/salary_record.SalaryRecord.
type SalaryRecord struct {
	ID           int
	Date         time.Time
	GrossAmount  decimal.Decimal
	ContractType string
	Company      string
	Owner        string
	IsActive     bool
	CreatedAt    time.Time
}

// ErrNotFound is returned for missing or soft-deleted rows.
var ErrNotFound = errors.New("salary record not found")

// ErrDuplicate is returned when (owner, date) already has an active record.
var ErrDuplicate = errors.New("salary record duplicate")

// ListFilter narrows the active rows.
type ListFilter struct {
	Owner    *string
	DateFrom *time.Time
	DateTo   *time.Time
	Company  *string
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
	id, date, gross_amount, contract_type, company, owner, is_active, created_at
`

// List returns active rows ordered by date desc + the distinct active-company
// list (unfiltered, so the dropdown stays usable across filters).
func (s *Store) List(ctx context.Context, f ListFilter) ([]SalaryRecord, []string, error) {
	conds := []string{"is_active = true"}
	args := []any{}
	if f.Owner != nil {
		args = append(args, *f.Owner)
		conds = append(conds, fmt.Sprintf("owner = $%d", len(args)))
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
	query := `SELECT ` + selectColumns + ` FROM salary_records WHERE ` +
		strings.Join(conds, " AND ") + ` ORDER BY date DESC`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("select salary_records: %w", err)
	}
	defer rows.Close()
	out := []SalaryRecord{}
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan salary record: %w", err)
		}
		out = append(out, *r)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate salary records: %w", err)
	}
	companies, err := s.activeCompanies(ctx)
	if err != nil {
		return nil, nil, err
	}
	return out, companies, nil
}

// Get returns the active record or ErrNotFound.
func (s *Store) Get(ctx context.Context, id int) (*SalaryRecord, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM salary_records WHERE id = $1`, id,
	)
	r, err := scanRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select salary record: %w", err)
	}
	if !r.IsActive {
		return nil, ErrNotFound
	}
	return r, nil
}

// Create inserts a row. Returns ErrDuplicate when (owner, date) already
// has an active record — matches Python's 409 contract.
func (s *Store) Create(ctx context.Context, r *SalaryRecord) (*SalaryRecord, error) {
	var existing int
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM salary_records
		WHERE owner = $1 AND date = $2 AND is_active = true
		LIMIT 1`,
		r.Owner, r.Date,
	).Scan(&existing)
	if err == nil {
		return nil, ErrDuplicate
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check duplicate salary: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO salary_records (
			date, gross_amount, contract_type, company, owner, is_active, created_at
		) VALUES ($1, $2, $3, $4, $5, true, $6)
		RETURNING `+selectColumns,
		r.Date, r.GrossAmount, r.ContractType, r.Company, r.Owner, time.Now().UTC(),
	)
	return scanRecord(row)
}

// UpdatePatch is the sparse update set; nil = no-op.
type UpdatePatch struct {
	Date         *time.Time
	GrossAmount  *decimal.Decimal
	ContractType *string
	Company      *string
	Owner        *string
}

// Update applies the patch and returns the refreshed row.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*SalaryRecord, error) {
	r, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Date != nil {
		r.Date = *p.Date
	}
	if p.GrossAmount != nil {
		r.GrossAmount = *p.GrossAmount
	}
	if p.ContractType != nil {
		r.ContractType = *p.ContractType
	}
	if p.Company != nil {
		r.Company = *p.Company
	}
	if p.Owner != nil {
		r.Owner = *p.Owner
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE salary_records SET
			date = $1, gross_amount = $2, contract_type = $3,
			company = $4, owner = $5
		WHERE id = $6 AND is_active = true
		RETURNING `+selectColumns,
		r.Date, r.GrossAmount, r.ContractType, r.Company, r.Owner, id,
	)
	updated, err := scanRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update salary record: %w", err)
	}
	return updated, nil
}

// Delete soft-deletes.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE salary_records SET is_active = false WHERE id = $1 AND is_active = true`,
		id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete salary record: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CurrentSalaryByPersona returns the latest active gross_amount on/before
// asOfDate for each persona name in the input list. Personas without any
// matching record are absent from the map (and rendered as null in the
// response).
func (s *Store) CurrentSalaryByPersona(
	ctx context.Context, personaNames []string, asOfDate time.Time,
) (map[string]decimal.Decimal, error) {
	if len(personaNames) == 0 {
		return map[string]decimal.Decimal{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (owner) owner, gross_amount
		FROM salary_records
		WHERE is_active = true AND owner = ANY($1) AND date <= $2
		ORDER BY owner, date DESC`,
		personaNames, asOfDate,
	)
	if err != nil {
		return nil, fmt.Errorf("select current salaries: %w", err)
	}
	defer rows.Close()
	out := map[string]decimal.Decimal{}
	for rows.Next() {
		var owner string
		var amount decimal.Decimal
		if err := rows.Scan(&owner, &amount); err != nil {
			return nil, fmt.Errorf("scan current salary: %w", err)
		}
		out[owner] = amount
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate current salaries: %w", err)
	}
	return out, nil
}

// RecentTwoByPersona returns up to two most-recent active records per owner
// on/before asOfDate, ordered by date desc per owner. Used to compute
// inflation_context (current vs previous salary).
func (s *Store) RecentTwoByPersona(
	ctx context.Context, personaNames []string, asOfDate time.Time,
) (map[string][]SalaryRecord, error) {
	if len(personaNames) == 0 {
		return map[string][]SalaryRecord{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
			SELECT id, date, gross_amount, contract_type, company, owner, is_active, created_at,
			       ROW_NUMBER() OVER (PARTITION BY owner ORDER BY date DESC) AS rn
			FROM salary_records
			WHERE is_active = true AND owner = ANY($1) AND date <= $2
		)
		SELECT id, date, gross_amount, contract_type, company, owner, is_active, created_at
		FROM ranked WHERE rn <= 2
		ORDER BY owner, date DESC`,
		personaNames, asOfDate,
	)
	if err != nil {
		return nil, fmt.Errorf("select recent salaries: %w", err)
	}
	defer rows.Close()
	out := map[string][]SalaryRecord{}
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recent salary: %w", err)
		}
		out[r.Owner] = append(out[r.Owner], *r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent salaries: %w", err)
	}
	return out, nil
}

// ActivePersonaNames returns the persona names sorted alphabetically.
func (s *Store) ActivePersonaNames(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT name FROM personas ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("select persona names: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, fmt.Errorf("scan persona name: %w", err)
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate persona names: %w", err)
	}
	return out, nil
}

func (s *Store) activeCompanies(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT company FROM salary_records
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

func scanRecord(row pgx.Row) (*SalaryRecord, error) {
	var r SalaryRecord
	if err := row.Scan(
		&r.ID, &r.Date, &r.GrossAmount, &r.ContractType,
		&r.Company, &r.Owner, &r.IsActive, &r.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

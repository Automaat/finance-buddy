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

	"github.com/Automaat/finance-buddy/backend-go/internal/dbutil"
)

// SalaryRecord mirrors backend/app/models/salary_record.SalaryRecord.
// OwnerUserID is the owning household member; nil means jointly owned.
type SalaryRecord struct {
	ID           int
	Date         time.Time
	GrossAmount  decimal.Decimal
	ContractType string
	Company      string
	OwnerUserID  *int
	IsActive     bool
	CreatedAt    time.Time
}

// ErrNotFound is returned for missing or soft-deleted rows.
var ErrNotFound = errors.New("salary record not found")

// ErrDuplicate is returned when (owner_user_id, date) already has an
// active record.
var ErrDuplicate = errors.New("salary record duplicate")

// ListFilter narrows the active rows.
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
	id, date, gross_amount, contract_type, company, owner_user_id, is_active, created_at
`

// List returns active rows ordered by date desc + the distinct active-company
// list (unfiltered, so the dropdown stays usable across filters).
func (s *Store) List(ctx context.Context, f ListFilter) ([]SalaryRecord, []string, error) {
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
		return nil, dbutil.MapErr(err, ErrNotFound, "select salary record")
	}
	if !r.IsActive {
		return nil, ErrNotFound
	}
	return r, nil
}

// Create inserts a row. Returns ErrDuplicate when (owner_user_id, date)
// already has an active record — matches Python's 409 contract.
func (s *Store) Create(ctx context.Context, r *SalaryRecord) (*SalaryRecord, error) {
	var existing int
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM salary_records
		WHERE owner_user_id IS NOT DISTINCT FROM $1 AND date = $2 AND is_active = true
		LIMIT 1`,
		r.OwnerUserID, r.Date,
	).Scan(&existing)
	if err == nil {
		return nil, ErrDuplicate
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check duplicate salary: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO salary_records (
			date, gross_amount, contract_type, company, owner_user_id,
			is_active, created_at
		) VALUES (
			$1, $2, $3, $4, $5, true, $6
		)
		RETURNING `+selectColumns,
		r.Date, r.GrossAmount, r.ContractType, r.Company, r.OwnerUserID, time.Now().UTC(),
	)
	return scanRecord(row)
}

// UpdatePatch is the sparse update set; nil = no-op.
type UpdatePatch struct {
	Date           *time.Time
	GrossAmount    *decimal.Decimal
	ContractType   *string
	Company        *string
	OwnerUserIDSet bool
	OwnerUserID    *int
}

// Update applies the patch and returns the refreshed row.
// Mirrors Python's duplicate check: after merging the patch, fail with
// ErrDuplicate if another active record exists for the same (owner, date).
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
	if p.OwnerUserIDSet {
		r.OwnerUserID = p.OwnerUserID
	}
	var conflict int
	err = s.pool.QueryRow(ctx, `
		SELECT id FROM salary_records
		WHERE owner_user_id IS NOT DISTINCT FROM $1 AND date = $2
		  AND id <> $3 AND is_active = true
		LIMIT 1`,
		r.OwnerUserID, r.Date, id,
	).Scan(&conflict)
	if err == nil {
		return nil, ErrDuplicate
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check duplicate on update: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE salary_records SET
			date = $1, gross_amount = $2, contract_type = $3, company = $4,
			owner_user_id = $5
		WHERE id = $6 AND is_active = true
		RETURNING `+selectColumns,
		r.Date, r.GrossAmount, r.ContractType, r.Company, r.OwnerUserID, id,
	)
	updated, err := scanRecord(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "update salary record")
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

// CurrentSalaryByUser returns the latest active gross_amount on/before
// asOfDate for each owner_user_id in the input list. Users without any
// matching record are absent from the map (and rendered as null in the
// response).
func (s *Store) CurrentSalaryByUser(
	ctx context.Context, userIDs []int, asOfDate time.Time,
) (map[int]decimal.Decimal, error) {
	if len(userIDs) == 0 {
		return map[int]decimal.Decimal{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (owner_user_id) owner_user_id, gross_amount
		FROM salary_records
		WHERE is_active = true AND owner_user_id = ANY($1) AND date <= $2
		ORDER BY owner_user_id, date DESC`,
		userIDs, asOfDate,
	)
	if err != nil {
		return nil, fmt.Errorf("select current salaries: %w", err)
	}
	defer rows.Close()
	out := map[int]decimal.Decimal{}
	for rows.Next() {
		var ownerID int
		var amount decimal.Decimal
		if err := rows.Scan(&ownerID, &amount); err != nil {
			return nil, fmt.Errorf("scan current salary: %w", err)
		}
		out[ownerID] = amount
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate current salaries: %w", err)
	}
	return out, nil
}

// RecentTwoByUser returns up to two most-recent active records per
// owner_user_id on/before asOfDate, ordered by date desc per owner. Used to
// compute inflation_context (current vs previous salary).
func (s *Store) RecentTwoByUser(
	ctx context.Context, userIDs []int, asOfDate time.Time,
) (map[int][]SalaryRecord, error) {
	if len(userIDs) == 0 {
		return map[int][]SalaryRecord{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
			SELECT id, date, gross_amount, contract_type, company, owner_user_id, is_active, created_at,
			       ROW_NUMBER() OVER (PARTITION BY owner_user_id ORDER BY date DESC) AS rn
			FROM salary_records
			WHERE is_active = true AND owner_user_id = ANY($1) AND date <= $2
		)
		SELECT id, date, gross_amount, contract_type, company, owner_user_id, is_active, created_at
		FROM ranked WHERE rn <= 2
		ORDER BY owner_user_id, date DESC`,
		userIDs, asOfDate,
	)
	if err != nil {
		return nil, fmt.Errorf("select recent salaries: %w", err)
	}
	defer rows.Close()
	out := map[int][]SalaryRecord{}
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recent salary: %w", err)
		}
		if r.OwnerUserID != nil {
			out[*r.OwnerUserID] = append(out[*r.OwnerUserID], *r)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent salaries: %w", err)
	}
	return out, nil
}

// ActiveOwnerUserIDs returns every household member's user id, ordered by
// username — the set the current-salary and inflation-context maps are
// keyed by.
func (s *Store) ActiveOwnerUserIDs(ctx context.Context) ([]int, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("select owner user ids: %w", err)
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan owner user id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owner user ids: %w", err)
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
		&r.Company, &r.OwnerUserID, &r.IsActive, &r.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

// Package retirement implements /api/retirement — yearly stats per
// IKE/IKZE wrapper, PPK aggregates, limit upsert, and PPK contribution
// generation that writes employee + employer transactions atomically.
package retirement

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

// Sentinel errors.
var (
	ErrNoSalary           = errors.New("no salary record for owner")
	ErrPersonaNotFound    = errors.New("persona not found")
	ErrNoPPKAccount       = errors.New("no active PPK account for owner")
	ErrContributionsExist = errors.New("contributions already exist for month")
)

// errNoLimit is the internal sentinel for "no retirement_limit row matches
// (year, wrapper, owner)" — unexported because callers use the (nil, err)
// branch directly.
var errNoLimit = errors.New("no limit configured for wrapper")

// Limit mirrors backend/app/models/retirement_limit.RetirementLimit.
type Limit struct {
	ID             int
	Year           int
	AccountWrapper string
	Owner          string
	LimitAmount    decimal.Decimal
	Notes          *string
}

// Store is the persistence boundary.
type Store struct{ pool *pgxpool.Pool }

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// PersonaNames returns persona names alphabetically.
func (s *Store) PersonaNames(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT name FROM personas ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("personas: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, fmt.Errorf("scan persona: %w", err)
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate personas: %w", err)
	}
	return out, nil
}

// AccountIDsForWrapper returns the active account IDs for one wrapper+owner.
func (s *Store) AccountIDsForWrapper(ctx context.Context, wrapper, owner string) ([]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM accounts
		WHERE account_wrapper = $1 AND owner = $2 AND is_active = true`,
		wrapper, owner,
	)
	if err != nil {
		return nil, fmt.Errorf("account ids: %w", err)
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan account id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account ids: %w", err)
	}
	return out, nil
}

// ContributionTotals aggregates active transaction amounts for one year +
// account set, broken down by transaction_type. typed_sum < total -> the
// remainder is untyped and the handler attributes it to the employee bucket
// (matches Python's behavior).
type ContributionTotals struct {
	Total      decimal.Decimal
	Employee   decimal.Decimal
	Employer   decimal.Decimal
	Government decimal.Decimal
}

// YearlyContributions aggregates transactions for one year limited to one
// year and one set of accounts.
func (s *Store) YearlyContributions(ctx context.Context, year int, accountIDs []int) (ContributionTotals, error) {
	if len(accountIDs) == 0 {
		return ContributionTotals{}, nil
	}
	row := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount), 0) AS total,
			COALESCE(SUM(CASE WHEN transaction_type = 'employee' THEN amount ELSE 0 END), 0) AS employee,
			COALESCE(SUM(CASE WHEN transaction_type = 'employer' THEN amount ELSE 0 END), 0) AS employer
		FROM transactions
		WHERE account_id = ANY($1)
		  AND EXTRACT(year FROM date) = $2
		  AND is_active = true`,
		accountIDs, year,
	)
	var t ContributionTotals
	if err := row.Scan(&t.Total, &t.Employee, &t.Employer); err != nil {
		return ContributionTotals{}, fmt.Errorf("yearly contributions: %w", err)
	}
	return t, nil
}

// PPKContributions aggregates all-time PPK transactions including the
// government bucket. Matches Python's three-typed + untyped-as-employee
// logic.
func (s *Store) PPKContributions(ctx context.Context, accountIDs []int) (ContributionTotals, error) {
	if len(accountIDs) == 0 {
		return ContributionTotals{}, nil
	}
	row := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE
				WHEN transaction_type IN ('employee', 'employer', 'government') OR transaction_type IS NULL
				THEN amount ELSE 0
			END), 0) AS total,
			COALESCE(SUM(CASE WHEN transaction_type = 'employee' THEN amount ELSE 0 END), 0) AS employee,
			COALESCE(SUM(CASE WHEN transaction_type = 'employer' THEN amount ELSE 0 END), 0) AS employer,
			COALESCE(SUM(CASE WHEN transaction_type = 'government' THEN amount ELSE 0 END), 0) AS government
		FROM transactions
		WHERE account_id = ANY($1) AND is_active = true`,
		accountIDs,
	)
	var t ContributionTotals
	if err := row.Scan(&t.Total, &t.Employee, &t.Employer, &t.Government); err != nil {
		return ContributionTotals{}, fmt.Errorf("ppk contributions: %w", err)
	}
	return t, nil
}

// LatestSnapshotValueSum returns the sum of snapshot values for accountIDs
// at the most recent snapshot date that touches any of them.
func (s *Store) LatestSnapshotValueSum(ctx context.Context, accountIDs []int) (decimal.Decimal, error) {
	if len(accountIDs) == 0 {
		return decimal.Zero, nil
	}
	row := s.pool.QueryRow(ctx, `
		WITH latest AS (
			SELECT MAX(s.date) AS d
			FROM snapshots s
			JOIN snapshot_values sv ON sv.snapshot_id = s.id
			WHERE sv.account_id = ANY($1)
		)
		SELECT COALESCE(SUM(sv.value), 0)
		FROM snapshot_values sv
		JOIN snapshots s ON s.id = sv.snapshot_id
		JOIN latest l ON s.date = l.d
		WHERE sv.account_id = ANY($1)`,
		accountIDs,
	)
	var v decimal.Decimal
	if err := row.Scan(&v); err != nil {
		return decimal.Zero, fmt.Errorf("latest ppk value: %w", err)
	}
	return v, nil
}

// LimitsForYear returns all RetirementLimit rows for a year, ordered by id.
func (s *Store) LimitsForYear(ctx context.Context, year int) ([]Limit, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, year, account_wrapper, owner, limit_amount, notes
		FROM retirement_limits WHERE year = $1
		ORDER BY id`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("limits: %w", err)
	}
	defer rows.Close()
	out := []Limit{}
	for rows.Next() {
		var l Limit
		if err := rows.Scan(&l.ID, &l.Year, &l.AccountWrapper, &l.Owner, &l.LimitAmount, &l.Notes); err != nil {
			return nil, fmt.Errorf("scan limit: %w", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate limits: %w", err)
	}
	return out, nil
}

// LimitFor returns the Limit row for (year, wrapper, owner). errNoLimit
// when the row is absent — callers handle that as "no limit configured".
func (s *Store) LimitFor(ctx context.Context, year int, wrapper, owner string) (*Limit, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, year, account_wrapper, owner, limit_amount, notes
		FROM retirement_limits
		WHERE year = $1 AND account_wrapper = $2 AND owner = $3`,
		year, wrapper, owner,
	)
	var l Limit
	if err := row.Scan(&l.ID, &l.Year, &l.AccountWrapper, &l.Owner, &l.LimitAmount, &l.Notes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNoLimit
		}
		return nil, fmt.Errorf("limit for: %w", err)
	}
	return &l, nil
}

// LimitConfigured wraps LimitFor with the "no row" branch surfaced as a
// (nil, nil) pair — callers don't need to import the sentinel.
func (s *Store) LimitConfigured(ctx context.Context, year int, wrapper, owner string) (*Limit, bool, error) {
	l, err := s.LimitFor(ctx, year, wrapper, owner)
	if err != nil {
		if errors.Is(err, errNoLimit) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return l, true, nil
}

// UpsertLimit creates the (year, wrapper, owner) row or updates limit_amount
// + notes when it exists. Mirrors Python's update_limit fall-through.
func (s *Store) UpsertLimit(ctx context.Context, year int, wrapper, owner string, amount decimal.Decimal, notes *string) (*Limit, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO retirement_limits (year, account_wrapper, owner, limit_amount, notes)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (year, account_wrapper, owner) DO UPDATE
		SET limit_amount = EXCLUDED.limit_amount,
		    notes = COALESCE(EXCLUDED.notes, retirement_limits.notes)
		RETURNING id, year, account_wrapper, owner, limit_amount, notes`,
		year, wrapper, owner, amount, notes,
	)
	var l Limit
	if err := row.Scan(&l.ID, &l.Year, &l.AccountWrapper, &l.Owner, &l.LimitAmount, &l.Notes); err != nil {
		return nil, fmt.Errorf("upsert limit: %w", err)
	}
	return &l, nil
}

// CurrentSalaryFor returns the latest active gross_amount on/before asOfDate
// for one owner.
func (s *Store) CurrentSalaryFor(ctx context.Context, owner string, asOfDate time.Time) (decimal.Decimal, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT gross_amount FROM salary_records
		WHERE owner = $1 AND is_active = true AND date <= $2
		ORDER BY date DESC LIMIT 1`,
		owner, asOfDate,
	)
	var v decimal.Decimal
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return decimal.Zero, ErrNoSalary
		}
		return decimal.Zero, fmt.Errorf("current salary: %w", err)
	}
	return v, nil
}

// PersonaPPKRates returns (employee_rate, employer_rate) percentages from
// the personas table or ErrPersonaNotFound.
func (s *Store) PersonaPPKRates(ctx context.Context, owner string) (decimal.Decimal, decimal.Decimal, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT ppk_employee_rate, ppk_employer_rate
		FROM personas WHERE name = $1`,
		owner,
	)
	var employee, employer decimal.Decimal
	if err := row.Scan(&employee, &employer); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return decimal.Zero, decimal.Zero, ErrPersonaNotFound
		}
		return decimal.Zero, decimal.Zero, fmt.Errorf("ppk rates: %w", err)
	}
	return employee, employer, nil
}

// ActivePPKAccountForOwner returns the receiving PPK account or ErrNoPPKAccount.
func (s *Store) ActivePPKAccountForOwner(ctx context.Context, owner string) (int, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id FROM accounts
		WHERE account_wrapper = 'PPK' AND owner = $1
		  AND is_active = true AND receives_contributions = true
		LIMIT 1`,
		owner,
	)
	var id int
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNoPPKAccount
		}
		return 0, fmt.Errorf("ppk account: %w", err)
	}
	return id, nil
}

// PPKContribution is the writeback shape — two transactions per call.
type PPKContribution struct {
	AccountID   int
	EmployeeAmt decimal.Decimal
	EmployerAmt decimal.Decimal
	Date        time.Time
	Owner       string
}

// InsertPPKContributions writes the employee + employer transactions
// atomically. ErrContributionsExist when a row already exists for the
// (account_id, date, employee|employer) tuple — guarded both by a
// pre-check and by a UniqueViolation catch.
func (s *Store) InsertPPKContributions(ctx context.Context, c PPKContribution) ([]int, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin ppk tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	var existing int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM transactions
		WHERE account_id = $1 AND date = $2
		  AND transaction_type IN ('employee', 'employer')
		  AND is_active = true`,
		c.AccountID, c.Date,
	).Scan(&existing)
	if err != nil {
		return nil, fmt.Errorf("count ppk existing: %w", err)
	}
	if existing > 0 {
		return nil, ErrContributionsExist
	}
	now := time.Now().UTC()
	var empID, emprID int
	if err := tx.QueryRow(ctx, `
		INSERT INTO transactions (account_id, amount, date, owner, transaction_type, is_active, created_at)
		VALUES ($1, $2, $3, $4, 'employee', true, $5)
		RETURNING id`,
		c.AccountID, c.EmployeeAmt, c.Date, c.Owner, now,
	).Scan(&empID); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrContributionsExist
		}
		return nil, fmt.Errorf("insert employee txn: %w", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO transactions (account_id, amount, date, owner, transaction_type, is_active, created_at)
		VALUES ($1, $2, $3, $4, 'employer', true, $5)
		RETURNING id`,
		c.AccountID, c.EmployerAmt, c.Date, c.Owner, now,
	).Scan(&emprID); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrContributionsExist
		}
		return nil, fmt.Errorf("insert employer txn: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ppk tx: %w", err)
	}
	committed = true
	return []int{empID, emprID}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr == nil {
		return false
	}
	return pgErr.Code == pgerrcode.UniqueViolation
}

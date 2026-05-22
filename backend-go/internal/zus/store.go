package zus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// errNoSalary marks the absence of any active salary record for a given
// owner. Lets latestSalary return a sentinel error instead of (nil, nil)
// which the nilnil lint forbids. Unexported — internal package signal,
// not part of the package API.
var errNoSalary = errors.New("no salary record for owner")

// Store reads salary records + users + app_config for prefill.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// PrefillData bundles everything prefill needs in a single struct so the
// handler can compose the response without per-field branching.
// OwnerUserID is the resolved household member; nil means jointly owned.
type PrefillData struct {
	OwnerUserID               *int
	BirthDate                 *time.Time
	RetirementAge             int
	CurrentGrossMonthlySalary *float64
	WorkStartYear             *int
	YearlySalaryHistory       map[int]float64
}

// LoadPrefill fetches the data Python's get_zus_prefill assembles:
// - config (birth_date + retirement_age)
// - resolved owner (passed-in owner_user_id or first user alphabetically)
// - latest active salary for owner -> current_gross_monthly_salary
// - all active salaries for owner -> yearly history (latest record per year)
func (s *Store) LoadPrefill(ctx context.Context, ownerHint *int) (PrefillData, error) {
	data := PrefillData{RetirementAge: 65, YearlySalaryHistory: map[int]float64{}}

	birth, ra, err := s.loadConfig(ctx)
	if err != nil {
		return data, err
	}
	if birth != nil {
		data.BirthDate = birth
	}
	if ra != 0 {
		data.RetirementAge = ra
	}

	var owner *int
	if ownerHint != nil {
		owner = ownerHint
	} else {
		first, found, err := s.firstUser(ctx)
		if err != nil {
			return data, err
		}
		if !found {
			return data, nil
		}
		owner = &first
	}
	data.OwnerUserID = owner

	current, err := s.latestSalary(ctx, owner)
	if err != nil && !errors.Is(err, errNoSalary) {
		return data, err
	}
	if current != nil {
		data.CurrentGrossMonthlySalary = current
	}

	yearly, workStart, err := s.yearlyHistory(ctx, owner)
	if err != nil {
		return data, err
	}
	data.YearlySalaryHistory = yearly
	if workStart != nil {
		data.WorkStartYear = workStart
	}
	return data, nil
}

func (s *Store) loadConfig(ctx context.Context) (*time.Time, int, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT birth_date, retirement_age FROM app_config LIMIT 1`,
	)
	var birth *time.Time
	var ra int
	if err := row.Scan(&birth, &ra); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("select app_config: %w", err)
	}
	return birth, ra, nil
}

// firstUser returns the id of the first named user ordered by name. The
// bool is false when no named users exist. Unnamed accounts (e.g. admin)
// are skipped — the ZUS prefill owner picker only covers household members.
func (s *Store) firstUser(ctx context.Context) (int, bool, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id FROM users WHERE name IS NOT NULL ORDER BY name LIMIT 1`,
	)
	var id int
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("select first user: %w", err)
	}
	return id, true, nil
}

func (s *Store) latestSalary(ctx context.Context, ownerUserID *int) (*float64, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT gross_amount FROM salary_records
		WHERE owner_user_id IS NOT DISTINCT FROM $1 AND is_active = true
		ORDER BY date DESC LIMIT 1`,
		ownerUserID,
	)
	var v float64
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNoSalary
		}
		return nil, fmt.Errorf("select latest salary: %w", err)
	}
	return &v, nil
}

func (s *Store) yearlyHistory(ctx context.Context, ownerUserID *int) (map[int]float64, *int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT date, gross_amount FROM salary_records
		WHERE owner_user_id IS NOT DISTINCT FROM $1 AND is_active = true
		ORDER BY date`,
		ownerUserID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("select salary history: %w", err)
	}
	defer rows.Close()
	yearly := map[int]float64{}
	for rows.Next() {
		var d time.Time
		var gross float64
		if err := rows.Scan(&d, &gross); err != nil {
			return nil, nil, fmt.Errorf("scan salary history: %w", err)
		}
		// "latest record per year" — keep overwriting; we ORDER BY date so
		// the last write wins (matches Python's dict assignment loop).
		yearly[d.Year()] = gross * 12
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate salary history: %w", err)
	}
	if len(yearly) == 0 {
		return yearly, nil, nil
	}
	minYear := 0
	for y := range yearly {
		if minYear == 0 || y < minYear {
			minYear = y
		}
	}
	return yearly, &minYear, nil
}

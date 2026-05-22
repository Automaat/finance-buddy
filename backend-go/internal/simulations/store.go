package simulations

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Default 2026 contribution limits — used when retirement_limits has no row.
const (
	defaultIKELimit  = 28260.0
	defaultIKZELimit = 11304.0
)

// Store handles the DB reads for prefill + the retirement-limit lookup.
type Store struct{ pool *pgxpool.Pool }

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// LimitForYear ports get_limit_for_year — retirement_limits row or the
// wrapper default.
func (s *Store) LimitForYear(ctx context.Context, year int, wrapper string, ownerUserID *int) (float64, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT limit_amount FROM retirement_limits
		WHERE year = $1 AND account_wrapper = $2
		  AND owner_user_id IS NOT DISTINCT FROM $3`,
		year, wrapper, ownerUserID,
	)
	var amount float64
	if err := row.Scan(&amount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			switch wrapper {
			case "IKE":
				return defaultIKELimit, nil
			case "IKZE":
				return defaultIKZELimit, nil
			default:
				return 0, nil
			}
		}
		return 0, fmt.Errorf("limit for year: %w", err)
	}
	return amount, nil
}

// PrefillData is everything GET /api/simulations/prefill needs. The
// owner-keyed maps are keyed by the owner_user_id rendered as a string
// (JSON object keys must be strings); Balances composite keys are
// "ike_<id>" / "ikze_<id>".
type PrefillData struct {
	CurrentAge      int
	RetirementAge   int
	Owners          []userRow
	Balances        map[string]float64
	PPKRates        map[string]map[string]float64
	MonthlySalaries map[string]*float64
	PPKBalances     map[string]float64
}

// LoadPrefill assembles the prefill payload.
func (s *Store) LoadPrefill(ctx context.Context) (PrefillData, error) {
	data := PrefillData{
		RetirementAge:   67,
		Balances:        map[string]float64{},
		PPKRates:        map[string]map[string]float64{},
		MonthlySalaries: map[string]*float64{},
		PPKBalances:     map[string]float64{},
	}
	birth, retirementAge, err := s.loadConfig(ctx)
	if err != nil {
		return data, err
	}
	if retirementAge != 0 {
		data.RetirementAge = retirementAge
	}
	data.CurrentAge = ageFromBirth(birth)

	users, err := s.loadUsers(ctx)
	if err != nil {
		return data, err
	}
	data.Owners = users
	for _, u := range users {
		key := strconv.Itoa(u.id)
		data.Balances["ike_"+key] = 0
		data.Balances["ikze_"+key] = 0
		data.PPKBalances[key] = 0
		data.PPKRates[key] = map[string]float64{
			"employee": u.employeeRate,
			"employer": u.employerRate,
		}
		data.MonthlySalaries[key] = nil
	}
	if err := s.fillLatestSalaries(ctx, users, data.MonthlySalaries); err != nil {
		return data, err
	}
	if err := s.fillWrapperBalances(ctx, data); err != nil {
		return data, err
	}
	return data, nil
}

// userRow is one household member with PPK rate defaults. NULL PPK rates on
// the users table fall back to zero.
type userRow struct {
	id           int
	name         string
	employeeRate float64
	employerRate float64
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
		return nil, 0, fmt.Errorf("load config: %w", err)
	}
	return birth, ra, nil
}

func (s *Store) loadUsers(ctx context.Context) ([]userRow, error) {
	// The prefill owner picker covers named users who own retirement data:
	// an IKE/IKZE/PPK account or a salary record. This excludes the admin
	// account (NULL name) and any login-only user with no financial rows.
	rows, err := s.pool.Query(ctx, `
		SELECT id, name,
		       COALESCE(ppk_employee_rate, 0), COALESCE(ppk_employer_rate, 0)
		FROM users u
		WHERE name IS NOT NULL
		  AND (
		    EXISTS (
		      SELECT 1 FROM accounts a
		      WHERE a.owner_user_id = u.id
		        AND a.account_wrapper IN ('IKE', 'IKZE', 'PPK')
		    )
		    OR EXISTS (
		      SELECT 1 FROM salary_records sr WHERE sr.owner_user_id = u.id
		    )
		  )
		ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("load users: %w", err)
	}
	defer rows.Close()
	out := []userRow{}
	for rows.Next() {
		var u userRow
		if err := rows.Scan(&u.id, &u.name, &u.employeeRate, &u.employerRate); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return out, nil
}

func (s *Store) fillLatestSalaries(ctx context.Context, users []userRow, dest map[string]*float64) error {
	for _, u := range users {
		row := s.pool.QueryRow(ctx, `
			SELECT gross_amount FROM salary_records
			WHERE owner_user_id = $1 AND is_active = true
			ORDER BY date DESC LIMIT 1`,
			u.id,
		)
		var v float64
		if err := row.Scan(&v); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return fmt.Errorf("latest salary: %w", err)
		}
		val := v
		dest[strconv.Itoa(u.id)] = &val
	}
	return nil
}

// fillWrapperBalances reads the latest snapshot and overwrites the
// ike_/ikze_ + PPK balance keys with the snapshot value for each
// wrapper account.
func (s *Store) fillWrapperBalances(ctx context.Context, data PrefillData) error {
	var snapshotID int
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM snapshots ORDER BY date DESC LIMIT 1`,
	).Scan(&snapshotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("latest snapshot: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT a.account_wrapper, a.owner_user_id, sv.value
		FROM accounts a
		JOIN snapshot_values sv ON sv.account_id = a.id
		WHERE a.is_active = true
		  AND a.account_wrapper IN ('IKE', 'IKZE', 'PPK')
		  AND a.owner_user_id IS NOT NULL
		  AND sv.snapshot_id = $1`,
		snapshotID,
	)
	if err != nil {
		return fmt.Errorf("wrapper balances: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var wrapper string
		var ownerUserID int
		var value float64
		if err := rows.Scan(&wrapper, &ownerUserID, &value); err != nil {
			return fmt.Errorf("scan wrapper balance: %w", err)
		}
		key := strconv.Itoa(ownerUserID)
		if wrapper == "PPK" {
			data.PPKBalances[key] = value
			continue
		}
		switch wrapper {
		case "IKE":
			data.Balances["ike_"+key] = value
		case "IKZE":
			data.Balances["ikze_"+key] = value
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate wrapper balances: %w", err)
	}
	return nil
}

// UserNames returns an owner_user_id -> name map for every named user — the
// retirement handler uses it to render AccountName labels.
func (s *Store) UserNames(ctx context.Context) (map[int]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name FROM users WHERE name IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("user names: %w", err)
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan user name: %w", err)
		}
		out[id] = name
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user names: %w", err)
	}
	return out, nil
}

func ageFromBirth(birth *time.Time) int {
	if birth == nil {
		return 30
	}
	now := time.Now().UTC()
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() ||
		(now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

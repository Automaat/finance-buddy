package simulations

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
func (s *Store) LimitForYear(ctx context.Context, year int, wrapper, owner string) (float64, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT limit_amount FROM retirement_limits
		WHERE year = $1 AND account_wrapper = $2 AND owner = $3`,
		year, wrapper, owner,
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

// PrefillData is everything GET /api/simulations/prefill needs.
type PrefillData struct {
	CurrentAge      int
	RetirementAge   int
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

	personas, err := s.loadPersonas(ctx)
	if err != nil {
		return data, err
	}
	for _, p := range personas {
		lower := strings.ToLower(p.name)
		data.Balances["ike_"+lower] = 0
		data.Balances["ikze_"+lower] = 0
		data.PPKBalances[lower] = 0
		data.PPKRates[lower] = map[string]float64{
			"employee": p.employeeRate,
			"employer": p.employerRate,
		}
		data.MonthlySalaries[lower] = nil
	}
	if err := s.fillLatestSalaries(ctx, personas, data.MonthlySalaries); err != nil {
		return data, err
	}
	if err := s.fillWrapperBalances(ctx, data); err != nil {
		return data, err
	}
	return data, nil
}

type personaRow struct {
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

func (s *Store) loadPersonas(ctx context.Context) ([]personaRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT name, ppk_employee_rate, ppk_employer_rate FROM personas ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("load personas: %w", err)
	}
	defer rows.Close()
	out := []personaRow{}
	for rows.Next() {
		var p personaRow
		if err := rows.Scan(&p.name, &p.employeeRate, &p.employerRate); err != nil {
			return nil, fmt.Errorf("scan persona: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate personas: %w", err)
	}
	return out, nil
}

func (s *Store) fillLatestSalaries(ctx context.Context, personas []personaRow, dest map[string]*float64) error {
	for _, p := range personas {
		row := s.pool.QueryRow(ctx, `
			SELECT gross_amount FROM salary_records
			WHERE owner = $1 AND is_active = true
			ORDER BY date DESC LIMIT 1`,
			p.name,
		)
		var v float64
		if err := row.Scan(&v); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return fmt.Errorf("latest salary: %w", err)
		}
		val := v
		dest[strings.ToLower(p.name)] = &val
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
		SELECT a.account_wrapper, a.owner, sv.value
		FROM accounts a
		JOIN snapshot_values sv ON sv.account_id = a.id
		WHERE a.is_active = true
		  AND a.account_wrapper IN ('IKE', 'IKZE', 'PPK')
		  AND sv.snapshot_id = $1`,
		snapshotID,
	)
	if err != nil {
		return fmt.Errorf("wrapper balances: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var wrapper, owner string
		var value float64
		if err := rows.Scan(&wrapper, &owner, &value); err != nil {
			return fmt.Errorf("scan wrapper balance: %w", err)
		}
		ownerLower := strings.ToLower(owner)
		if wrapper == "PPK" {
			data.PPKBalances[ownerLower] = value
			continue
		}
		data.Balances[strings.ToLower(wrapper)+"_"+ownerLower] = value
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate wrapper balances: %w", err)
	}
	return nil
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

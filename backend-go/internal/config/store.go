// Package config implements the /api/config endpoints.
//
// app_config is a singleton (id=1, enforced by a CHECK constraint). Read
// returns the row or 404; write upserts.
package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Config mirrors backend/app/models/app_config.AppConfig — DB column names
// and types kept identical for parity-suite assertions.
type Config struct {
	ID                      int             `json:"id"`
	BirthDate               time.Time       `json:"birth_date"`
	RetirementAge           int             `json:"retirement_age"`
	RetirementMonthlySalary decimal.Decimal `json:"retirement_monthly_salary"`
	AllocationRealEstate    int             `json:"allocation_real_estate"`
	AllocationStocks        int             `json:"allocation_stocks"`
	AllocationBonds         int             `json:"allocation_bonds"`
	AllocationGold          int             `json:"allocation_gold"`
	AllocationCommodities   int             `json:"allocation_commodities"`
	MonthlyExpenses         decimal.Decimal `json:"monthly_expenses"`
	MonthlyMortgagePayment  decimal.Decimal `json:"monthly_mortgage_payment"`
}

// ErrNotFound is returned when no row exists at id=1.
var ErrNotFound = errors.New("config not found")

// Store is the persistence boundary for app_config.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `
	id, birth_date, retirement_age, retirement_monthly_salary,
	allocation_real_estate, allocation_stocks, allocation_bonds,
	allocation_gold, allocation_commodities,
	monthly_expenses, monthly_mortgage_payment
`

// Get returns the singleton config or ErrNotFound.
func (s *Store) Get(ctx context.Context) (*Config, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+selectColumns+` FROM app_config WHERE id = 1`)
	c, err := scanConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select app_config: %w", err)
	}
	return c, nil
}

// Upsert writes the singleton row (id=1), creating it if missing.
//
// This mirrors backend/app/services/config.py::upsert_config — single
// statement, no separate create/update branches because the CHECK
// constraint guarantees a single row.
func (s *Store) Upsert(ctx context.Context, in *Config) (*Config, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO app_config (
			id, birth_date, retirement_age, retirement_monthly_salary,
			allocation_real_estate, allocation_stocks, allocation_bonds,
			allocation_gold, allocation_commodities,
			monthly_expenses, monthly_mortgage_payment
		) VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			birth_date = EXCLUDED.birth_date,
			retirement_age = EXCLUDED.retirement_age,
			retirement_monthly_salary = EXCLUDED.retirement_monthly_salary,
			allocation_real_estate = EXCLUDED.allocation_real_estate,
			allocation_stocks = EXCLUDED.allocation_stocks,
			allocation_bonds = EXCLUDED.allocation_bonds,
			allocation_gold = EXCLUDED.allocation_gold,
			allocation_commodities = EXCLUDED.allocation_commodities,
			monthly_expenses = EXCLUDED.monthly_expenses,
			monthly_mortgage_payment = EXCLUDED.monthly_mortgage_payment
		RETURNING `+selectColumns,
		in.BirthDate,
		in.RetirementAge,
		in.RetirementMonthlySalary,
		in.AllocationRealEstate,
		in.AllocationStocks,
		in.AllocationBonds,
		in.AllocationGold,
		in.AllocationCommodities,
		in.MonthlyExpenses,
		in.MonthlyMortgagePayment,
	)
	c, err := scanConfig(row)
	if err != nil {
		return nil, fmt.Errorf("upsert app_config: %w", err)
	}
	return c, nil
}

func scanConfig(row pgx.Row) (*Config, error) {
	var c Config
	err := row.Scan(
		&c.ID,
		&c.BirthDate,
		&c.RetirementAge,
		&c.RetirementMonthlySalary,
		&c.AllocationRealEstate,
		&c.AllocationStocks,
		&c.AllocationBonds,
		&c.AllocationGold,
		&c.AllocationCommodities,
		&c.MonthlyExpenses,
		&c.MonthlyMortgagePayment,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

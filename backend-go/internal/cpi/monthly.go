// Monthly CPI data path. The Ministry's "list emisyjny" formula for
// inflation-indexed retail bonds (EDO, COI, ROS, ROD) reads as:
//
//	rate for okres odsetkowy N = margin + CPI YoY published in the month
//	preceding the first month of the period
//
// E.g. an EDO purchased on 2023-01-10 with period 2 starting 2024-01-10
// uses the YoY value published in December 2023 — which is the YoY for
// November 2023 (GUS releases month X's reading around day 15 of month
// X+1). So the reference is always two months before the period's first
// month.
//
// Annual CPI (cpi_index) is too coarse: it would attribute the period-2
// rate to the 2024 annual YoY (3.6%) when the actual Nov-2023 monthly
// figure was ~6.6%. That mismatch was the source of the ~0.3% systematic
// undervaluation observed against PKO portfolio screenshots.
//
// Data source: Eurostat HICP monthly (prc_hicp_manr, Poland, all-items,
// annual rate of change). HICP differs from the Ministry's reference
// (GUS national CPI) by ~0.1-0.3pp; the residual error is documented in
// internal/bonds/calc.go.
package cpi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// MonthYearRate is one (year, month, yoy_rate) tuple.
type MonthYearRate struct {
	Year  int
	Month int // 1-12
	YoY   decimal.Decimal
}

// YearMonth is the composite key used by LoadMonthlyYoYMap. Compares
// cleanly when year/month are within the documented ranges (1900–9999, 1–12).
type YearMonth struct {
	Year  int
	Month int
}

// EnsureMonthlySchema creates cpi_monthly_index if missing. Idempotent so
// it can run on every startup like the other EnsureSchema helpers.
func (s *Store) EnsureMonthlySchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS cpi_monthly_index (
			year integer NOT NULL,
			month integer NOT NULL,
			yoy_rate numeric(8,4) NOT NULL,
			source character varying(64) NOT NULL,
			fetched_at timestamp with time zone NOT NULL DEFAULT (now() at time zone 'utc'),
			PRIMARY KEY (year, month)
		)`)
	if err != nil {
		return fmt.Errorf("create cpi_monthly_index: %w", err)
	}
	return nil
}

// LoadMonthlyYoYMap returns YearMonth → yoy_rate for every cpi_monthly_index
// row. Used by bonds.calc to look up the period-aware reference rate.
func (s *Store) LoadMonthlyYoYMap(ctx context.Context) (map[YearMonth]decimal.Decimal, error) {
	rows, err := s.pool.Query(ctx, `SELECT year, month, yoy_rate FROM cpi_monthly_index`)
	if err != nil {
		return nil, fmt.Errorf("select cpi_monthly_index: %w", err)
	}
	defer rows.Close()
	out := map[YearMonth]decimal.Decimal{}
	for rows.Next() {
		var y, m int
		var rate decimal.Decimal
		if err := rows.Scan(&y, &m, &rate); err != nil {
			return nil, fmt.Errorf("scan cpi_monthly_index: %w", err)
		}
		out[YearMonth{Year: y, Month: m}] = rate
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cpi_monthly_index: %w", err)
	}
	return out, nil
}

// NeedsMonthlyRefresh mirrors NeedsRefresh but for cpi_monthly_index.
func (s *Store) NeedsMonthlyRefresh(ctx context.Context, staleAfter time.Duration) (bool, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT fetched_at FROM cpi_monthly_index ORDER BY fetched_at DESC LIMIT 1`,
	)
	var latest time.Time
	if err := row.Scan(&latest); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, fmt.Errorf("monthly cpi staleness check: %w", err)
	}
	return time.Since(latest) > staleAfter, nil
}

// UpsertMonthly applies the (year, month, yoy_rate) batch. Same change-only
// semantics as Upsert: rows whose rate hasn't moved are skipped. Returns
// the count of rows actually written or refreshed.
func (s *Store) UpsertMonthly(ctx context.Context, source string, rows []MonthYearRate) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin monthly upsert: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	written, err := writeMonthlyRows(ctx, tx, source, rows)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit monthly upsert: %w", err)
	}
	committed = true
	return written, nil
}

func writeMonthlyRows(ctx context.Context, tx pgx.Tx, source string, rows []MonthYearRate) (int, error) {
	existing := map[YearMonth]decimal.Decimal{}
	q, err := tx.Query(ctx, `SELECT year, month, yoy_rate FROM cpi_monthly_index FOR UPDATE`)
	if err != nil {
		return 0, fmt.Errorf("lock cpi_monthly_index: %w", err)
	}
	for q.Next() {
		var y, m int
		var rate decimal.Decimal
		if err := q.Scan(&y, &m, &rate); err != nil {
			q.Close()
			return 0, fmt.Errorf("scan existing cpi_monthly_index: %w", err)
		}
		existing[YearMonth{Year: y, Month: m}] = rate
	}
	q.Close()
	if err := q.Err(); err != nil {
		return 0, fmt.Errorf("iterate cpi_monthly_index: %w", err)
	}
	written := 0
	now := time.Now().UTC()
	for _, r := range rows {
		if current, ok := existing[YearMonth{Year: r.Year, Month: r.Month}]; ok && current.Equal(r.YoY) {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO cpi_monthly_index (year, month, yoy_rate, source, fetched_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (year, month) DO UPDATE SET
				yoy_rate = EXCLUDED.yoy_rate,
				source = EXCLUDED.source,
				fetched_at = EXCLUDED.fetched_at
			WHERE cpi_monthly_index.yoy_rate IS DISTINCT FROM EXCLUDED.yoy_rate`,
			r.Year, r.Month, r.YoY, source, now,
		); err != nil {
			return written, fmt.Errorf("upsert cpi_monthly_index %d-%02d: %w", r.Year, r.Month, err)
		}
		written++
	}
	return written, nil
}

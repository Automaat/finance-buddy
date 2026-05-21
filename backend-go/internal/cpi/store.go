package cpi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// CpiIndex mirrors backend/app/models/cpi_index.CpiIndex.
type CpiIndex struct {
	Year      int
	YoYRate   decimal.Decimal
	Source    string
	FetchedAt time.Time
}

// Store is the persistence boundary.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// LoadYoYMap returns year → yoy_rate for every cpi_index row.
func (s *Store) LoadYoYMap(ctx context.Context) (map[int]decimal.Decimal, error) {
	rows, err := s.pool.Query(ctx, `SELECT year, yoy_rate FROM cpi_index`)
	if err != nil {
		return nil, fmt.Errorf("select cpi_index: %w", err)
	}
	defer rows.Close()
	out := map[int]decimal.Decimal{}
	for rows.Next() {
		var y int
		var rate decimal.Decimal
		if err := rows.Scan(&y, &rate); err != nil {
			return nil, fmt.Errorf("scan cpi_index: %w", err)
		}
		out[y] = rate
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cpi_index: %w", err)
	}
	return out, nil
}

// LatestKnownYear returns the highest year present in cpi_index, or (0, false)
// when the table is empty.
func (s *Store) LatestKnownYear(ctx context.Context) (int, bool, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT year FROM cpi_index ORDER BY year DESC LIMIT 1`,
	)
	var y int
	if err := row.Scan(&y); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("select latest cpi year: %w", err)
	}
	return y, true, nil
}

// Upsert applies the (year, yoy_rate) batch as Python's refresh_cpi_sync does:
// only updates the existing row when the value changed and inserts otherwise.
// Returns the number of rows actually written.
func (s *Store) Upsert(ctx context.Context, source string, rows []YearRate) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	existing, err := s.LoadYoYMap(ctx)
	if err != nil {
		return 0, err
	}
	written := 0
	now := time.Now().UTC()
	for _, r := range rows {
		if current, ok := existing[r.Year]; ok {
			if current.Equal(r.YoY) {
				continue
			}
			if _, err := s.pool.Exec(ctx, `
				UPDATE cpi_index SET yoy_rate = $1, source = $2, fetched_at = $3
				WHERE year = $4`,
				r.YoY, source, now, r.Year,
			); err != nil {
				return written, fmt.Errorf("update cpi_index year %d: %w", r.Year, err)
			}
			written++
			continue
		}
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO cpi_index (year, yoy_rate, source, fetched_at)
			VALUES ($1, $2, $3, $4)`,
			r.Year, r.YoY, source, now,
		); err != nil {
			return written, fmt.Errorf("insert cpi_index year %d: %w", r.Year, err)
		}
		written++
	}
	return written, nil
}

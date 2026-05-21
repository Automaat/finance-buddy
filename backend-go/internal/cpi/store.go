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
// only writes rows whose yoy_rate actually changed. The whole batch is wrapped
// in a transaction so concurrent refreshes can't leave the table partially
// updated. Per-row uses INSERT ... ON CONFLICT (year) DO UPDATE so concurrent
// inserts race-cleanly on the primary key.
//
// Returns the number of rows actually written.
func (s *Store) Upsert(ctx context.Context, source string, rows []YearRate) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin upsert tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	written, err := writeRows(ctx, tx, source, rows)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit upsert tx: %w", err)
	}
	committed = true
	return written, nil
}

func writeRows(ctx context.Context, tx pgx.Tx, source string, rows []YearRate) (int, error) {
	existing := map[int]decimal.Decimal{}
	q, err := tx.Query(ctx, `SELECT year, yoy_rate FROM cpi_index FOR UPDATE`)
	if err != nil {
		return 0, fmt.Errorf("lock cpi_index: %w", err)
	}
	for q.Next() {
		var y int
		var rate decimal.Decimal
		if err := q.Scan(&y, &rate); err != nil {
			q.Close()
			return 0, fmt.Errorf("scan existing cpi_index: %w", err)
		}
		existing[y] = rate
	}
	q.Close()
	if err := q.Err(); err != nil {
		return 0, fmt.Errorf("iterate cpi_index: %w", err)
	}
	written := 0
	now := time.Now().UTC()
	for _, r := range rows {
		if current, ok := existing[r.Year]; ok && current.Equal(r.YoY) {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO cpi_index (year, yoy_rate, source, fetched_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (year) DO UPDATE SET
				yoy_rate = EXCLUDED.yoy_rate,
				source = EXCLUDED.source,
				fetched_at = EXCLUDED.fetched_at
			WHERE cpi_index.yoy_rate IS DISTINCT FROM EXCLUDED.yoy_rate`,
			r.Year, r.YoY, source, now,
		); err != nil {
			return written, fmt.Errorf("upsert cpi_index year %d: %w", r.Year, err)
		}
		written++
	}
	return written, nil
}

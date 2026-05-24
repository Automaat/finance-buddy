package holdings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Store is the persistence boundary for securities, lots and price quotes.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Security mirrors the securities table.
type Security struct {
	ID        int
	Symbol    string
	ISIN      *string
	Name      string
	AssetType string
	Currency  string
	CreatedAt time.Time
}

// PriceQuote mirrors the price_quotes table.
type PriceQuote struct {
	ID         int
	SecurityID int
	Date       time.Time
	Price      decimal.Decimal
	Source     string
	CreatedAt  time.Time
}

// HoldingRow is one aggregated security position across all accounts.
type HoldingRow struct {
	Security        Security
	Running         Running
	LatestQuote     decimal.Decimal
	LatestQuoteDate *time.Time
	MarketValue     decimal.Decimal
	UnrealizedGain  decimal.Decimal
}

// Sentinel errors.
var (
	ErrSecurityNotFound = errors.New("security not found")
	ErrLotNotFound      = errors.New("lot not found")
	ErrQuoteNotFound    = errors.New("price quote not found")
)

const secCols = `id, symbol, isin, name, asset_type, currency, created_at`

func scanSecurity(row pgx.Row) (Security, error) {
	var s Security
	if err := row.Scan(&s.ID, &s.Symbol, &s.ISIN, &s.Name, &s.AssetType, &s.Currency, &s.CreatedAt); err != nil {
		return Security{}, err
	}
	return s, nil
}

// ListSecurities returns all securities, ordered by symbol.
func (s *Store) ListSecurities(ctx context.Context) ([]Security, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+secCols+` FROM securities ORDER BY symbol`)
	if err != nil {
		return nil, fmt.Errorf("list securities: %w", err)
	}
	defer rows.Close()
	out := []Security{}
	for rows.Next() {
		sec, err := scanSecurity(rows)
		if err != nil {
			return nil, fmt.Errorf("scan security: %w", err)
		}
		out = append(out, sec)
	}
	return out, rows.Err()
}

// CreateSecurity inserts a new security.
func (s *Store) CreateSecurity(ctx context.Context, sec Security) (Security, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO securities (symbol, isin, name, asset_type, currency)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+secCols,
		sec.Symbol, sec.ISIN, sec.Name, sec.AssetType, sec.Currency,
	)
	return scanSecurity(row)
}

// GetSecurity returns one security by ID.
func (s *Store) GetSecurity(ctx context.Context, id int) (Security, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+secCols+` FROM securities WHERE id = $1`, id)
	sec, err := scanSecurity(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Security{}, ErrSecurityNotFound
	}
	return sec, err
}

// DeleteSecurity removes a security. Referenced lots block via ON DELETE RESTRICT.
func (s *Store) DeleteSecurity(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM securities WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete security: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSecurityNotFound
	}
	return nil
}

const lotCols = `id, account_id, security_id, side, quantity, price, fee, date, created_at`

func scanLot(row pgx.Row) (Lot, error) {
	var l Lot
	var sideStr string
	if err := row.Scan(&l.ID, &l.AccountID, &l.SecurityID, &sideStr, &l.Quantity, &l.Price, &l.Fee, &l.Date, &l.CreatedAt); err != nil {
		return Lot{}, err
	}
	l.Side = Side(sideStr)
	return l, nil
}

// ListLots returns lots for a specific account or security, both optional.
func (s *Store) ListLots(ctx context.Context, accountID, securityID *int) ([]Lot, error) {
	args := []any{}
	clauses := []string{}
	if accountID != nil {
		args = append(args, *accountID)
		clauses = append(clauses, fmt.Sprintf("account_id = $%d", len(args)))
	}
	if securityID != nil {
		args = append(args, *securityID)
		clauses = append(clauses, fmt.Sprintf("security_id = $%d", len(args)))
	}
	where := ""
	for i, c := range clauses {
		if i == 0 {
			where = " WHERE " + c
		} else {
			where += " AND " + c
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT `+lotCols+` FROM lots`+where+` ORDER BY date ASC, id ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	defer rows.Close()
	out := []Lot{}
	for rows.Next() {
		l, err := scanLot(rows)
		if err != nil {
			return nil, fmt.Errorf("scan lot: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CreateLot inserts a new lot.
func (s *Store) CreateLot(ctx context.Context, l Lot) (Lot, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO lots (account_id, security_id, side, quantity, price, fee, date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+lotCols,
		l.AccountID, l.SecurityID, string(l.Side), l.Quantity, l.Price, l.Fee, l.Date,
	)
	return scanLot(row)
}

// DeleteLot removes a lot by id.
func (s *Store) DeleteLot(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM lots WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete lot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrLotNotFound
	}
	return nil
}

const quoteCols = `id, security_id, date, price, source, created_at`

func scanQuote(row pgx.Row) (PriceQuote, error) {
	var q PriceQuote
	if err := row.Scan(&q.ID, &q.SecurityID, &q.Date, &q.Price, &q.Source, &q.CreatedAt); err != nil {
		return PriceQuote{}, err
	}
	return q, nil
}

// UpsertQuote inserts or updates a quote on (security_id, date).
func (s *Store) UpsertQuote(ctx context.Context, q PriceQuote) (PriceQuote, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO price_quotes (security_id, date, price, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (security_id, date) DO UPDATE
		SET price = EXCLUDED.price, source = EXCLUDED.source
		RETURNING `+quoteCols,
		q.SecurityID, q.Date, q.Price, q.Source,
	)
	return scanQuote(row)
}

// ListQuotes returns all quotes for a security, ordered by date desc.
func (s *Store) ListQuotes(ctx context.Context, securityID int) ([]PriceQuote, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+quoteCols+` FROM price_quotes WHERE security_id = $1 ORDER BY date DESC`, securityID)
	if err != nil {
		return nil, fmt.Errorf("list quotes: %w", err)
	}
	defer rows.Close()
	out := []PriceQuote{}
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			return nil, fmt.Errorf("scan quote: %w", err)
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// DeleteQuote removes a quote by id.
func (s *Store) DeleteQuote(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM price_quotes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete quote: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrQuoteNotFound
	}
	return nil
}

// latestQuotes returns latest price per security as of now.
func (s *Store) latestQuotes(ctx context.Context) (map[int]PriceQuote, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (security_id) `+quoteCols+`
		FROM price_quotes
		ORDER BY security_id, date DESC`)
	if err != nil {
		return nil, fmt.Errorf("latest quotes: %w", err)
	}
	defer rows.Close()
	out := map[int]PriceQuote{}
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			return nil, fmt.Errorf("scan latest quote: %w", err)
		}
		out[q.SecurityID] = q
	}
	return out, rows.Err()
}

// Holdings replays every lot, groups by security, attaches the latest quote,
// and returns one HoldingRow per security with non-zero quantity.
func (s *Store) Holdings(ctx context.Context) ([]HoldingRow, error) {
	securities, err := s.ListSecurities(ctx)
	if err != nil {
		return nil, err
	}
	allLots, err := s.ListLots(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	quotes, err := s.latestQuotes(ctx)
	if err != nil {
		return nil, err
	}
	bySec := map[int][]Lot{}
	for i := range allLots {
		l := &allLots[i]
		bySec[l.SecurityID] = append(bySec[l.SecurityID], *l)
	}
	out := []HoldingRow{}
	for _, sec := range securities {
		lots := bySec[sec.ID]
		run, runErr := ComputeRunning(lots)
		if runErr != nil {
			// Skip securities with corrupted lot histories rather than failing
			// the whole endpoint. Log so the corruption is visible in ops.
			slog.Default().Warn("holdings: skipping security with bad lot history",
				"security_id", sec.ID, "symbol", sec.Symbol, "err", runErr)
			continue
		}
		if run.Quantity.IsZero() && run.RealizedGain.IsZero() {
			continue
		}
		row := HoldingRow{Security: sec, Running: run}
		if q, ok := quotes[sec.ID]; ok {
			row.LatestQuote = q.Price
			d := q.Date
			row.LatestQuoteDate = &d
		}
		row.MarketValue = MarketValue(run, row.LatestQuote)
		row.UnrealizedGain = UnrealizedGain(run, row.LatestQuote)
		out = append(out, row)
	}
	return out, nil
}

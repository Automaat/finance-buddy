package holdings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// ErrDividendNotFound is returned by DeleteDividend when no row matches.
var ErrDividendNotFound = errors.New("dividend not found")

// Dividend is one cash-dividend payout recorded against a security held in an
// account: the gross amount and the withholding tax taken at source. Net (the
// cash that actually landed) is gross − withholding.
type Dividend struct {
	ID             int
	AccountID      int
	SecurityID     int
	PayDate        time.Time
	GrossAmount    decimal.Decimal
	WithholdingTax decimal.Decimal
	Currency       string
	CreatedAt      time.Time
}

// Net is the cash received after withholding.
func (d Dividend) Net() decimal.Decimal { return d.GrossAmount.Sub(d.WithholdingTax) }

// EnsureDividendsSchema creates the holding_dividends table + index if missing.
// Idempotent so it can run on every startup — schema.sql only seeds empty
// databases, so existing installs need the additive DDL here too (mirrors the
// bonds / allocation EnsureSchema pattern).
func (s *Store) EnsureDividendsSchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS holding_dividends (
			id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			account_id integer NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			security_id integer NOT NULL REFERENCES securities(id) ON DELETE RESTRICT,
			pay_date date NOT NULL,
			gross_amount numeric(15, 2) NOT NULL,
			withholding_tax numeric(15, 2) NOT NULL DEFAULT 0,
			currency varchar(3) NOT NULL DEFAULT 'PLN',
			created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
		)`); err != nil {
		return fmt.Errorf("ensure holding_dividends table: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS ix_holding_dividends_acct_sec_date
		ON holding_dividends (account_id, security_id, pay_date DESC)`); err != nil {
		return fmt.Errorf("ensure holding_dividends index: %w", err)
	}
	return nil
}

const divCols = `id, account_id, security_id, pay_date, gross_amount, withholding_tax, currency, created_at`

func scanDividend(row pgx.Row) (Dividend, error) {
	var d Dividend
	if err := row.Scan(&d.ID, &d.AccountID, &d.SecurityID, &d.PayDate,
		&d.GrossAmount, &d.WithholdingTax, &d.Currency, &d.CreatedAt); err != nil {
		return Dividend{}, err
	}
	return d, nil
}

// ListDividends returns dividends, newest pay-date first, optionally filtered
// by account and/or security.
func (s *Store) ListDividends(ctx context.Context, accountID, securityID *int) ([]Dividend, error) {
	args := []any{}
	where := ""
	add := func(clause string, val any) {
		args = append(args, val)
		if where == "" {
			where = fmt.Sprintf("WHERE %s$%d", clause, len(args))
		} else {
			where += fmt.Sprintf(" AND %s$%d", clause, len(args))
		}
	}
	if accountID != nil {
		add("account_id = ", *accountID)
	}
	if securityID != nil {
		add("security_id = ", *securityID)
	}
	rows, err := s.pool.Query(ctx, `SELECT `+divCols+` FROM holding_dividends `+where+`
		ORDER BY pay_date DESC, id DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list dividends: %w", err)
	}
	defer rows.Close()
	out := []Dividend{}
	for rows.Next() {
		d, err := scanDividend(rows)
		if err != nil {
			return nil, fmt.Errorf("scan dividend: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CreateDividend inserts one dividend and returns the persisted row.
func (s *Store) CreateDividend(ctx context.Context, d Dividend) (Dividend, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO holding_dividends (account_id, security_id, pay_date, gross_amount, withholding_tax, currency)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+divCols,
		d.AccountID, d.SecurityID, d.PayDate, d.GrossAmount, d.WithholdingTax, d.Currency)
	created, err := scanDividend(row)
	if err != nil {
		return Dividend{}, fmt.Errorf("create dividend: %w", err)
	}
	return created, nil
}

// DeleteDividend removes a dividend by id, returning ErrDividendNotFound when
// no row matched.
func (s *Store) DeleteDividend(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM holding_dividends WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete dividend: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDividendNotFound
	}
	return nil
}

// Package debtpayments implements /api/accounts/{id}/payments + /api/payments.
package debtpayments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/dbutil"
)

// DebtPayment mirrors backend/app/models/debt_payment.DebtPayment.
type DebtPayment struct {
	ID          int
	AccountID   int
	Amount      decimal.Decimal
	Date        time.Time
	OwnerUserID *int
	IsActive    bool
	CreatedAt   time.Time
}

// AccountInfo carries the subset of the parent account we need.
type AccountInfo struct {
	ID       int
	Name     string
	Type     string
	IsActive bool
}

// Sentinel errors mapped to HTTP statuses.
var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrAccountNotLiability = errors.New("account is not a liability")
	ErrNotFound            = errors.New("payment not found")
	ErrDuplicate           = errors.New("payment for date already exists")
	ErrCrossAccount        = errors.New("payment does not belong to account")
)

// Store is the persistence boundary.
type Store struct{ pool *pgxpool.Pool }

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

const paymentCols = `id, account_id, amount, date, owner_user_id, is_active, created_at`

// LoadAccount mirrors Python's validation: active account that is type=liability.
// Returns the account along with ErrAccountNotLiability so the handler can use
// its name in the 400 detail.
func (s *Store) LoadAccount(ctx context.Context, id int) (*AccountInfo, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, name, type, is_active FROM accounts WHERE id = $1`, id,
	)
	var a AccountInfo
	if err := row.Scan(&a.ID, &a.Name, &a.Type, &a.IsActive); err != nil {
		return nil, dbutil.MapErr(err, ErrAccountNotFound, "load account")
	}
	if !a.IsActive {
		return nil, ErrAccountNotFound
	}
	if a.Type != "liability" {
		return &a, ErrAccountNotLiability
	}
	return &a, nil
}

// ListForAccount returns active payments for one account.
func (s *Store) ListForAccount(ctx context.Context, accountID int) ([]DebtPayment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+paymentCols+` FROM debt_payments
		 WHERE account_id = $1 AND is_active = true
		 ORDER BY date DESC, id DESC`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("query payments: %w", err)
	}
	return dbutil.CollectRows(rows, scanPaymentValue, "scan payment", "iterate payments")
}

// ListFilter narrows the all-payments view.
type ListFilter struct {
	AccountID   *int
	OwnerUserID *int
	DateFrom    *time.Time
	DateTo      *time.Time
}

// PaymentWithAccount is one payment with its account name joined in.
type PaymentWithAccount struct {
	Payment     DebtPayment
	AccountName string
}

// ListAll joins debt_payments to active accounts and applies optional filters.
func (s *Store) ListAll(ctx context.Context, f ListFilter) ([]PaymentWithAccount, error) {
	conds := []string{"p.is_active = true", "a.is_active = true"}
	args := []any{}
	if f.AccountID != nil {
		args = append(args, *f.AccountID)
		conds = append(conds, fmt.Sprintf("p.account_id = $%d", len(args)))
	}
	if f.OwnerUserID != nil {
		args = append(args, *f.OwnerUserID)
		conds = append(conds, fmt.Sprintf("p.owner_user_id = $%d", len(args)))
	}
	if f.DateFrom != nil {
		args = append(args, *f.DateFrom)
		conds = append(conds, fmt.Sprintf("p.date >= $%d", len(args)))
	}
	if f.DateTo != nil {
		args = append(args, *f.DateTo)
		conds = append(conds, fmt.Sprintf("p.date <= $%d", len(args)))
	}
	q := `SELECT p.id, p.account_id, p.amount, p.date, p.owner_user_id, p.is_active, p.created_at, a.name
	      FROM debt_payments p
	      JOIN accounts a ON a.id = p.account_id
	      WHERE ` + strings.Join(conds, " AND ") + `
	      ORDER BY p.date DESC, p.id DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	return dbutil.CollectRows(rows, scanPaymentWithAccount, "scan payment join", "iterate payments")
}

// Create inserts a payment. ErrDuplicate on same (account_id, date) for an
// active row.
func (s *Store) Create(ctx context.Context, p *DebtPayment) (*DebtPayment, error) {
	var existing int
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM debt_payments
		WHERE account_id = $1 AND date = $2 AND is_active = true
		LIMIT 1`,
		p.AccountID, p.Date,
	).Scan(&existing)
	if err == nil {
		return nil, ErrDuplicate
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check duplicate payment: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO debt_payments (account_id, amount, date, owner_user_id, is_active, created_at)
		VALUES ($1, $2, $3, $4, true, $5)
		RETURNING `+paymentCols,
		p.AccountID, p.Amount, p.Date, p.OwnerUserID, time.Now().UTC(),
	)
	out, err := scanPayment(row)
	if err != nil {
		return nil, fmt.Errorf("insert payment: %w", err)
	}
	return out, nil
}

// SoftDelete soft-deletes the payment, enforcing account ownership.
func (s *Store) SoftDelete(ctx context.Context, accountID, paymentID int) error {
	var owned bool
	err := s.pool.QueryRow(ctx, `
		SELECT account_id = $1 FROM debt_payments WHERE id = $2`,
		accountID, paymentID,
	).Scan(&owned)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("check payment ownership: %w", err)
	}
	if !owned {
		return ErrCrossAccount
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE debt_payments SET is_active = false WHERE id = $1`, paymentID,
	); err != nil {
		return fmt.Errorf("soft-delete payment: %w", err)
	}
	return nil
}

// CountsByAccount returns the per-account count of active payments.
func (s *Store) CountsByAccount(ctx context.Context) (map[int]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT account_id, COUNT(*)
		FROM debt_payments WHERE is_active = true
		GROUP BY account_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("payment counts: %w", err)
	}
	defer rows.Close()
	out := map[int]int{}
	for rows.Next() {
		var id, n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		out[id] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate counts: %w", err)
	}
	return out, nil
}

func scanPayment(row pgx.Row) (*DebtPayment, error) {
	p, err := scanPaymentValue(row)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPaymentValue(row pgx.Row) (DebtPayment, error) {
	var p DebtPayment
	if err := row.Scan(&p.ID, &p.AccountID, &p.Amount, &p.Date, &p.OwnerUserID, &p.IsActive, &p.CreatedAt); err != nil {
		return DebtPayment{}, err
	}
	return p, nil
}

func scanPaymentWithAccount(row pgx.Row) (PaymentWithAccount, error) {
	var p DebtPayment
	var name string
	if err := row.Scan(&p.ID, &p.AccountID, &p.Amount, &p.Date, &p.OwnerUserID, &p.IsActive, &p.CreatedAt, &name); err != nil {
		return PaymentWithAccount{}, err
	}
	return PaymentWithAccount{Payment: p, AccountName: name}, nil
}

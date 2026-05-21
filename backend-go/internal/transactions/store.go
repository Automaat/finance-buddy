// Package transactions implements transaction endpoints for investment +
// retirement accounts. Soft-delete is the only delete; create enforces
// one-per-day per account.
package transactions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Transaction mirrors backend/app/models/transaction.Transaction.
type Transaction struct {
	ID              int
	AccountID       int
	Amount          decimal.Decimal
	Date            time.Time
	Owner           string
	TransactionType *string
	IsActive        bool
	CreatedAt       time.Time
}

// AccountInfo is the subset of Account columns we need for response building
// + parent-account validation.
type AccountInfo struct {
	ID             int
	Name           string
	Type           string
	Category       string
	AccountWrapper *string
	IsActive       bool
}

// Investment categories that may have transactions; retirement wrappers
// (IKE/IKZE/PPK) are also allowed regardless of category.
var investmentCategories = map[string]struct{}{
	"stock": {}, "bond": {}, "fund": {}, "etf": {},
}

// Sentinel errors mapped to HTTP statuses by the handler.
var (
	ErrAccountNotFound      = errors.New("account not found")
	ErrAccountNotInvestment = errors.New("account is not investment/retirement")
	ErrNotFound             = errors.New("transaction not found")
	ErrDuplicate            = errors.New("transaction for date already exists")
	ErrCrossAccount         = errors.New("transaction does not belong to account")
)

// Store is the persistence boundary.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const accountCols = `id, name, type, category, account_wrapper, is_active`

const txCols = `
	id, account_id, amount, date, owner, transaction_type, is_active, created_at
`

// LoadAccount returns the parent-account info; on ErrAccountNotInvestment the
// account is still returned so the handler can render the account name in
// the 400 detail. ErrAccountNotFound for missing or inactive rows.
func (s *Store) LoadAccount(ctx context.Context, id int) (*AccountInfo, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+accountCols+` FROM accounts WHERE id = $1`, id,
	)
	a, err := scanAccount(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("load account: %w", err)
	}
	if !a.IsActive {
		return nil, ErrAccountNotFound
	}
	if _, ok := investmentCategories[a.Category]; !ok && (a.AccountWrapper == nil || *a.AccountWrapper == "") {
		return a, ErrAccountNotInvestment
	}
	return a, nil
}

// ListForAccount returns active transactions for one account, ordered by
// date desc.
func (s *Store) ListForAccount(ctx context.Context, accountID int) ([]Transaction, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+txCols+` FROM transactions
		 WHERE account_id = $1 AND is_active = true
		 ORDER BY date DESC, id DESC`,
		accountID,
	)
	return scanTransactions(rows, err)
}

// ListFilter narrows the all-transactions view.
type ListFilter struct {
	AccountID *int
	Owner     *string
	DateFrom  *time.Time
	DateTo    *time.Time
}

// TxnWithAccount carries one transaction with its account name joined in.
type TxnWithAccount struct {
	Transaction Transaction
	AccountName string
}

// ListAll joins transactions to active accounts and applies optional filters.
func (s *Store) ListAll(ctx context.Context, f ListFilter) ([]TxnWithAccount, error) {
	conds := []string{"t.is_active = true", "a.is_active = true"}
	args := []any{}
	if f.AccountID != nil {
		args = append(args, *f.AccountID)
		conds = append(conds, fmt.Sprintf("t.account_id = $%d", len(args)))
	}
	if f.Owner != nil {
		args = append(args, *f.Owner)
		conds = append(conds, fmt.Sprintf("t.owner = $%d", len(args)))
	}
	if f.DateFrom != nil {
		args = append(args, *f.DateFrom)
		conds = append(conds, fmt.Sprintf("t.date >= $%d", len(args)))
	}
	if f.DateTo != nil {
		args = append(args, *f.DateTo)
		conds = append(conds, fmt.Sprintf("t.date <= $%d", len(args)))
	}
	q := `SELECT t.id, t.account_id, t.amount, t.date, t.owner, t.transaction_type,
	             t.is_active, t.created_at, a.name
	      FROM transactions t
	      JOIN accounts a ON a.id = t.account_id
	      WHERE ` + strings.Join(conds, " AND ") + `
	      ORDER BY t.date DESC, t.id DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()
	out := []TxnWithAccount{}
	for rows.Next() {
		var t Transaction
		var name string
		if err := rows.Scan(
			&t.ID, &t.AccountID, &t.Amount, &t.Date, &t.Owner,
			&t.TransactionType, &t.IsActive, &t.CreatedAt, &name,
		); err != nil {
			return nil, fmt.Errorf("scan transaction join: %w", err)
		}
		out = append(out, TxnWithAccount{Transaction: t, AccountName: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return out, nil
}

// Create inserts a transaction. ErrDuplicate on same (account_id, date) for
// an active row.
func (s *Store) Create(ctx context.Context, t *Transaction) (*Transaction, error) {
	var existing int
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM transactions
		WHERE account_id = $1 AND date = $2 AND is_active = true
		LIMIT 1`,
		t.AccountID, t.Date,
	).Scan(&existing)
	if err == nil {
		return nil, ErrDuplicate
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check duplicate transaction: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO transactions (account_id, amount, date, owner, transaction_type, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, true, $6)
		RETURNING `+txCols,
		t.AccountID, t.Amount, t.Date, t.Owner, t.TransactionType, time.Now().UTC(),
	)
	return scanTransaction(row)
}

// SoftDelete soft-deletes a transaction. Enforces account ownership.
func (s *Store) SoftDelete(ctx context.Context, accountID, transactionID int) error {
	var owned bool
	err := s.pool.QueryRow(ctx, `
		SELECT account_id = $1
		FROM transactions WHERE id = $2`,
		accountID, transactionID,
	).Scan(&owned)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("check transaction ownership: %w", err)
	}
	if !owned {
		return ErrCrossAccount
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE transactions SET is_active = false WHERE id = $1`,
		transactionID,
	); err != nil {
		return fmt.Errorf("soft-delete transaction: %w", err)
	}
	return nil
}

// CountsByAccount returns the per-account count of active transactions.
func (s *Store) CountsByAccount(ctx context.Context) (map[int]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT account_id, COUNT(*)
		FROM transactions WHERE is_active = true
		GROUP BY account_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("transaction counts: %w", err)
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

func scanTransactions(rows pgx.Rows, err error) ([]Transaction, error) {
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()
	out := []Transaction{}
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return out, nil
}

func scanTransaction(row pgx.Row) (*Transaction, error) {
	var t Transaction
	if err := row.Scan(
		&t.ID, &t.AccountID, &t.Amount, &t.Date, &t.Owner,
		&t.TransactionType, &t.IsActive, &t.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &t, nil
}

func scanAccount(row pgx.Row) (*AccountInfo, error) {
	var a AccountInfo
	if err := row.Scan(&a.ID, &a.Name, &a.Type, &a.Category, &a.AccountWrapper, &a.IsActive); err != nil {
		return nil, err
	}
	return &a, nil
}

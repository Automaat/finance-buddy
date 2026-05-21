// Package debts implements /api/debts and /api/accounts/{id}/debts.
package debts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Debt mirrors backend/app/models/debt.Debt.
type Debt struct {
	ID            int
	AccountID     int
	Name          string
	DebtType      string
	StartDate     time.Time
	InitialAmount decimal.Decimal
	InterestRate  decimal.Decimal
	Currency      string
	Notes         *string
	IsActive      bool
	CreatedAt     time.Time
}

// AccountInfo is the subset of Account columns we need.
type AccountInfo struct {
	ID       int
	Name     string
	Owner    string
	Type     string
	IsActive bool
}

// LatestBalance is one row's most-recent snapshot value.
type LatestBalance struct {
	AccountID int
	Value     decimal.Decimal
	Date      time.Time
}

// Sentinel errors.
var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrAccountNotLiability = errors.New("account is not a liability")
	ErrNotFound            = errors.New("debt not found")
	ErrDuplicateForAccount = errors.New("debt already exists for account")
)

// Store is the persistence boundary.
type Store struct{ pool *pgxpool.Pool }

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

const debtCols = `
	id, account_id, name, debt_type, start_date, initial_amount,
	interest_rate, currency, notes, is_active, created_at
`

// debtColsQualified is debtCols with `d.` prefix on every column — used in
// queries that JOIN accounts (which also has `id`).
const debtColsQualified = `
	d.id, d.account_id, d.name, d.debt_type, d.start_date, d.initial_amount,
	d.interest_rate, d.currency, d.notes, d.is_active, d.created_at
`

// LoadAccount mirrors Python's create-debt validation.
func (s *Store) LoadAccount(ctx context.Context, id int) (*AccountInfo, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, name, owner, type, is_active FROM accounts WHERE id = $1`, id,
	)
	var a AccountInfo
	if err := row.Scan(&a.ID, &a.Name, &a.Owner, &a.Type, &a.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("load account: %w", err)
	}
	if !a.IsActive {
		return nil, ErrAccountNotFound
	}
	if a.Type != "liability" {
		return &a, ErrAccountNotLiability
	}
	return &a, nil
}

// DebtWithAccount carries a debt with its account info joined in.
type DebtWithAccount struct {
	Debt    Debt
	Account AccountInfo
}

// ListFilter narrows the all-debts view.
type ListFilter struct {
	AccountID *int
	DebtType  *string
}

// ListAll returns active debts + active-account names/owners.
func (s *Store) ListAll(ctx context.Context, f ListFilter) ([]DebtWithAccount, error) {
	conds := []string{"d.is_active = true", "a.is_active = true"}
	args := []any{}
	if f.AccountID != nil {
		args = append(args, *f.AccountID)
		conds = append(conds, fmt.Sprintf("d.account_id = $%d", len(args)))
	}
	if f.DebtType != nil {
		args = append(args, *f.DebtType)
		conds = append(conds, fmt.Sprintf("d.debt_type = $%d", len(args)))
	}
	q := `SELECT ` + debtColsQualified + `, a.name, a.owner
	      FROM debts d
	      JOIN accounts a ON a.id = d.account_id
	      WHERE ` + strings.Join(conds, " AND ") + `
	      ORDER BY d.start_date DESC, d.id DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list debts: %w", err)
	}
	defer rows.Close()
	out := []DebtWithAccount{}
	for rows.Next() {
		var d Debt
		var a AccountInfo
		if err := rows.Scan(
			&d.ID, &d.AccountID, &d.Name, &d.DebtType, &d.StartDate,
			&d.InitialAmount, &d.InterestRate, &d.Currency, &d.Notes,
			&d.IsActive, &d.CreatedAt, &a.Name, &a.Owner,
		); err != nil {
			return nil, fmt.Errorf("scan debt: %w", err)
		}
		a.ID = d.AccountID
		out = append(out, DebtWithAccount{Debt: d, Account: a})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate debts: %w", err)
	}
	return out, nil
}

// LatestBalanceForAccount returns the most recent snapshot value for an account.
// Returns (false, ...) when no snapshot value exists.
func (s *Store) LatestBalanceForAccount(ctx context.Context, accountID int) (LatestBalance, bool, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT s.date, sv.value
		FROM snapshot_values sv
		JOIN snapshots s ON s.id = sv.snapshot_id
		WHERE sv.account_id = $1
		ORDER BY s.date DESC LIMIT 1`,
		accountID,
	)
	var lb LatestBalance
	lb.AccountID = accountID
	if err := row.Scan(&lb.Date, &lb.Value); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LatestBalance{}, false, nil
		}
		return LatestBalance{}, false, fmt.Errorf("latest balance: %w", err)
	}
	return lb, true, nil
}

// LatestBalancesByAccount batches LatestBalanceForAccount across many accounts.
func (s *Store) LatestBalancesByAccount(ctx context.Context, ids []int) (map[int]LatestBalance, error) {
	if len(ids) == 0 {
		return map[int]LatestBalance{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (sv.account_id) sv.account_id, s.date, sv.value
		FROM snapshot_values sv
		JOIN snapshots s ON s.id = sv.snapshot_id
		WHERE sv.account_id = ANY($1)
		ORDER BY sv.account_id, s.date DESC`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("batch latest balances: %w", err)
	}
	defer rows.Close()
	out := map[int]LatestBalance{}
	for rows.Next() {
		var lb LatestBalance
		if err := rows.Scan(&lb.AccountID, &lb.Date, &lb.Value); err != nil {
			return nil, fmt.Errorf("scan latest balance: %w", err)
		}
		out[lb.AccountID] = lb
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate balances: %w", err)
	}
	return out, nil
}

// TotalPaidByAccount returns the sum of active debt_payments per account.
func (s *Store) TotalPaidByAccount(ctx context.Context, ids []int) (map[int]decimal.Decimal, error) {
	if len(ids) == 0 {
		return map[int]decimal.Decimal{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT account_id, COALESCE(SUM(amount), 0)
		FROM debt_payments WHERE account_id = ANY($1) AND is_active = true
		GROUP BY account_id`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("total paid: %w", err)
	}
	defer rows.Close()
	out := map[int]decimal.Decimal{}
	for rows.Next() {
		var id int
		var total decimal.Decimal
		if err := rows.Scan(&id, &total); err != nil {
			return nil, fmt.Errorf("scan total paid: %w", err)
		}
		out[id] = total
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate total paid: %w", err)
	}
	return out, nil
}

// Get returns one active debt + its account info.
func (s *Store) Get(ctx context.Context, debtID int) (*DebtWithAccount, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+debtColsQualified+`, a.name, a.owner
		FROM debts d
		JOIN accounts a ON a.id = d.account_id
		WHERE d.id = $1 AND d.is_active = true`,
		debtID,
	)
	var d Debt
	var a AccountInfo
	if err := row.Scan(
		&d.ID, &d.AccountID, &d.Name, &d.DebtType, &d.StartDate,
		&d.InitialAmount, &d.InterestRate, &d.Currency, &d.Notes,
		&d.IsActive, &d.CreatedAt, &a.Name, &a.Owner,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get debt: %w", err)
	}
	a.ID = d.AccountID
	return &DebtWithAccount{Debt: d, Account: a}, nil
}

// Create inserts a debt. Returns ErrDuplicateForAccount if the
// uq_debt_account constraint trips (one debt per account in Python).
func (s *Store) Create(ctx context.Context, d *Debt) (*Debt, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO debts (
			account_id, name, debt_type, start_date, initial_amount,
			interest_rate, currency, notes, is_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9)
		RETURNING `+debtCols,
		d.AccountID, d.Name, d.DebtType, d.StartDate, d.InitialAmount,
		d.InterestRate, d.Currency, d.Notes, time.Now().UTC(),
	)
	out, err := scanDebt(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateForAccount
		}
		return nil, fmt.Errorf("insert debt: %w", err)
	}
	return out, nil
}

// UpdatePatch is the sparse update set.
type UpdatePatch struct {
	Name          *string
	DebtType      *string
	StartDate     *time.Time
	InitialAmount *decimal.Decimal
	InterestRate  *decimal.Decimal
	Currency      *string
	NotesSet      bool
	Notes         *string
}

// Update applies the patch.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*DebtWithAccount, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin update tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	row := tx.QueryRow(ctx,
		`SELECT `+debtCols+` FROM debts WHERE id = $1 AND is_active = true FOR UPDATE`, id,
	)
	current, err := scanDebt(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock debt: %w", err)
	}
	applyPatch(current, p)
	if _, err := tx.Exec(ctx, `
		UPDATE debts SET
			name = $1, debt_type = $2, start_date = $3, initial_amount = $4,
			interest_rate = $5, currency = $6, notes = $7
		WHERE id = $8`,
		current.Name, current.DebtType, current.StartDate, current.InitialAmount,
		current.InterestRate, current.Currency, current.Notes, id,
	); err != nil {
		return nil, fmt.Errorf("update debt: %w", err)
	}
	row = tx.QueryRow(ctx,
		`SELECT name, owner FROM accounts WHERE id = $1`, current.AccountID,
	)
	var a AccountInfo
	a.ID = current.AccountID
	if err := row.Scan(&a.Name, &a.Owner); err != nil {
		return nil, fmt.Errorf("load account for debt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update: %w", err)
	}
	committed = true
	return &DebtWithAccount{Debt: *current, Account: a}, nil
}

// Delete soft-deletes a debt.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE debts SET is_active = false WHERE id = $1 AND is_active = true`,
		id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete debt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func applyPatch(d *Debt, p UpdatePatch) {
	if p.Name != nil {
		d.Name = *p.Name
	}
	if p.DebtType != nil {
		d.DebtType = *p.DebtType
	}
	if p.StartDate != nil {
		d.StartDate = *p.StartDate
	}
	if p.InitialAmount != nil {
		d.InitialAmount = *p.InitialAmount
	}
	if p.InterestRate != nil {
		d.InterestRate = *p.InterestRate
	}
	if p.Currency != nil {
		d.Currency = *p.Currency
	}
	if p.NotesSet {
		d.Notes = p.Notes
	}
}

func scanDebt(row pgx.Row) (*Debt, error) {
	var d Debt
	if err := row.Scan(
		&d.ID, &d.AccountID, &d.Name, &d.DebtType, &d.StartDate,
		&d.InitialAmount, &d.InterestRate, &d.Currency, &d.Notes,
		&d.IsActive, &d.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &d, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr == nil {
		return false
	}
	return pgErr.Code == pgerrcode.UniqueViolation
}

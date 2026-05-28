package holdings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
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

	// PLN-converted fields. Populated when Running.HasPLN is true and an FX
	// rate exists for LatestQuoteDate. LatestQuoteRatePLN is the rate used
	// for MarketValuePLN/UnrealizedGainPLN — exposed so the UI can show it.
	HasPLN             bool
	MarketValuePLN     decimal.Decimal
	UnrealizedGainPLN  decimal.Decimal
	LatestQuoteRatePLN decimal.Decimal

	// Accounts is the per-account breakdown for this security. The same
	// position appears in each account that has lots; the consolidated
	// fields above are the sum/weighted-average across all of them.
	Accounts []AccountPosition
}

// AccountPosition is the slice of a HoldingRow attributable to one account.
// Cost basis is the weighted average across that account's lots only;
// LatestQuote and FX rate are shared with the consolidated row.
type AccountPosition struct {
	AccountID         int
	AccountName       string
	OwnerUserID       int
	Running           Running
	MarketValue       decimal.Decimal
	UnrealizedGain    decimal.Decimal
	HasPLN            bool
	MarketValuePLN    decimal.Decimal
	UnrealizedGainPLN decimal.Decimal
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

// UpsertAutomatedQuote writes a quote without overwriting a manual entry for
// the same (security_id, date). Used by scheduled fetchers (Stooq) so the
// user's explicit price wins over any provider price for the same day.
// Returns true when the row was written or refreshed, false when a manual
// quote already occupied the slot.
func (s *Store) UpsertAutomatedQuote(ctx context.Context, q PriceQuote) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO price_quotes (security_id, date, price, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (security_id, date) DO UPDATE
		SET price = EXCLUDED.price, source = EXCLUDED.source
		WHERE price_quotes.source = EXCLUDED.source`,
		q.SecurityID, q.Date, q.Price, q.Source,
	)
	if err != nil {
		return false, fmt.Errorf("upsert automated quote: %w", err)
	}
	return tag.RowsAffected() > 0, nil
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

// LastStooqQuoteDate returns the latest stored stooq quote date for a
// security. Zero time when none exist yet — caller uses min(lot.date) as
// the backfill start.
func (s *Store) LastStooqQuoteDate(ctx context.Context, securityID int) (time.Time, error) {
	var t *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT MAX(date) FROM price_quotes
		WHERE security_id = $1 AND source = 'stooq'`, securityID).Scan(&t)
	if err != nil {
		return time.Time{}, fmt.Errorf("last stooq quote date: %w", err)
	}
	if t == nil {
		return time.Time{}, nil
	}
	return *t, nil
}

// FirstLotDate returns the earliest lot date for a security. Zero time when
// there are no lots — used as the backfill start when no stooq history exists.
func (s *Store) FirstLotDate(ctx context.Context, securityID int) (time.Time, error) {
	var t *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT MIN(date) FROM lots WHERE security_id = $1`, securityID).Scan(&t)
	if err != nil {
		return time.Time{}, fmt.Errorf("first lot date: %w", err)
	}
	if t == nil {
		return time.Time{}, nil
	}
	return *t, nil
}

// BulkUpsertAutomatedQuotes writes many quotes in a single transaction,
// preserving rows tagged source != EXCLUDED.source (i.e. manual entries).
// Returns the count of rows actually written or refreshed.
func (s *Store) BulkUpsertAutomatedQuotes(ctx context.Context, rows []PriceQuote) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin bulk upsert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	const q = `
		INSERT INTO price_quotes (security_id, date, price, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (security_id, date) DO UPDATE
		SET price = EXCLUDED.price, source = EXCLUDED.source
		WHERE price_quotes.source = EXCLUDED.source`
	written := 0
	for i := range rows {
		r := &rows[i]
		tag, err := tx.Exec(ctx, q, r.SecurityID, r.Date, r.Price, r.Source)
		if err != nil {
			return 0, fmt.Errorf("bulk upsert row %d: %w", i, err)
		}
		if tag.RowsAffected() > 0 {
			written++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit bulk upsert: %w", err)
	}
	return written, nil
}

// Valuator binds a Store + RateProvider so callers that don't want to deal
// with FX wiring (notably accounts.Handler) can get per-account PLN values
// through one method. Implements accounts.HoldingsValuator.
type Valuator struct {
	store *Store
	rates RateProvider
}

// NewValuator wires a store + rates. rates may be nil — the valuator then
// returns native-currency-zero for every account (PLN conversion needs FX).
func NewValuator(store *Store, rates RateProvider) *Valuator {
	return &Valuator{store: store, rates: rates}
}

// AccountValuesPLN returns the live PLN market value for every account that
// holds at least one open position with FX data. Sums per-security MV PLN
// across the account's lots.
func (v *Valuator) AccountValuesPLN(ctx context.Context) (map[int]decimal.Decimal, error) {
	rows, err := v.store.Holdings(ctx, v.rates)
	if err != nil {
		return nil, err
	}
	out := map[int]decimal.Decimal{}
	for i := range rows {
		accs := rows[i].Accounts
		for j := range accs {
			acct := &accs[j]
			if !acct.HasPLN {
				continue
			}
			out[acct.AccountID] = out[acct.AccountID].Add(acct.MarketValuePLN)
		}
	}
	return out, nil
}

// LatestStooqQuoteTime returns the most recent created_at across price_quotes
// rows tagged source='stooq'. Zero time when no stooq quotes exist yet.
// Used by the quotes scheduler to decide whether to refresh on startup.
func (s *Store) LatestStooqQuoteTime(ctx context.Context) (time.Time, error) {
	var t *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT MAX(created_at) FROM price_quotes WHERE source = 'stooq'`).Scan(&t)
	if err != nil {
		return time.Time{}, fmt.Errorf("latest stooq quote time: %w", err)
	}
	if t == nil {
		return time.Time{}, nil
	}
	return *t, nil
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

// accountMeta is the slice of accounts data Holdings needs to label the
// per-account breakdown. Keyed by account id.
type accountMeta struct {
	Name        string
	OwnerUserID int
}

func (s *Store) accountMeta(ctx context.Context) (map[int]accountMeta, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, owner_user_id FROM accounts`)
	if err != nil {
		return nil, fmt.Errorf("account meta: %w", err)
	}
	defer rows.Close()
	out := map[int]accountMeta{}
	for rows.Next() {
		var id int
		var name string
		var owner *int // accounts.owner_user_id is nullable
		if err := rows.Scan(&id, &name, &owner); err != nil {
			return nil, fmt.Errorf("scan account meta: %w", err)
		}
		meta := accountMeta{Name: name}
		if owner != nil {
			meta.OwnerUserID = *owner
		}
		out[id] = meta
	}
	return out, rows.Err()
}

// Holdings replays every lot, groups by security, attaches the latest quote,
// and returns one HoldingRow per security with non-zero quantity. When rates
// is non-nil, PLN-converted fields are also populated using trade-date NBP
// rates for cost basis and the latest-quote-date rate for market value.
// Each row also exposes a per-account breakdown via HoldingRow.Accounts.
func (s *Store) Holdings(ctx context.Context, rates RateProvider) ([]HoldingRow, error) {
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
	accounts, err := s.accountMeta(ctx)
	if err != nil {
		return nil, err
	}
	bySec, bySecAcct := groupLots(allLots)
	out := []HoldingRow{}
	for _, sec := range securities {
		row, ok := s.buildHoldingRow(ctx, sec, bySec[sec.ID], bySecAcct[sec.ID], quotes, accounts, rates)
		if !ok {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

// groupLots bins every lot by security_id and by (security_id, account_id).
// Returned in lot insertion order, which matches ListLots' chronological sort.
func groupLots(lots []Lot) (map[int][]Lot, map[int]map[int][]Lot) {
	bySec := map[int][]Lot{}
	bySecAcct := map[int]map[int][]Lot{}
	for i := range lots {
		l := &lots[i]
		bySec[l.SecurityID] = append(bySec[l.SecurityID], *l)
		if bySecAcct[l.SecurityID] == nil {
			bySecAcct[l.SecurityID] = map[int][]Lot{}
		}
		bySecAcct[l.SecurityID][l.AccountID] = append(bySecAcct[l.SecurityID][l.AccountID], *l)
	}
	return bySec, bySecAcct
}

// buildHoldingRow runs ComputeRunningPLN for the consolidated security
// position and then for every account that holds it, returning the assembled
// HoldingRow. Skips (returns false) when the security has no open position
// or its lot history is corrupted.
func (s *Store) buildHoldingRow(
	ctx context.Context,
	sec Security,
	allLots []Lot,
	perAccount map[int][]Lot,
	quotes map[int]PriceQuote,
	accounts map[int]accountMeta,
	rates RateProvider,
) (HoldingRow, bool) {
	run, runErr := ComputeRunningPLN(ctx, allLots, sec.Currency, rates)
	if runErr != nil {
		slog.Default().Warn("holdings: skipping security with bad lot history",
			"security_id", sec.ID, "symbol", sec.Symbol, "err", runErr)
		return HoldingRow{}, false
	}
	if run.Quantity.IsZero() && run.RealizedGain.IsZero() {
		return HoldingRow{}, false
	}
	row := HoldingRow{Security: sec, Running: run}
	if q, ok := quotes[sec.ID]; ok {
		row.LatestQuote = q.Price
		d := q.Date
		row.LatestQuoteDate = &d
	}
	row.MarketValue = MarketValue(run, row.LatestQuote)
	row.UnrealizedGain = UnrealizedGain(run, row.LatestQuote)
	s.fillPLNValuation(ctx, &row, rates)

	row.Accounts = s.buildAccountPositions(ctx, sec, perAccount, accounts, row.LatestQuote, rates, row.LatestQuoteRatePLN, row.HasPLN)
	return row, true
}

// buildAccountPositions runs the cost-basis walk per account and returns one
// AccountPosition per account that currently holds the security. Reuses the
// consolidated row's FX rate (already looked up against latest-quote-date)
// so per-account PLN math is consistent with the consolidated total.
func (s *Store) buildAccountPositions(
	ctx context.Context,
	sec Security,
	perAccount map[int][]Lot,
	accounts map[int]accountMeta,
	latestQuote decimal.Decimal,
	rates RateProvider,
	rowRatePLN decimal.Decimal,
	rowHasPLN bool,
) []AccountPosition {
	out := []AccountPosition{}
	for accountID, lots := range perAccount {
		runAcct, err := ComputeRunningPLN(ctx, lots, sec.Currency, rates)
		if err != nil {
			slog.Default().Warn("holdings: skipping account with bad lot history",
				"security_id", sec.ID, "account_id", accountID, "err", err)
			continue
		}
		if runAcct.Quantity.IsZero() && runAcct.RealizedGain.IsZero() {
			continue
		}
		meta := accounts[accountID]
		pos := AccountPosition{
			AccountID:      accountID,
			AccountName:    meta.Name,
			OwnerUserID:    meta.OwnerUserID,
			Running:        runAcct,
			MarketValue:    MarketValue(runAcct, latestQuote),
			UnrealizedGain: UnrealizedGain(runAcct, latestQuote),
		}
		if rowHasPLN && runAcct.HasPLN && !rowRatePLN.IsZero() {
			mvPLN := pos.MarketValue.Mul(rowRatePLN)
			if sec.Currency == "PLN" {
				mvPLN = pos.MarketValue
			}
			pos.HasPLN = true
			pos.MarketValuePLN = mvPLN
			pos.UnrealizedGainPLN = mvPLN.Sub(runAcct.CostBasisPLN)
		}
		out = append(out, pos)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OwnerUserID != out[j].OwnerUserID {
			return out[i].OwnerUserID < out[j].OwnerUserID
		}
		return out[i].AccountID < out[j].AccountID
	})
	return out
}

// fillPLNValuation converts MarketValue + UnrealizedGain to PLN using the FX
// rate for the latest-quote date (or today if no quote). Leaves PLN fields
// zero on lookup miss — UI degrades to native-currency display.
func (s *Store) fillPLNValuation(ctx context.Context, row *HoldingRow, rates RateProvider) {
	if rates == nil || !row.Running.HasPLN {
		return
	}
	rateDate := time.Now().UTC()
	if row.LatestQuoteDate != nil {
		rateDate = *row.LatestQuoteDate
	}
	rate, err := rates.GetRateToPLN(ctx, row.Security.Currency, rateDate)
	if err != nil {
		slog.Default().Warn("holdings: fx lookup failed for market value",
			"security_id", row.Security.ID, "symbol", row.Security.Symbol, "err", err)
		return
	}
	mv := row.MarketValue
	mvPLN, ok := fx.ToPLN(&mv, row.Security.Currency, rate)
	if !ok {
		return
	}
	row.HasPLN = true
	row.MarketValuePLN = mvPLN
	row.LatestQuoteRatePLN = rate.Rate
	row.UnrealizedGainPLN = mvPLN.Sub(row.Running.CostBasisPLN)
}

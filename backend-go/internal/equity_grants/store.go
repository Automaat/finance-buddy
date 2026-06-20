package equitygrants

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/dbutil"
)

// EquityGrant mirrors backend/app/models/equity_grant.EquityGrant.
// OwnerUserID is the owning household member; nil means jointly owned.
type EquityGrant struct {
	ID                     int
	GrantDate              time.Time
	Type                   string
	Company                string
	OwnerUserID            *int
	TotalShares            int
	StrikePrice            *decimal.Decimal
	Currency               string
	VestStartDate          time.Time
	VestCliffMonths        int
	VestTotalMonths        int
	VestFrequency          string
	VestCustomSchedule     []CustomScheduleEntry
	RequiresLiquidityEvent bool
	LiquidityEventDate     *time.Time
	TaxTreatment           string
	Notes                  *string
	IsActive               bool
	CreatedAt              time.Time
}

// ErrNotFound is returned when a row is missing or soft-deleted.
var ErrNotFound = errors.New("equity grant not found")

// InvariantError is returned by Update when applying a patch violates a
// cross-field invariant (cliff > total, option without strike, etc). The
// handler maps it to a 422.
type InvariantError struct {
	Field string
	Msg   string
}

func (e *InvariantError) Error() string {
	return e.Field + ": " + e.Msg
}

// ListFilter narrows the active rows.
type ListFilter struct {
	OwnerUserID *int
	Company     *string
}

// Store is the persistence boundary.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `
	id, grant_date, type, company, owner_user_id, total_shares, strike_price, currency,
	vest_start_date, vest_cliff_months, vest_total_months, vest_frequency,
	vest_custom_schedule, requires_liquidity_event, liquidity_event_date,
	tax_treatment, notes, is_active, created_at
`

// List returns active grants ordered by grant_date desc + distinct active companies.
func (s *Store) List(ctx context.Context, f ListFilter) ([]EquityGrant, []string, error) {
	where := dbutil.NewWhereBuilder("is_active = true")
	if f.OwnerUserID != nil {
		where.Add("owner_user_id = $%d", *f.OwnerUserID)
	}
	if f.Company != nil {
		where.Add("company = $%d", *f.Company)
	}
	query := `SELECT ` + selectColumns + ` FROM equity_grants WHERE ` +
		where.SQL() + ` ORDER BY grant_date DESC`

	rows, err := s.pool.Query(ctx, query, where.Args()...)
	if err != nil {
		return nil, nil, fmt.Errorf("select equity grants: %w", err)
	}
	defer rows.Close()
	out := []EquityGrant{}
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan equity grant: %w", err)
		}
		out = append(out, *g)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate equity grants: %w", err)
	}
	companies, err := s.activeCompanies(ctx)
	if err != nil {
		return nil, nil, err
	}
	return out, companies, nil
}

// Get returns the active grant or ErrNotFound.
func (s *Store) Get(ctx context.Context, id int) (*EquityGrant, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM equity_grants WHERE id = $1`, id,
	)
	g, err := scanGrant(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "select equity grant")
	}
	if !g.IsActive {
		return nil, ErrNotFound
	}
	return g, nil
}

// Create inserts a row.
func (s *Store) Create(ctx context.Context, g *EquityGrant) (*EquityGrant, error) {
	customJSON, err := encodeSchedule(g.VestCustomSchedule)
	if err != nil {
		return nil, fmt.Errorf("encode schedule: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO equity_grants (
			grant_date, type, company, owner_user_id, total_shares,
			strike_price, currency,
			vest_start_date, vest_cliff_months, vest_total_months, vest_frequency,
			vest_custom_schedule, requires_liquidity_event, liquidity_event_date,
			tax_treatment, notes, is_active, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			true, $17
		) RETURNING `+selectColumns,
		g.GrantDate, g.Type, g.Company, g.OwnerUserID, g.TotalShares, g.StrikePrice, g.Currency,
		g.VestStartDate, g.VestCliffMonths, g.VestTotalMonths, g.VestFrequency,
		customJSON, g.RequiresLiquidityEvent, g.LiquidityEventDate,
		g.TaxTreatment, g.Notes, time.Now().UTC(),
	)
	return scanGrant(row)
}

// UpdatePatch is the sparse update set; nil means "leave alone".
type UpdatePatch struct {
	GrantDate              *time.Time
	Type                   *string
	Company                *string
	OwnerUserIDSet         bool
	OwnerUserID            *int
	TotalShares            *int
	StrikePrice            *decimal.Decimal
	Currency               *string
	VestStartDate          *time.Time
	VestCliffMonths        *int
	VestTotalMonths        *int
	VestFrequency          *string
	VestCustomScheduleSet  bool
	VestCustomSchedule     []CustomScheduleEntry
	RequiresLiquidityEvent *bool
	LiquidityEventDate     *time.Time
	TaxTreatment           *string
	Notes                  *string
}

// Update applies the patch.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*EquityGrant, error) {
	g, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	applyPatch(g, p)
	// Mirror Python's cross-field invariants on the merged record.
	// RSU clears any lingering strike — Python's schema enforces this in
	// the model validator; we do it post-patch.
	if g.Type == "rsu" {
		g.StrikePrice = nil
	}
	if g.VestCliffMonths > g.VestTotalMonths {
		return nil, &InvariantError{
			Field: "vest_cliff_months",
			Msg:   "Cliff months cannot exceed total vesting months",
		}
	}
	if g.Type == "option" && g.StrikePrice == nil {
		return nil, &InvariantError{
			Field: "strike_price",
			Msg:   "Stock options require a strike price",
		}
	}
	customJSON, err := encodeSchedule(g.VestCustomSchedule)
	if err != nil {
		return nil, fmt.Errorf("encode schedule: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE equity_grants SET
			grant_date = $1, type = $2, company = $3,
			owner_user_id = $4,
			total_shares = $5, strike_price = $6, currency = $7,
			vest_start_date = $8, vest_cliff_months = $9, vest_total_months = $10,
			vest_frequency = $11, vest_custom_schedule = $12,
			requires_liquidity_event = $13, liquidity_event_date = $14,
			tax_treatment = $15, notes = $16
		WHERE id = $17 AND is_active = true
		RETURNING `+selectColumns,
		g.GrantDate, g.Type, g.Company, g.OwnerUserID,
		g.TotalShares, g.StrikePrice, g.Currency,
		g.VestStartDate, g.VestCliffMonths, g.VestTotalMonths,
		g.VestFrequency, customJSON,
		g.RequiresLiquidityEvent, g.LiquidityEventDate,
		g.TaxTreatment, g.Notes, id,
	)
	updated, err := scanGrant(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "update equity grant")
	}
	return updated, nil
}

// Delete soft-deletes.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE equity_grants SET is_active = false WHERE id = $1 AND is_active = true`,
		id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete equity grant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) activeCompanies(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT company FROM equity_grants
		 WHERE is_active = true ORDER BY company`,
	)
	if err != nil {
		return nil, fmt.Errorf("select active companies: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate companies: %w", err)
	}
	return out, nil
}

func scanGrant(row pgx.Row) (*EquityGrant, error) {
	var g EquityGrant
	var customJSON []byte
	err := row.Scan(
		&g.ID, &g.GrantDate, &g.Type, &g.Company, &g.OwnerUserID,
		&g.TotalShares, &g.StrikePrice, &g.Currency,
		&g.VestStartDate, &g.VestCliffMonths, &g.VestTotalMonths,
		&g.VestFrequency, &customJSON,
		&g.RequiresLiquidityEvent, &g.LiquidityEventDate,
		&g.TaxTreatment, &g.Notes, &g.IsActive, &g.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(customJSON) > 0 && string(customJSON) != "null" {
		var parsed []CustomScheduleEntry
		if err := json.Unmarshal(customJSON, &parsed); err != nil {
			return nil, fmt.Errorf("decode vest_custom_schedule: %w", err)
		}
		g.VestCustomSchedule = parsed
	}
	return &g, nil
}

// encodeSchedule returns the JSON encoding of entries, or nil bytes to write
// SQL NULL into the vest_custom_schedule column when there is no schedule.
func encodeSchedule(entries []CustomScheduleEntry) ([]byte, error) {
	if entries == nil {
		return nil, nil
	}
	return json.Marshal(entries)
}

func applyPatch(g *EquityGrant, p UpdatePatch) {
	if p.GrantDate != nil {
		g.GrantDate = *p.GrantDate
	}
	if p.Type != nil {
		g.Type = *p.Type
	}
	if p.Company != nil {
		g.Company = *p.Company
	}
	if p.OwnerUserIDSet {
		g.OwnerUserID = p.OwnerUserID
	}
	if p.TotalShares != nil {
		g.TotalShares = *p.TotalShares
	}
	if p.StrikePrice != nil {
		g.StrikePrice = p.StrikePrice
	}
	if p.Currency != nil {
		g.Currency = *p.Currency
	}
	if p.VestStartDate != nil {
		g.VestStartDate = *p.VestStartDate
	}
	if p.VestCliffMonths != nil {
		g.VestCliffMonths = *p.VestCliffMonths
	}
	if p.VestTotalMonths != nil {
		g.VestTotalMonths = *p.VestTotalMonths
	}
	if p.VestFrequency != nil {
		g.VestFrequency = *p.VestFrequency
	}
	if p.VestCustomScheduleSet {
		g.VestCustomSchedule = p.VestCustomSchedule
	}
	if p.RequiresLiquidityEvent != nil {
		g.RequiresLiquidityEvent = *p.RequiresLiquidityEvent
	}
	if p.LiquidityEventDate != nil {
		g.LiquidityEventDate = p.LiquidityEventDate
	}
	if p.TaxTreatment != nil {
		g.TaxTreatment = *p.TaxTreatment
	}
	if p.Notes != nil {
		g.Notes = p.Notes
	}
}

package bonds

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/dbutil"
)

// Store is the persistence boundary for treasury_bonds + the bond-side CPI
// lookup the calculator needs.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `id, type, series, face_value, purchase_date, owner_user_id,
		first_year_rate, margin, capitalize, is_active, created_at`

// EnsureSchema creates the treasury_bonds table if missing. Idempotent so it
// can run on every backend start — schema.sql only seeds empty databases, so
// existing prod installs need the additive DDL here too.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS treasury_bonds (
			id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			type varchar(8) NOT NULL,
			series varchar(64) NOT NULL,
			face_value numeric(15,2) NOT NULL,
			purchase_date date NOT NULL,
			owner_user_id integer,
			first_year_rate numeric(8,4) NOT NULL,
			margin numeric(8,4) NOT NULL DEFAULT 0,
			capitalize boolean NOT NULL DEFAULT true,
			is_active boolean NOT NULL DEFAULT true,
			created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			CONSTRAINT treasury_bonds_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT
		)`); err != nil {
		return fmt.Errorf("ensure treasury_bonds table: %w", err)
	}
	return nil
}

// List returns every active treasury bond ordered by purchase_date desc.
func (s *Store) List(ctx context.Context) ([]TreasuryBond, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+selectColumns+` FROM treasury_bonds
		 WHERE is_active = true
		 ORDER BY purchase_date DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list treasury bonds: %w", err)
	}
	defer rows.Close()
	out := []TreasuryBond{}
	for rows.Next() {
		b, err := scanBond(rows)
		if err != nil {
			return nil, fmt.Errorf("scan treasury bond: %w", err)
		}
		out = append(out, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate treasury bonds: %w", err)
	}
	return out, nil
}

// Get returns one bond by id; ErrNotFound when missing or soft-deleted.
func (s *Store) Get(ctx context.Context, id int) (*TreasuryBond, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM treasury_bonds WHERE id = $1 AND is_active = true`, id,
	)
	b, err := scanBond(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "get treasury bond")
	}
	return b, nil
}

// Create inserts a new treasury bond.
func (s *Store) Create(ctx context.Context, b *TreasuryBond) (*TreasuryBond, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO treasury_bonds (
			type, series, face_value, purchase_date, owner_user_id,
			first_year_rate, margin, capitalize, is_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9)
		RETURNING `+selectColumns,
		string(b.Type), b.Series, b.FaceValue, b.PurchaseDate, b.OwnerUserID,
		b.FirstYearRate, b.Margin, b.Capitalize, time.Now().UTC(),
	)
	created, err := scanBond(row)
	if err != nil {
		return nil, fmt.Errorf("insert treasury bond: %w", err)
	}
	return created, nil
}

// UpdatePatch is the sparse update; only set fields are applied. OwnerUserID
// uses the explicit-null pattern (Set=true + nil) so the caller can clear
// it.
type UpdatePatch struct {
	Type           *BondType
	Series         *string
	FaceValue      *decimal.Decimal
	PurchaseDate   *time.Time
	OwnerUserIDSet bool
	OwnerUserID    *int
	FirstYearRate  *decimal.Decimal
	Margin         *decimal.Decimal
	Capitalize     *bool
}

// Update applies the patch and returns the refreshed bond.
func (s *Store) Update(ctx context.Context, id int, p UpdatePatch) (*TreasuryBond, error) {
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
		`SELECT `+selectColumns+` FROM treasury_bonds WHERE id = $1 AND is_active = true FOR UPDATE`, id,
	)
	b, err := scanBond(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "lock treasury bond")
	}
	if p.Type != nil {
		b.Type = *p.Type
	}
	if p.Series != nil {
		b.Series = *p.Series
	}
	if p.FaceValue != nil {
		b.FaceValue = *p.FaceValue
	}
	if p.PurchaseDate != nil {
		b.PurchaseDate = *p.PurchaseDate
	}
	if p.OwnerUserIDSet {
		b.OwnerUserID = p.OwnerUserID
	}
	if p.FirstYearRate != nil {
		b.FirstYearRate = *p.FirstYearRate
	}
	if p.Margin != nil {
		b.Margin = *p.Margin
	}
	if p.Capitalize != nil {
		b.Capitalize = *p.Capitalize
	}
	row = tx.QueryRow(ctx, `
		UPDATE treasury_bonds SET
			type = $1, series = $2, face_value = $3, purchase_date = $4,
			owner_user_id = $5, first_year_rate = $6, margin = $7, capitalize = $8
		WHERE id = $9
		RETURNING `+selectColumns,
		string(b.Type), b.Series, b.FaceValue, b.PurchaseDate, b.OwnerUserID,
		b.FirstYearRate, b.Margin, b.Capitalize, id,
	)
	updated, err := scanBond(row)
	if err != nil {
		return nil, fmt.Errorf("update treasury bond: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update tx: %w", err)
	}
	committed = true
	return updated, nil
}

// Delete soft-deletes the bond by toggling is_active. Hard delete would
// orphan future audit references; soft matches the assets/accounts pattern.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE treasury_bonds SET is_active = false WHERE id = $1 AND is_active = true`, id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete treasury bond: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TotalCurrentValue sums CurrentValue across every active bond for the
// dashboard "Obligacje skarbowe" tile. yoyByYear is the cpi_index map.
func TotalCurrentValue(bonds []TreasuryBond, yoyByYear map[int]decimal.Decimal, now time.Time) decimal.Decimal {
	total := decimal.Zero
	for i := range bonds {
		total = total.Add(CurrentValue(&bonds[i], yoyByYear, now))
	}
	return total
}

func scanBond(row pgx.Row) (*TreasuryBond, error) {
	var b TreasuryBond
	var typ string
	if err := row.Scan(
		&b.ID, &typ, &b.Series, &b.FaceValue, &b.PurchaseDate, &b.OwnerUserID,
		&b.FirstYearRate, &b.Margin, &b.Capitalize, &b.IsActive, &b.CreatedAt,
	); err != nil {
		return nil, err
	}
	b.Type = BondType(typ)
	return &b, nil
}

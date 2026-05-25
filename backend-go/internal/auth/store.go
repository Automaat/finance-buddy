package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/dbutil"
)

// User is an application login account and household member. Name/Surname and
// the PPK rates are nullable — older accounts and the seeded admin may not
// have them set yet.
type User struct {
	ID              int
	Username        string
	PasswordHash    string
	IsAdmin         bool
	Name            *string
	Surname         *string
	PPKEmployeeRate *decimal.Decimal
	PPKEmployerRate *decimal.Decimal
	CreatedAt       time.Time
}

// CreateParams is the input for creating a user.
type CreateParams struct {
	Username        string
	PasswordHash    string
	Name            *string
	Surname         *string
	PPKEmployeeRate *decimal.Decimal
	PPKEmployerRate *decimal.Decimal
}

// UpdateParams is the editable subset of a user (not username/password/admin).
type UpdateParams struct {
	Name            *string
	Surname         *string
	PPKEmployeeRate *decimal.Decimal
	PPKEmployerRate *decimal.Decimal
}

// Sentinel errors so handlers map to HTTP status without sniffing pg text.
var (
	ErrNotFound     = errors.New("user not found")
	ErrNameConflict = errors.New("username already exists")
)

// Store is the persistence boundary for users.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const userColumns = `id, username, password_hash, is_admin, name, surname,
	ppk_employee_rate, ppk_employer_rate, created_at`

// EnsureSchema creates the users table if absent and adds any columns missing
// on databases created before the persona-merge migration.
//
// The baseline schema.sql is applied only to empty databases (see
// db.ApplySchema), so this additive DDL also runs against existing
// production databases and must be idempotent.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			username varchar(100) NOT NULL UNIQUE,
			password_hash text NOT NULL,
			is_admin boolean NOT NULL DEFAULT false,
			name varchar(100),
			surname varchar(100),
			ppk_employee_rate numeric(5,2),
			ppk_employer_rate numeric(5,2),
			created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
		)`); err != nil {
		return fmt.Errorf("ensure users table: %w", err)
	}
	for _, col := range []string{
		"name varchar(100)",
		"surname varchar(100)",
		"ppk_employee_rate numeric(5,2)",
		"ppk_employer_rate numeric(5,2)",
	} {
		if _, err := s.pool.Exec(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS `+col); err != nil {
			return fmt.Errorf("add users column %q: %w", col, err)
		}
	}
	return nil
}

// GetByUsername looks up a user by exact username; ErrNotFound when absent.
func (s *Store) GetByUsername(ctx context.Context, username string) (*User, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE username = $1`, username)
	return scanUser(row)
}

// List returns all users ordered by username.
func (s *Store) List(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return out, nil
}

// OwnerRef is the minimal user identity behind the owner-picker endpoint.
type OwnerRef struct {
	ID       int
	Username string
	Name     *string
}

// ListOwners returns every user's id + names, ordered by username. It is a
// lean query — no password hash loaded — for the frequently-hit, non-admin
// owner-picker endpoint.
func (s *Store) ListOwners(ctx context.Context) ([]OwnerRef, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, username, name FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("select owners: %w", err)
	}
	defer rows.Close()
	out := []OwnerRef{}
	for rows.Next() {
		var o OwnerRef
		if err := rows.Scan(&o.ID, &o.Username, &o.Name); err != nil {
			return nil, fmt.Errorf("scan owner: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owners: %w", err)
	}
	return out, nil
}

// Create inserts a non-admin user; ErrNameConflict on duplicate username.
func (s *Store) Create(ctx context.Context, p CreateParams) (*User, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, is_admin, name, surname,
			ppk_employee_rate, ppk_employer_rate)
		VALUES ($1, $2, false, $3, $4, $5, $6)
		RETURNING `+userColumns,
		p.Username, p.PasswordHash, p.Name, p.Surname, p.PPKEmployeeRate, p.PPKEmployerRate)
	u, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrNameConflict
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}

// Update replaces a user's editable fields (name, surname, PPK rates);
// ErrNotFound if no user has that id.
func (s *Store) Update(ctx context.Context, id int, p UpdateParams) (*User, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE users
		SET name = $1, surname = $2, ppk_employee_rate = $3, ppk_employer_rate = $4
		WHERE id = $5
		RETURNING `+userColumns,
		p.Name, p.Surname, p.PPKEmployeeRate, p.PPKEmployerRate, id)
	u, err := scanUser(row)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// UpsertAdmin creates or refreshes the admin account from configuration.
// Runs on every startup so a changed FB_ADMIN_PASSWORD takes effect. It does
// not touch name/surname/PPK — those are managed through the users UI.
func (s *Store) UpsertAdmin(ctx context.Context, username, passwordHash string) error {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO users (username, password_hash, is_admin)
		VALUES ($1, $2, true)
		ON CONFLICT (username) DO UPDATE
		SET password_hash = EXCLUDED.password_hash, is_admin = true`,
		username, passwordHash); err != nil {
		return fmt.Errorf("upsert admin: %w", err)
	}
	return nil
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin,
		&u.Name, &u.Surname, &u.PPKEmployeeRate, &u.PPKEmployerRate, &u.CreatedAt); err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "scan user")
	}
	return &u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr == nil {
		return false
	}
	return pgErr.Code == pgerrcode.UniqueViolation
}

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
)

// User is an application login account.
type User struct {
	ID           int
	Username     string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
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

const userColumns = `id, username, password_hash, is_admin, created_at`

// EnsureSchema creates the users table if it does not exist.
//
// The baseline schema.sql is applied only to empty databases (see
// db.ApplySchema), so this additive table needs its own idempotent DDL that
// also runs against existing production databases.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			username varchar(100) NOT NULL UNIQUE,
			password_hash text NOT NULL,
			is_admin boolean NOT NULL DEFAULT false,
			created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
		)`); err != nil {
		return fmt.Errorf("ensure users table: %w", err)
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
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return out, nil
}

// Create inserts a non-admin user; ErrNameConflict on duplicate username.
func (s *Store) Create(ctx context.Context, username, passwordHash string) (*User, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, is_admin)
		VALUES ($1, $2, false)
		RETURNING `+userColumns,
		username, passwordHash)
	u, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrNameConflict
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}

// UpsertAdmin creates or refreshes the admin account from configuration.
// Runs on every startup so a changed FB_ADMIN_PASSWORD takes effect.
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
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
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

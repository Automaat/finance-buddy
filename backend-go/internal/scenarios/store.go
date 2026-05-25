// Package scenarios implements the /api/scenarios endpoints — saved
// simulation input bundles.
//
// inputs_json is opaque to the backend: the simulations form decides what
// to serialize, the backend round-trips it. Adding fields to the form
// doesn't require a schema migration.
package scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/finance-buddy/backend-go/internal/dbutil"
)

// Scenario is the persisted form of one saved simulation input bundle.
type Scenario struct {
	ID         int
	Name       string
	Kind       string
	InputsJSON json.RawMessage
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ErrNotFound is returned when no row matches the supplied id.
var ErrNotFound = errors.New("scenario not found")

// Store is the persistence boundary for simulation_scenarios.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `
	id, name, kind, inputs_json, created_at, updated_at
`

// List returns every scenario sorted by updated_at desc, optionally
// filtered by kind. An empty kind returns all scenarios.
func (s *Store) List(ctx context.Context, kind string) ([]Scenario, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if kind == "" {
		rows, err = s.pool.Query(ctx,
			`SELECT `+selectColumns+` FROM simulation_scenarios ORDER BY updated_at DESC, id DESC`,
		)
	} else {
		rows, err = s.pool.Query(ctx,
			`SELECT `+selectColumns+` FROM simulation_scenarios WHERE kind = $1 ORDER BY updated_at DESC, id DESC`,
			kind,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("select scenarios: %w", err)
	}
	defer rows.Close()
	out := []Scenario{}
	for rows.Next() {
		sc, err := scanScenario(rows)
		if err != nil {
			return nil, fmt.Errorf("scan scenario: %w", err)
		}
		out = append(out, *sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scenarios: %w", err)
	}
	return out, nil
}

// Get returns one scenario or ErrNotFound.
func (s *Store) Get(ctx context.Context, id int) (*Scenario, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+selectColumns+` FROM simulation_scenarios WHERE id = $1`, id,
	)
	sc, err := scanScenario(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "select scenario")
	}
	return sc, nil
}

// Create inserts a new scenario.
func (s *Store) Create(ctx context.Context, name, kind string, inputs json.RawMessage) (*Scenario, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO simulation_scenarios (name, kind, inputs_json)
		VALUES ($1, $2, $3)
		RETURNING `+selectColumns,
		name, kind, inputs,
	)
	created, err := scanScenario(row)
	if err != nil {
		return nil, fmt.Errorf("insert scenario: %w", err)
	}
	return created, nil
}

// Update replaces name and inputs_json. kind is immutable — changing it would
// stash a scenario in a list it doesn't belong to.
func (s *Store) Update(ctx context.Context, id int, name string, inputs json.RawMessage) (*Scenario, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE simulation_scenarios
		SET name = $1, inputs_json = $2, updated_at = (now() at time zone 'utc')
		WHERE id = $3
		RETURNING `+selectColumns,
		name, inputs, id,
	)
	updated, err := scanScenario(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "update scenario")
	}
	return updated, nil
}

// Clone reads scenario `id`, then inserts a copy with `name` overriding the
// source name. Atomic in a single statement so concurrent deletes can't
// produce a half-clone.
func (s *Store) Clone(ctx context.Context, id int, name string) (*Scenario, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO simulation_scenarios (name, kind, inputs_json)
		SELECT $1, kind, inputs_json
		FROM simulation_scenarios WHERE id = $2
		RETURNING `+selectColumns,
		name, id,
	)
	created, err := scanScenario(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "clone scenario")
	}
	return created, nil
}

// CloneWithSuffix copies row `id` and suffixes the new name with " (copy)",
// truncated so the result fits within maxNameLen. Single statement —
// eliminates the Get+Clone race window the handler used to have when no
// override name was supplied.
func (s *Store) CloneWithSuffix(ctx context.Context, id int) (*Scenario, error) {
	const suffix = " (copy)"
	row := s.pool.QueryRow(ctx, `
		INSERT INTO simulation_scenarios (name, kind, inputs_json)
		SELECT left(name, $2) || $3, kind, inputs_json
		FROM simulation_scenarios WHERE id = $1
		RETURNING `+selectColumns,
		id, maxNameLen-len(suffix), suffix,
	)
	created, err := scanScenario(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "clone scenario")
	}
	return created, nil
}

// Delete removes the scenario (hard delete).
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM simulation_scenarios WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete scenario: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanScenario(row pgx.Row) (*Scenario, error) {
	var sc Scenario
	if err := row.Scan(
		&sc.ID, &sc.Name, &sc.Kind, &sc.InputsJSON, &sc.CreatedAt, &sc.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &sc, nil
}

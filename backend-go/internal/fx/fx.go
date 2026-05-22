// Package fx ports backend/app/services/fx — NBP table A rate lookups with a
// Postgres-backed cache.
//
// PLN passes through as 1. For other currencies, the cache is consulted first;
// on miss or stale (>7 days) the NBP API is queried for a [target-10d, target]
// window. The most recent rate ≤ target wins. Failed network or empty payload
// falls back to whatever's already in cache (or nil if truly absent).
package fx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

const (
	nbpBaseURL        = "https://api.nbp.pl/api/exchangerates/rates/A"
	lookbackDays      = 10
	cacheToleranceDay = 7
	httpTimeout       = 5 * time.Second
)

// Rate represents a cached NBP rate row.
type Rate struct {
	Date    time.Time
	RatePLN decimal.Decimal
}

// errCacheMiss is the sentinel returned by lookupCached when no cached rate
// exists for the requested currency on/before the target date. Promoted to
// a typed value so the lint rule (nilnil) is satisfied without a `nolint`.
var errCacheMiss = errors.New("fx rate cache miss")

// Service is the FX boundary. Cache reads + NBP fetches are routed through it.
type Service struct {
	pool   *pgxpool.Pool
	client *http.Client
	logger *slog.Logger
}

// NewService wires the pool. The HTTP client uses a 5s timeout matching
// Python.
func NewService(pool *pgxpool.Pool, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		pool:   pool,
		client: &http.Client{Timeout: httpTimeout},
		logger: logger,
	}
}

// Result wraps the optional rate so callers don't deal with (nil, nil) — the
// Found flag distinguishes "no rate available" (a normal outcome on first
// access to an exotic currency over a long weekend) from a real DB failure.
type Result struct {
	Rate  decimal.Decimal
	Found bool
}

// GetRateToPLN returns the PLN-per-unit rate for currency on/before onDate.
// PLN returns {1, true}. Returns {_, false} when no rate exists and NBP
// couldn't supply one. An error only surfaces for DB outages — NBP transient
// failures fall back to whatever's in the cache (stale or absent) and don't
// propagate, matching Python's behavior.
func (s *Service) GetRateToPLN(ctx context.Context, currency string, onDate time.Time) (Result, error) {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if code == "PLN" {
		return Result{Rate: decimal.NewFromInt(1), Found: true}, nil
	}
	target := onDate.UTC()
	if target.IsZero() {
		target = time.Now().UTC()
	}
	target = truncateDay(target)

	cached, err := s.lookupCached(ctx, code, target)
	if err != nil && !errors.Is(err, errCacheMiss) {
		return Result{}, fmt.Errorf("fx cache lookup: %w", err)
	}
	if cached != nil {
		ageDays := int(target.Sub(cached.Date).Hours() / 24)
		// A cached rate close enough to the target wins outright. For a
		// *historical* target (older than the tolerance window) any cached
		// rate ≤ target is final too: NBP only publishes business-day rates,
		// so a prior fetch already persisted everything that exists for that
		// period — re-fetching can never return a rate closer to a fixed
		// past date. Without this guard a list endpoint re-hits NBP on every
		// request for grants/bonuses dated more than a week ago.
		targetIsRecent := time.Since(target) <= cacheToleranceDay*24*time.Hour
		if ageDays <= cacheToleranceDay || !targetIsRecent {
			return Result{Rate: cached.RatePLN, Found: true}, nil
		}
	}

	// Cache miss or stale — fetch the window.
	start := target.AddDate(0, 0, -lookbackDays)
	fetched, ferr := s.fetchRange(ctx, code, start, target)
	if ferr != nil {
		s.logger.Warn("nbp fetch failed", "currency", code, "err", ferr)
		if cached != nil {
			return Result{Rate: cached.RatePLN, Found: true}, nil
		}
		return Result{}, nil
	}
	if len(fetched) == 0 {
		if cached != nil {
			return Result{Rate: cached.RatePLN, Found: true}, nil
		}
		return Result{}, nil
	}
	sort.Slice(fetched, func(i, j int) bool {
		return fetched[i].Date.Before(fetched[j].Date)
	})
	for _, r := range fetched {
		if r.Date.After(target) {
			continue
		}
		if err := s.persist(ctx, code, r); err != nil {
			s.logger.Warn("persist fx rate", "currency", code, "date", r.Date, "err", err)
		}
	}
	refreshed, err := s.lookupCached(ctx, code, target)
	if err != nil && !errors.Is(err, errCacheMiss) {
		return Result{}, fmt.Errorf("fx cache lookup after fetch: %w", err)
	}
	if refreshed != nil {
		return Result{Rate: refreshed.RatePLN, Found: true}, nil
	}
	if cached != nil {
		return Result{Rate: cached.RatePLN, Found: true}, nil
	}
	return Result{}, nil
}

// ToPLN converts amount in currency to PLN using the supplied rate. Returns
// (_, false) when amount is missing or (non-PLN and no rate found). PLN
// amounts pass through unchanged regardless of rate.
func ToPLN(amount *decimal.Decimal, currency string, rate Result) (decimal.Decimal, bool) {
	if amount == nil {
		return decimal.Decimal{}, false
	}
	if strings.EqualFold(currency, "PLN") {
		return *amount, true
	}
	if !rate.Found {
		return decimal.Decimal{}, false
	}
	return amount.Mul(rate.Rate), true
}

func (s *Service) lookupCached(ctx context.Context, code string, onOrBefore time.Time) (*Rate, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT date, rate_pln FROM fx_rates
		WHERE currency = $1 AND date <= $2
		ORDER BY date DESC LIMIT 1`,
		code, onOrBefore,
	)
	var r Rate
	if err := row.Scan(&r.Date, &r.RatePLN); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errCacheMiss
		}
		return nil, fmt.Errorf("select fx_rate: %w", err)
	}
	return &r, nil
}

type nbpResponse struct {
	Rates []nbpRate `json:"rates"`
}

type nbpRate struct {
	EffectiveDate string  `json:"effectiveDate"`
	Mid           float64 `json:"mid"`
}

func (s *Service) fetchRange(ctx context.Context, code string, start, end time.Time) ([]Rate, error) {
	url := fmt.Sprintf(
		"%s/%s/%s/%s/?format=json",
		nbpBaseURL, code, start.Format("2006-01-02"), end.Format("2006-01-02"),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nbp request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return []Rate{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("nbp http %d", resp.StatusCode)
	}
	var body nbpResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode nbp: %w", err)
	}
	out := make([]Rate, 0, len(body.Rates))
	for _, r := range body.Rates {
		d, err := time.Parse("2006-01-02", r.EffectiveDate)
		if err != nil {
			continue
		}
		out = append(out, Rate{
			Date:    d,
			RatePLN: decimal.NewFromFloat(r.Mid),
		})
	}
	return out, nil
}

func (s *Service) persist(ctx context.Context, code string, r Rate) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO fx_rates (date, currency, rate_pln, created_at)
		VALUES ($1, $2, $3, $4)`,
		r.Date, code, r.RatePLN, time.Now().UTC(),
	)
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr != nil && pgErr.Code == pgerrcode.UniqueViolation {
		// Another writer cached the same (date, currency) pair — fine.
		return nil
	}
	return fmt.Errorf("insert fx_rate: %w", err)
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

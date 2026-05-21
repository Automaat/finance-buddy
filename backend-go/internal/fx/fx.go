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

// GetRateToPLN returns the PLN-per-unit rate for currency on/before onDate.
// PLN returns 1. Returns nil if no rate could be found in cache or fetched.
func (s *Service) GetRateToPLN(ctx context.Context, currency string, onDate time.Time) *decimal.Decimal {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if code == "PLN" {
		one := decimal.NewFromInt(1)
		return &one
	}
	target := onDate.UTC()
	if target.IsZero() {
		target = time.Now().UTC()
	}
	target = truncateDay(target)

	cached, _ := s.lookupCached(ctx, code, target)
	if cached != nil {
		ageDays := int(target.Sub(cached.Date).Hours() / 24)
		if ageDays <= cacheToleranceDay {
			rate := cached.RatePLN
			return &rate
		}
	}

	// Cache miss or stale — fetch the window.
	start := target.AddDate(0, 0, -lookbackDays)
	fetched, err := s.fetchRange(ctx, code, start, target)
	if err != nil {
		s.logger.Warn("nbp fetch failed", "currency", code, "err", err)
		if cached != nil {
			rate := cached.RatePLN
			return &rate
		}
		return nil
	}
	if len(fetched) == 0 {
		if cached != nil {
			rate := cached.RatePLN
			return &rate
		}
		return nil
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
	refreshed, _ := s.lookupCached(ctx, code, target)
	if refreshed != nil {
		rate := refreshed.RatePLN
		return &rate
	}
	if cached != nil {
		rate := cached.RatePLN
		return &rate
	}
	return nil
}

// ToPLN converts amount in currency to PLN using the supplied rate. Returns
// nil when amount is missing or (non-PLN and rate is nil). PLN amounts pass
// through unchanged regardless of rate.
func ToPLN(amount *decimal.Decimal, currency string, rate *decimal.Decimal) *decimal.Decimal {
	if amount == nil {
		return nil
	}
	if strings.EqualFold(currency, "PLN") {
		v := *amount
		return &v
	}
	if rate == nil {
		return nil
	}
	v := amount.Mul(*rate)
	return &v
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

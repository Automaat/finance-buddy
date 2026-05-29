package cpi

// FRED fetcher for Polish monthly CPI YoY. FRED's POLCPIALLMINMEI is sourced
// from OECD which mirrors GUS national CPI exactly — matching the Ministry's
// rate-setting input perfectly (Eurostat HICP, our default, drifts 0.1-0.3pp
// because HICP uses a slightly different basket).
//
// Free with one-time API key signup at https://fredaccount.stlouisfed.org/apikey
// (32-char alpha-numeric). When FB_FRED_API_KEY is unset the scheduler falls
// back to Eurostat.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	// FREDObservationsURL is FRED's series-observations endpoint.
	FREDObservationsURL = "https://api.stlouisfed.org/fred/series/observations"
	// FREDPolandCPISeries is the OECD-sourced Polish all-items CPI.
	FREDPolandCPISeries = "POLCPIALLMINMEI"
	// FREDMonthlySource is the source tag stored in cpi_monthly_index.
	FREDMonthlySource = "fred-polcpiallminmei"

	fredHTTPTimeout = 8 * time.Second
)

// ErrFREDMissingKey signals an empty FB_FRED_API_KEY at construction. The
// scheduler treats this as "FRED disabled" and falls back to Eurostat.
var ErrFREDMissingKey = errors.New("cpi: FRED api key required")

// FREDFetcher hits FRED's observations endpoint with units=pc1 to get
// pre-computed YoY percent change.
type FREDFetcher struct {
	client *http.Client
	apiKey string
}

// NewFREDFetcher wires the fetcher. apiKey must be the 32-char key issued
// by FRED; empty returns nil so the scheduler can branch cleanly.
func NewFREDFetcher(apiKey string) *FREDFetcher {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	return &FREDFetcher{
		client: &http.Client{Timeout: fredHTTPTimeout},
		apiKey: apiKey,
	}
}

// fredObservation is one row from the FRED observations endpoint.
type fredObservation struct {
	Date  string `json:"date"`
	Value string `json:"value"` // "." for missing
}

type fredResponse struct {
	Observations []fredObservation `json:"observations"`
}

// FetchSince returns Polish CPI YoY values from `since` (inclusive)
// onwards, sorted ascending by (year, month). `since` is a YYYY-MM-01 in
// any timezone; only the year-month is used.
//
// FRED's units=pc1 returns the YoY percent change directly (e.g. 6.6 for
// +6.6%); we add 100 to match the GUS convention used elsewhere in this
// package (114.4 = +14.4%).
func (f *FREDFetcher) FetchSince(ctx context.Context, since time.Time) ([]MonthYearRate, error) {
	if since.IsZero() {
		return nil, fmt.Errorf("fred: since is zero")
	}
	q := fmt.Sprintf(
		"?series_id=%s&api_key=%s&file_type=json&units=pc1&observation_start=%04d-%02d-01",
		FREDPolandCPISeries, f.apiKey, since.Year(), int(since.Month()),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, FREDObservationsURL+q, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fred fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("fred http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var body fredResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode fred: %w", err)
	}
	return parseFRED(body.Observations)
}

func parseFRED(obs []fredObservation) ([]MonthYearRate, error) {
	out := make([]MonthYearRate, 0, len(obs))
	for _, o := range obs {
		if o.Value == "" || o.Value == "." {
			continue // FRED's sentinel for not-yet-published months
		}
		year, month, err := parseFREDDate(o.Date)
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", o.Date, err)
		}
		yoy, err := decimal.NewFromString(o.Value)
		if err != nil {
			return nil, fmt.Errorf("parse yoy %q: %w", o.Value, err)
		}
		out = append(out, MonthYearRate{
			Year:  year,
			Month: month,
			YoY:   yoy.Add(decimal.NewFromInt(100)),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Year != out[j].Year {
			return out[i].Year < out[j].Year
		}
		return out[i].Month < out[j].Month
	})
	return out, nil
}

func parseFREDDate(s string) (int, int, error) {
	// FRED dates are always YYYY-MM-DD; we only care about year-month.
	parts := strings.SplitN(s, "-", 3)
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("expected YYYY-MM-DD, got %q", s)
	}
	y, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("year %q: %w", parts[0], err)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("month %q: %w", parts[1], err)
	}
	if m < 1 || m > 12 {
		return 0, 0, fmt.Errorf("month %d out of range", m)
	}
	return y, m, nil
}

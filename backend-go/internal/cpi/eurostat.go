package cpi

import (
	"context"
	"encoding/json"
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
	// EurostatHICPURL is the prc_hicp_manr endpoint — monthly HICP annual
	// rate of change for Poland, all-items.
	EurostatHICPURL = "https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/prc_hicp_manr"
	// EurostatMonthlySource is the source tag stored in cpi_monthly_index.
	EurostatMonthlySource = "eurostat-hicp-manr"

	eurostatHTTPTimeout = 8 * time.Second
)

// EurostatHICPFetcher hits Eurostat's open data API for Polish HICP YoY.
// Stateless; reuse one instance per scheduler.
type EurostatHICPFetcher struct {
	client *http.Client
}

// NewEurostatHICPFetcher wires a default HTTP client.
func NewEurostatHICPFetcher() *EurostatHICPFetcher {
	return &EurostatHICPFetcher{
		client: &http.Client{Timeout: eurostatHTTPTimeout},
	}
}

// eurostatPayload mirrors the JSON-stat 2.0 shape returned by the
// dissemination API. Only the fields the parser needs are decoded; the rest
// of the response (label, source, dimension metadata) is ignored.
type eurostatPayload struct {
	Value     map[string]json.RawMessage `json:"value"`
	Dimension struct {
		Time struct {
			Category struct {
				Index map[string]int `json:"index"`
			} `json:"category"`
		} `json:"time"`
	} `json:"dimension"`
}

// FetchSince returns Polish HICP YoY values from `since` (inclusive)
// onwards, sorted ascending by (year, month). `since` is a YYYY-MM-01 in
// any timezone; the day component is ignored.
func (f *EurostatHICPFetcher) FetchSince(ctx context.Context, since time.Time) ([]MonthYearRate, error) {
	if since.IsZero() {
		return nil, fmt.Errorf("eurostat: since is zero")
	}
	q := fmt.Sprintf("?format=JSON&geo=PL&coicop=CP00&sinceTimePeriod=%04d-%02d",
		since.Year(), int(since.Month()))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, EurostatHICPURL+q, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eurostat fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("eurostat http %d", resp.StatusCode)
	}
	var body eurostatPayload
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode eurostat: %w", err)
	}
	return parseEurostat(&body)
}

// parseEurostat walks the JSON-stat dimension index and pairs every time
// label (e.g. "2023-11") with its numeric value at the same index. Missing
// or non-numeric values are skipped — Eurostat occasionally serializes
// null for a still-unpublished month.
func parseEurostat(body *eurostatPayload) ([]MonthYearRate, error) {
	out := make([]MonthYearRate, 0, len(body.Value))
	for label, idx := range body.Dimension.Time.Category.Index {
		raw, ok := body.Value[strconv.Itoa(idx)]
		if !ok || len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			continue
		}
		year, month, err := parseEurostatMonth(label)
		if err != nil {
			return nil, fmt.Errorf("parse time label %q: %w", label, err)
		}
		// Eurostat exposes the YoY % as a plain number (e.g. 6.3 for +6.3%).
		// Convert via string to avoid float→decimal precision drift, then
		// add 100 so the value matches the GUS convention used elsewhere in
		// this package (114.4 = +14.4%).
		dec, err := decimal.NewFromString(strconv.FormatFloat(f, 'f', -1, 64))
		if err != nil {
			return nil, fmt.Errorf("decimal %v: %w", f, err)
		}
		out = append(out, MonthYearRate{
			Year:  year,
			Month: month,
			YoY:   dec.Add(decimal.NewFromInt(100)),
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

func parseEurostatMonth(label string) (int, int, error) {
	// JSON-stat formats Eurostat monthly periods as "YYYY-MM".
	parts := strings.SplitN(label, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected YYYY-MM, got %q", label)
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

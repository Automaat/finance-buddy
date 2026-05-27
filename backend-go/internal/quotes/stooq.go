// Package quotes fetches latest market quotes from external providers and
// upserts them into the price_quotes table.
//
// Stooq's "light" CSV endpoint (q/l/?s=…&f=sd2t2ohlcv&h&e=csv) returns one
// row per symbol with the latest intraday snapshot. UCITS ETF coverage is
// good (isac.uk, cspx.uk, eunl.de, …); prices are in the symbol's native
// trading currency. The security row declares that currency at creation
// time — Stooq's price is stored as-is, no conversion.
package quotes

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	stooqLatestURL  = "https://stooq.com/q/l/"
	stooqHistoryURL = "https://stooq.com/q/d/l/"
	stooqUserAgent  = "finance-buddy/1.0 (+https://finance.mskalski.dev)"
	httpTimeout     = 8 * time.Second
)

// Quote is a single latest snapshot.
type Quote struct {
	Symbol string
	Date   time.Time
	Close  decimal.Decimal
}

// ErrNoData signals Stooq returned a row but with N/D fields — the symbol
// exists in their universe but has no price for today (delisted, market
// closed, or unknown).
var ErrNoData = errors.New("quotes: stooq returned N/D")

// StooqFetcher is the Stooq HTTP client. Stateless; reuse one instance.
// apiKey gates the daily-history endpoint; Latest works without one.
type StooqFetcher struct {
	client *http.Client
	apiKey string
}

// NewStooqFetcher wires the fetcher with default timeout. apiKey can be empty
// — Latest still works (Stooq's intraday endpoint is keyless), but Daily
// will return ErrNoAPIKey.
func NewStooqFetcher(apiKey string) *StooqFetcher {
	return &StooqFetcher{
		client: &http.Client{Timeout: httpTimeout},
		apiKey: strings.TrimSpace(apiKey),
	}
}

// ErrNoAPIKey signals Daily was called without FB_STOOQ_APIKEY configured.
// Stooq gates daily-history downloads behind a captcha-issued key.
var ErrNoAPIKey = errors.New("quotes: stooq apikey required for daily history")

// Latest fetches the latest snapshot for symbol. Returns (Quote, nil) on
// success, (zero, ErrNoData) when Stooq has no data for the symbol, or
// (zero, err) on network / parse failure. The destination URL is built from
// a package constant; the symbol is encoded as a query parameter so taint
// flows only through net/url, not into the request URL path.
func (f *StooqFetcher) Latest(ctx context.Context, symbol string) (Quote, error) {
	sym := strings.ToLower(strings.TrimSpace(symbol))
	if sym == "" {
		return Quote{}, errors.New("quotes: empty symbol")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, stooqLatestURL, http.NoBody)
	if err != nil {
		return Quote{}, fmt.Errorf("build request: %w", err)
	}
	req.URL.RawQuery = url.Values{
		"s": {sym},
		"f": {"sd2t2ohlcv"},
		"h": {""},
		"e": {"csv"},
	}.Encode()
	req.Header.Set("User-Agent", stooqUserAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return Quote{}, fmt.Errorf("stooq request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return Quote{}, fmt.Errorf("stooq http %d", resp.StatusCode)
	}
	return parseStooqLatest(resp.Body, sym)
}

// parseStooqLatest reads the CSV stream and returns the first data row.
// Header: Symbol,Date,Time,Open,High,Low,Close,Volume.
func parseStooqLatest(r io.Reader, sym string) (Quote, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return Quote{}, fmt.Errorf("read header: %w", err)
	}
	idx := map[string]int{}
	for i, name := range header {
		idx[strings.ToLower(strings.TrimSpace(name))] = i
	}
	needed := []string{"date", "close"}
	for _, n := range needed {
		if _, ok := idx[n]; !ok {
			return Quote{}, fmt.Errorf("stooq missing %q column", n)
		}
	}
	row, err := reader.Read()
	if err != nil {
		return Quote{}, fmt.Errorf("read row: %w", err)
	}
	if row[idx["date"]] == "N/D" || row[idx["close"]] == "N/D" {
		return Quote{}, ErrNoData
	}
	d, err := time.Parse("2006-01-02", row[idx["date"]])
	if err != nil {
		return Quote{}, fmt.Errorf("parse date %q: %w", row[idx["date"]], err)
	}
	price, err := decimal.NewFromString(row[idx["close"]])
	if err != nil {
		return Quote{}, fmt.Errorf("parse close %q: %w", row[idx["close"]], err)
	}
	return Quote{
		Symbol: strings.ToUpper(sym),
		Date:   d,
		Close:  price,
	}, nil
}

// Daily fetches the daily history for symbol in [from, to] inclusive from
// Stooq's q/d/l/?i=d endpoint (apikey required). Returns rows sorted by
// date ascending; empty slice when Stooq has no rows in the window.
func (f *StooqFetcher) Daily(ctx context.Context, symbol string, from, to time.Time) ([]Quote, error) {
	if f.apiKey == "" {
		return nil, ErrNoAPIKey
	}
	sym := strings.ToLower(strings.TrimSpace(symbol))
	if sym == "" {
		return nil, errors.New("quotes: empty symbol")
	}
	if to.Before(from) {
		return nil, fmt.Errorf("quotes: to %s before from %s", to.Format("2006-01-02"), from.Format("2006-01-02"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, stooqHistoryURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.URL.RawQuery = url.Values{
		"s":      {sym},
		"i":      {"d"},
		"d1":     {from.Format("20060102")},
		"d2":     {to.Format("20060102")},
		"apikey": {f.apiKey},
	}.Encode()
	req.Header.Set("User-Agent", stooqUserAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stooq daily request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("stooq daily http %d", resp.StatusCode)
	}
	return parseStooqDaily(resp.Body, sym)
}

// parseStooqDaily reads the daily-history CSV. Header:
// Date,Open,High,Low,Close,Volume — no symbol column.
func parseStooqDaily(r io.Reader, sym string) ([]Quote, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return []Quote{}, nil
		}
		return nil, fmt.Errorf("read header: %w", err)
	}
	idx := map[string]int{}
	for i, name := range header {
		idx[strings.ToLower(strings.TrimSpace(name))] = i
	}
	for _, n := range []string{"date", "close"} {
		if _, ok := idx[n]; !ok {
			return nil, fmt.Errorf("stooq daily missing %q column", n)
		}
	}
	out := []Quote{}
	upperSym := strings.ToUpper(sym)
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		if row[idx["date"]] == "" || row[idx["close"]] == "" {
			continue
		}
		d, err := time.Parse("2006-01-02", row[idx["date"]])
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", row[idx["date"]], err)
		}
		price, err := decimal.NewFromString(row[idx["close"]])
		if err != nil {
			return nil, fmt.Errorf("parse close %q: %w", row[idx["close"]], err)
		}
		out = append(out, Quote{Symbol: upperSym, Date: d, Close: price})
	}
	return out, nil
}

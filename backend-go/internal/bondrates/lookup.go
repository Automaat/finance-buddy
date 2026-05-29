// Package bondrates resolves Y1 interest rate + CPI margin for a Polish
// retail treasury bond emission by scraping the public product page on
// obligacjeskarbowe.pl. The Ministry publishes every emission's terms in a
// "List emisyjny" PDF; the same numbers are rendered as HTML on each
// emission's product page in a deterministic format, so we parse the HTML
// instead of OCR-ing the PDF.
//
// No public JSON API exists for this data — dane.gov.pl has no per-emission
// dataset and obligacjeskarbowe.pl ships only HTML/PDF. The product page
// URL pattern + render is stable across types (EDO/COI/TOS/ROR/DOR/ROS/ROD)
// so the same parser handles them all.
package bondrates

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	productBaseURL = "https://www.obligacjeskarbowe.pl"
	userAgent      = "finance-buddy/1.0 (+https://finance.mskalski.dev)"
	httpTimeout    = 8 * time.Second
)

// typePath maps our bond-type tag to the URL slug obligacjeskarbowe.pl uses.
// Keys match holdings.types' constants (uppercase, no diacritics).
var typePath = map[string]string{
	"EDO": "obligacje-10-letnie-edo",
	"COI": "obligacje-4-letnie-coi",
	"TOS": "obligacje-3-letnie-tos",
	"TOZ": "obligacje-3-letnie-tos", // legacy alias — TOZ was renamed to TOS
	"DOR": "obligacje-2-letnie-dor",
	"DOS": "obligacje-2-letnie-dor", // legacy alias — DOS was renamed to DOR
	"ROR": "obligacje-roczne-ror",
	"ROS": "obligacje-6-letnie-ros",
	"ROD": "obligacje-12-letnie-rod",
	"OTS": "obligacje-3-miesieczne-ots",
}

// Rate is the parsed emission terms.
type Rate struct {
	FirstYearRate decimal.Decimal // Y1 fixed coupon (percent, e.g. 7.25)
	Margin        decimal.Decimal // CPI add-on for years 2+ (percent, e.g. 1.25)
}

// ErrUnknownType signals the requested bond type has no mapped URL path.
var ErrUnknownType = errors.New("bondrates: unknown bond type")

// ErrNotFound signals the product page returned 404 — series doesn't exist.
var ErrNotFound = errors.New("bondrates: series not found")

// ErrParse signals the HTML downloaded fine but the rate fields weren't
// where we expected them. Usually means the page layout shifted upstream.
var ErrParse = errors.New("bondrates: rate fields missing from page")

// Fetcher resolves rates for one (type, series) pair.
type Fetcher interface {
	Lookup(ctx context.Context, bondType, series string) (Rate, error)
}

// ObligacjeSkarbowePLFetcher hits the public product page and extracts the
// fixed Y1 rate + CPI margin. Stateless; reuse one instance.
type ObligacjeSkarbowePLFetcher struct {
	client *http.Client
}

// NewObligacjeSkarbowePLFetcher wires the default HTTP client with timeout.
func NewObligacjeSkarbowePLFetcher() *ObligacjeSkarbowePLFetcher {
	return &ObligacjeSkarbowePLFetcher{
		client: &http.Client{Timeout: httpTimeout},
	}
}

// Lookup fetches the rate for one emission. Series is case-insensitive; the
// URL path on obligacjeskarbowe.pl is lowercase.
func (f *ObligacjeSkarbowePLFetcher) Lookup(ctx context.Context, bondType, series string) (Rate, error) {
	path, ok := typePath[strings.ToUpper(strings.TrimSpace(bondType))]
	if !ok {
		return Rate{}, fmt.Errorf("%w: %q", ErrUnknownType, bondType)
	}
	slug := strings.ToLower(strings.TrimSpace(series))
	if slug == "" {
		return Rate{}, errors.New("bondrates: empty series")
	}
	url := fmt.Sprintf("%s/oferta-obligacji/%s/%s/", productBaseURL, path, slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return Rate{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return Rate{}, fmt.Errorf("fetch product page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Rate{}, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return Rate{}, fmt.Errorf("product page http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Rate{}, fmt.Errorf("read body: %w", err)
	}
	return parseRate(string(body))
}

// rateLineRegex matches the rate description line as the 2023+ Ministry
// renders it: "7,25% w pierwszym rocznym okresie odsetkowym, w kolejnych
// ... marża 1,25% + inflacja". Both numbers use the Polish decimal comma.
//
// The leading rate is captured greedily — guarding against single-digit
// integer-only renders ("4%") that pre-2017 emissions sometimes use.
var rateLineRegex = regexp.MustCompile(
	`([0-9]+(?:[,\.][0-9]+)?)\s*%\s*w\s*pierwszym\s*rocznym\s*okresie\s*odsetkowym` +
		`[\s\S]{0,300}?marża\s*([0-9]+(?:[,\.][0-9]+)?)\s*%\s*\+\s*inflacja`,
)

// rateLineRegexLegacy matches the pre-2023 ordering where the phrase
// comes first and the rate after a colon: "w pierwszym rocznym okresie
// odsetkowym: 7,25%, w kolejnych ... marża 1,25% + inflacja". Older EDO
// emissions (EDO1132, EDO1232, …) render this way on their product pages.
var rateLineRegexLegacy = regexp.MustCompile(
	`w\s*pierwszym\s*rocznym\s*okresie\s*odsetkowym\s*:\s*([0-9]+(?:[,\.][0-9]+)?)\s*%` +
		`[\s\S]{0,300}?marża\s*([0-9]+(?:[,\.][0-9]+)?)\s*%\s*\+\s*inflacja`,
)

// parseRate extracts Y1 + margin from the product page HTML. Tries the
// modern phrase-after-rate format first, then falls back to the legacy
// phrase-then-colon-then-rate format pre-2023 emissions use. Both regexes
// missing trips ErrParse rather than silently returning wrong numbers.
func parseRate(html string) (Rate, error) {
	m := rateLineRegex.FindStringSubmatch(html)
	if m == nil {
		m = rateLineRegexLegacy.FindStringSubmatch(html)
	}
	if m == nil {
		return Rate{}, ErrParse
	}
	y1, err := parsePolishDecimal(m[1])
	if err != nil {
		return Rate{}, fmt.Errorf("parse Y1 rate %q: %w", m[1], err)
	}
	margin, err := parsePolishDecimal(m[2])
	if err != nil {
		return Rate{}, fmt.Errorf("parse margin %q: %w", m[2], err)
	}
	return Rate{FirstYearRate: y1, Margin: margin}, nil
}

func parsePolishDecimal(s string) (decimal.Decimal, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	if _, err := strconv.ParseFloat(normalized, 64); err != nil {
		return decimal.Decimal{}, err
	}
	return decimal.NewFromString(normalized)
}

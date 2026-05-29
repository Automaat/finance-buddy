package bondrates

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

const edo0133HTMLSnippet = `
<html>
<body>
  <h1>EDO0133</h1>
  <div class="hero__rate">7,25<sub>%</sub></div>
  <div class="oprocentowanie">
    7,25% w pierwszym rocznym okresie odsetkowym, w kolejnych rocznych okresach odsetkowych: marża 1,25% + inflacja
  </div>
</body>
</html>`

const edo0335HTMLSnippet = `
<p>Oprocentowanie</p>
<p>6,55% w pierwszym rocznym okresie odsetkowym, w kolejnych
   rocznych okresach odsetkowych: marża 2,00% + inflacja</p>`

const coiHTMLSnippet = `
<span>5,90% w pierwszym rocznym okresie odsetkowym, w kolejnych
rocznych okresach odsetkowych: marża 1,25% + inflacja</span>`

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestParseRate_EDO0133(t *testing.T) {
	r, err := parseRate(edo0133HTMLSnippet)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !r.FirstYearRate.Equal(d("7.25")) {
		t.Errorf("Y1 = %s, want 7.25", r.FirstYearRate)
	}
	if !r.Margin.Equal(d("1.25")) {
		t.Errorf("margin = %s, want 1.25", r.Margin)
	}
}

func TestParseRate_EDO0335(t *testing.T) {
	r, err := parseRate(edo0335HTMLSnippet)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !r.FirstYearRate.Equal(d("6.55")) {
		t.Errorf("Y1 = %s, want 6.55", r.FirstYearRate)
	}
	if !r.Margin.Equal(d("2.00")) {
		t.Errorf("margin = %s, want 2.00", r.Margin)
	}
}

func TestParseRate_COIFormat(t *testing.T) {
	r, err := parseRate(coiHTMLSnippet)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !r.FirstYearRate.Equal(d("5.90")) {
		t.Errorf("Y1 = %s, want 5.90", r.FirstYearRate)
	}
}

func TestParseRate_LegacyFormat(t *testing.T) {
	// Pre-2023 emissions render "w pierwszym ... : 7,25%" instead of
	// "7,25% w pierwszym ...". EDO1232 (Dec 2022) is the canonical case.
	html := `<div>w pierwszym rocznym okresie odsetkowym: 7,25%, w kolejnych ` +
		`rocznych okresach odsetkowych: marża 1,25% + inflacja</div>`
	r, err := parseRate(html)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !r.FirstYearRate.Equal(d("7.25")) {
		t.Errorf("Y1 = %s, want 7.25", r.FirstYearRate)
	}
	if !r.Margin.Equal(d("1.25")) {
		t.Errorf("margin = %s, want 1.25", r.Margin)
	}
}

func TestParseRate_MissingFieldsErrs(t *testing.T) {
	_, err := parseRate("<html>nothing relevant here</html>")
	if !errors.Is(err, ErrParse) {
		t.Errorf("want ErrParse, got %v", err)
	}
}

// redirectTransport rewrites every outgoing request to point at target.
// Used to redirect the hardcoded productBaseURL at an httptest server.
type redirectTransport struct{ target *url.URL }

func (r *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.target.Scheme
	req.URL.Host = r.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestLookup_UnknownTypeErrs(t *testing.T) {
	f := NewObligacjeSkarbowePLFetcher()
	_, err := f.Lookup(context.Background(), "XYZ", "XYZ0125")
	if !errors.Is(err, ErrUnknownType) {
		t.Errorf("want ErrUnknownType, got %v", err)
	}
}

func TestLookup_EmptySeriesErrs(t *testing.T) {
	f := NewObligacjeSkarbowePLFetcher()
	_, err := f.Lookup(context.Background(), "EDO", "  ")
	if err == nil {
		t.Errorf("want error for empty series")
	}
}

func TestLookup_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/oferta-obligacji/obligacje-10-letnie-edo/edo0133/"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		_, _ = w.Write([]byte(edo0133HTMLSnippet))
	}))
	defer srv.Close()
	target, _ := url.Parse(srv.URL)
	f := NewObligacjeSkarbowePLFetcher()
	f.client.Transport = &redirectTransport{target: target}
	r, err := f.Lookup(context.Background(), "EDO", "EDO0133")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !r.FirstYearRate.Equal(d("7.25")) || !r.Margin.Equal(d("1.25")) {
		t.Errorf("got Y1=%s margin=%s, want 7.25/1.25", r.FirstYearRate, r.Margin)
	}
}

func TestLookup_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	target, _ := url.Parse(srv.URL)
	f := NewObligacjeSkarbowePLFetcher()
	f.client.Transport = &redirectTransport{target: target}
	_, err := f.Lookup(context.Background(), "EDO", "edo9999")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestLookup_LegacyTypeAliases(t *testing.T) {
	// TOZ → maps to TOS URL slug; DOS → DOR. Just verify the path is built
	// correctly; rate parsing tested separately.
	cases := []struct {
		typ, slug string
	}{
		{"TOZ", "obligacje-3-letnie-tos"},
		{"DOS", "obligacje-2-letnie-dor"},
	}
	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			seen := ""
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen = r.URL.Path
				_, _ = w.Write([]byte(edo0133HTMLSnippet))
			}))
			defer srv.Close()
			target, _ := url.Parse(srv.URL)
			f := NewObligacjeSkarbowePLFetcher()
			f.client.Transport = &redirectTransport{target: target}
			_, _ = f.Lookup(context.Background(), tc.typ, "x0125")
			if !strings.Contains(seen, tc.slug) {
				t.Errorf("path %q does not contain %q", seen, tc.slug)
			}
		})
	}
}

package quotes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestParseStooqLatest_Success(t *testing.T) {
	csv := "Symbol,Date,Time,Open,High,Low,Close,Volume\nISAC.UK,2026-05-27,11:41:18,121.13,121.49,121.10,121.48,21839\n"
	q, err := parseStooqLatest(strings.NewReader(csv), "isac.uk")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if q.Symbol != "ISAC.UK" {
		t.Errorf("Symbol = %q, want ISAC.UK", q.Symbol)
	}
	if !q.Close.Equal(decimal.RequireFromString("121.48")) {
		t.Errorf("Close = %s, want 121.48", q.Close)
	}
	if q.Date.Format("2006-01-02") != "2026-05-27" {
		t.Errorf("Date = %s, want 2026-05-27", q.Date)
	}
}

func TestParseStooqLatest_NoData(t *testing.T) {
	csv := "Symbol,Date,Time,Open,High,Low,Close,Volume\nNONSENSE.XX,N/D,N/D,N/D,N/D,N/D,N/D,N/D\n"
	_, err := parseStooqLatest(strings.NewReader(csv), "nonsense.xx")
	if !errors.Is(err, ErrNoData) {
		t.Errorf("want ErrNoData, got %v", err)
	}
}

// redirectTransport rewrites every outgoing request to point at target.
// Used to redirect Latest's hardcoded Stooq URL at an httptest server.
type redirectTransport struct {
	target *url.URL
}

func (r *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.target.Scheme
	req.URL.Host = r.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestStooqFetcher_Latest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("s"); got != "isac.uk" {
			t.Errorf("symbol = %q, want isac.uk", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("Symbol,Date,Time,Open,High,Low,Close,Volume\nISAC.UK,2026-05-27,11:41:18,121.13,121.49,121.10,121.48,21839\n"))
	}))
	defer srv.Close()
	target, _ := url.Parse(srv.URL)
	f := NewStooqFetcher("")
	f.client.Transport = &redirectTransport{target: target}
	q, err := f.Latest(context.Background(), "ISAC.UK")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !q.Close.Equal(decimal.RequireFromString("121.48")) {
		t.Errorf("Close = %s, want 121.48", q.Close)
	}
}

func TestStooqFetcher_Http500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	target, _ := url.Parse(srv.URL)
	f := NewStooqFetcher("")
	f.client.Transport = &redirectTransport{target: target}
	_, err := f.Latest(context.Background(), "isac.uk")
	if err == nil || !strings.Contains(err.Error(), "stooq http 500") {
		t.Errorf("want http 500 error, got %v", err)
	}
}

func TestStooqFetcher_EmptySymbolError(t *testing.T) {
	f := NewStooqFetcher("")
	_, err := f.Latest(context.Background(), "  ")
	if err == nil {
		t.Errorf("want error for empty symbol")
	}
}

func TestParseStooqDaily_Success(t *testing.T) {
	csv := "Date,Open,High,Low,Close,Volume\n2026-05-20,118.16,119.22,117.55,119,526880\n2026-05-21,118.97,119.45,118.51,118.68,208618\n"
	rows, err := parseStooqDaily(strings.NewReader(csv), "isac.uk")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0].Date.Format("2006-01-02") != "2026-05-20" {
		t.Errorf("row 0 date = %s, want 2026-05-20", rows[0].Date)
	}
	if !rows[0].Close.Equal(decimal.RequireFromString("119")) {
		t.Errorf("row 0 close = %s, want 119", rows[0].Close)
	}
	if rows[1].Symbol != "ISAC.UK" {
		t.Errorf("symbol = %q, want ISAC.UK", rows[1].Symbol)
	}
}

func TestParseStooqDaily_EmptyBody(t *testing.T) {
	rows, err := parseStooqDaily(strings.NewReader(""), "isac.uk")
	if err != nil {
		t.Fatalf("unexpected err on empty body: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("want empty rows on EOF, got %d", len(rows))
	}
}

func TestStooqFetcher_DailyRequiresAPIKey(t *testing.T) {
	f := NewStooqFetcher("")
	_, err := f.Daily(context.Background(), "isac.uk", time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC))
	if !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("want ErrNoAPIKey, got %v", err)
	}
}

func TestStooqFetcher_Daily(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("apikey") != "test-key" {
			t.Errorf("apikey = %q, want test-key", q.Get("apikey"))
		}
		if q.Get("s") != "isac.uk" || q.Get("i") != "d" || q.Get("d1") != "20260520" || q.Get("d2") != "20260527" {
			t.Errorf("query = %v, want s=isac.uk i=d d1=20260520 d2=20260527", q)
		}
		_, _ = w.Write([]byte("Date,Open,High,Low,Close,Volume\n2026-05-20,118.16,119.22,117.55,119,526880\n"))
	}))
	defer srv.Close()
	target, _ := url.Parse(srv.URL)
	f := NewStooqFetcher("test-key")
	f.client.Transport = &redirectTransport{target: target}
	rows, err := f.Daily(context.Background(), "ISAC.UK",
		time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
}

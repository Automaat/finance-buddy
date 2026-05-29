package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestHealthEndpoint(t *testing.T) {
	srv := httptest.NewServer(New(Config{CORSOrigins: "*"}, slog.New(slog.DiscardHandler), Deps{}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/health", http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got := body["status"]; got != "ok" {
		t.Fatalf("status field = %q, want %q", got, "ok")
	}
}

func TestHealthHandlerDBProbe(t *testing.T) {
	cases := []struct {
		name       string
		pool       pinger
		wantStatus int
		wantField  string
	}{
		{"nil pool reports ok", nil, http.StatusOK, "ok"},
		{"healthy ping reports ok", fakePinger{err: nil}, http.StatusOK, "ok"},
		{"failed ping reports 503", fakePinger{err: errors.New("pool down")}, http.StatusServiceUnavailable, "unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := healthHandler(slog.New(slog.DiscardHandler), tc.pool)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody)
			rec := httptest.NewRecorder()
			h(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got := body["status"]; got != tc.wantField {
				t.Fatalf("status field = %q, want %q", got, tc.wantField)
			}
		})
	}
}

func TestMetricsEndpointServesPrometheus(t *testing.T) {
	srv := httptest.NewServer(New(Config{CORSOrigins: "*"}, slog.New(slog.DiscardHandler), Deps{}))
	defer srv.Close()

	// Hit /health first so the request observer records at least one request.
	healthReq, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/health", http.NoBody)
	healthResp, err := http.DefaultClient.Do(healthReq)
	if err != nil {
		t.Fatalf("get /health: %v", err)
	}
	healthResp.Body.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/metrics", http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "fb_http_requests_total") {
		t.Fatalf("expected fb_http_requests_total in /metrics output")
	}
}

func TestSplitOrigins(t *testing.T) {
	// Parity with Python's `settings.cors_origins.split(",")` — verbatim,
	// no trimming, no wildcard fallback.
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty passes through", "", []string{""}},
		{"single origin", "http://localhost:3000", []string{"http://localhost:3000"}},
		{"comma-separated kept verbatim", "http://a, http://b ,http://c", []string{"http://a", " http://b ", "http://c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitOrigins(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("splitOrigins(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

package investment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseScope_DefaultIsAll(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), "GET", "/api/investment/returns", http.NoBody)
	scope, wire, err := parseScope(r)
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	if !scope.All {
		t.Errorf("default scope should be All")
	}
	if wire.Type != "all" {
		t.Errorf("wire type = %q, expected all", wire.Type)
	}
}

func TestParseScope_Account(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), "GET", "/api/investment/returns?scope=account&id=42", http.NoBody)
	scope, wire, err := parseScope(r)
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	if scope.AccountID == nil || *scope.AccountID != 42 {
		t.Errorf("expected AccountID=42, got %v", scope.AccountID)
	}
	if wire.Account == nil || *wire.Account != 42 {
		t.Errorf("wire account = %v, expected 42", wire.Account)
	}
}

func TestParseScope_AccountRejectsBadID(t *testing.T) {
	cases := []string{
		"/api/investment/returns?scope=account",
		"/api/investment/returns?scope=account&id=abc",
		"/api/investment/returns?scope=account&id=0",
		"/api/investment/returns?scope=account&id=-3",
	}
	for _, url := range cases {
		r := httptest.NewRequestWithContext(context.Background(), "GET", url, http.NoBody)
		_, _, err := parseScope(r)
		if err == "" {
			t.Errorf("expected error for %s", url)
		}
	}
}

func TestParseScope_CategoryRequiresValue(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), "GET", "/api/investment/returns?scope=category", http.NoBody)
	_, _, err := parseScope(r)
	if err == "" {
		t.Errorf("expected error when scope=category and value missing")
	}
}

func TestParseScope_WrapperRequiresValue(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), "GET", "/api/investment/returns?scope=wrapper", http.NoBody)
	_, _, err := parseScope(r)
	if err == "" {
		t.Errorf("expected error when scope=wrapper and value missing")
	}
}

func TestParseScope_CategoryHappyPath(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), "GET", "/api/investment/returns?scope=category&value=stock", http.NoBody)
	scope, wire, err := parseScope(r)
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	if scope.Category == nil || *scope.Category != "stock" {
		t.Errorf("category not set")
	}
	if wire.Type != "category" || wire.Value != "stock" {
		t.Errorf("wire mismatch: %+v", wire)
	}
}

func TestParseScope_Unknown(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), "GET", "/api/investment/returns?scope=bogus", http.NoBody)
	_, _, err := parseScope(r)
	if err == "" {
		t.Errorf("expected error for unknown scope")
	}
}

func TestWindowFor_Periods(t *testing.T) {
	asOf := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		period       string
		wantSinceNil bool
		wantSince    string
	}{
		{"1m", false, "2026-04-24"},
		{"3m", false, "2026-02-24"},
		{"ytd", false, "2026-01-01"},
		{"1y", false, "2025-05-24"},
		{"all", true, ""},
		{"unknown", true, ""},
	}
	for _, tc := range cases {
		w := windowFor(tc.period, asOf)
		if tc.wantSinceNil {
			if w.Since != nil {
				t.Errorf("%s: expected nil Since, got %v", tc.period, w.Since)
			}
			continue
		}
		if w.Since == nil {
			t.Errorf("%s: expected non-nil Since", tc.period)
			continue
		}
		got := w.Since.Format("2006-01-02")
		if got != tc.wantSince {
			t.Errorf("%s: Since=%s, want %s", tc.period, got, tc.wantSince)
		}
	}
}

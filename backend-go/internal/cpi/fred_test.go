package cpi

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseFRED(t *testing.T) {
	obs := []fredObservation{
		{Date: "2023-11-01", Value: "6.6"},
		{Date: "2024-11-01", Value: "4.7"},
		{Date: "2025-11-01", Value: "2.9"},
		{Date: "2026-04-01", Value: "."}, // not yet published
	}
	out, err := parseFRED(obs)
	if err != nil {
		t.Fatalf("parseFRED: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("rows = %d, want 3 (sentinel skipped)", len(out))
	}
	// Sorted ascending; YoY stored in GUS form (114.4 == +14.4%).
	if out[0].Year != 2023 || out[0].Month != 11 {
		t.Errorf("row 0 = %v, want 2023-11", out[0])
	}
	if !out[0].YoY.Equal(decimal.RequireFromString("106.6")) {
		t.Errorf("row 0 YoY = %s, want 106.6", out[0].YoY)
	}
	if !out[2].YoY.Equal(decimal.RequireFromString("102.9")) {
		t.Errorf("row 2 YoY = %s, want 102.9", out[2].YoY)
	}
}

func TestNewFREDFetcher_EmptyKeyReturnsNil(t *testing.T) {
	if f := NewFREDFetcher(""); f != nil {
		t.Errorf("empty key should yield nil fetcher")
	}
	if f := NewFREDFetcher("   "); f != nil {
		t.Errorf("whitespace key should yield nil fetcher")
	}
}

func TestNewFREDFetcher_KeyReturnsFetcher(t *testing.T) {
	if f := NewFREDFetcher("abc123"); f == nil {
		t.Errorf("non-empty key should yield fetcher")
	}
}

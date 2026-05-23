package accounts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

func TestIsoNaiveMarshalJSON(t *testing.T) {
	tm := isoNaive(time.Date(2026, 5, 23, 10, 11, 12, 345678000, time.UTC))
	got, err := tm.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `"2026-05-23T10:11:12.345678"`
	if string(got) != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestIsoNaiveZeroValueFormatsAsZero(t *testing.T) {
	tm := isoNaive(time.Time{})
	got, err := tm.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("zero time should still marshal to a quoted string, got %s", got)
	}
}

func TestPyFloatIntegerGetsDotZero(t *testing.T) {
	got, err := pyFloat(5).MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != "5.0" {
		t.Fatalf("got %s want 5.0", got)
	}
}

func TestPyFloatPreservesDecimal(t *testing.T) {
	got, err := pyFloat(5.5).MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != "5.5" {
		t.Fatalf("got %s want 5.5", got)
	}
}

func TestPyFloatNegative(t *testing.T) {
	got, err := pyFloat(-12.34).MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != "-12.34" {
		t.Fatalf("got %s want -12.34", got)
	}
}

func TestToResponseSquareMetersOmitWhenNil(t *testing.T) {
	a := &Account{
		ID: 7, Name: "Konto", Type: "asset", Category: "bank",
		Currency: "PLN", Purpose: "general",
	}
	r := toResponse(a, decimal.NewFromInt(100))
	if r.SquareMeters != nil {
		t.Fatalf("square meters should be nil, got %+v", r.SquareMeters)
	}
	if float64(r.CurrentValue) != 100 {
		t.Fatalf("current value mismatch: %v", r.CurrentValue)
	}
}

func TestToResponseSquareMetersForwarded(t *testing.T) {
	sq := decimal.NewFromFloat(55.5)
	a := &Account{
		ID: 1, Name: "Mieszkanie", Type: "asset", Category: "real_estate",
		Currency: "PLN", Purpose: "general",
		SquareMeters: &sq,
	}
	r := toResponse(a, decimal.Zero)
	if r.SquareMeters == nil || float64(*r.SquareMeters) != 55.5 {
		t.Fatalf("square meters mismatch: %+v", r.SquareMeters)
	}
}

func TestParseIDParamValid(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/42", http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "42")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	id, ok := parseIDParam(rec, req)
	if !ok || id != 42 {
		t.Fatalf("expected id=42 ok=true, got %d %v", id, ok)
	}
}

func TestParseIDParamInvalidWrites422(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/abc", http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	_, ok := parseIDParam(rec, req)
	if ok {
		t.Fatal("expected ok=false on bad id")
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if _, ok := body["detail"]; !ok {
		t.Fatalf("expected detail key in error body: %s", rec.Body)
	}
}

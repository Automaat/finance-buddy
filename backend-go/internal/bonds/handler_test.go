package bonds

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

func validCreate() createRequest {
	return createRequest{
		Type:          "COI",
		Series:        "COI0631",
		FaceValue:     100,
		PurchaseDate:  wire.IsoDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		FirstYearRate: 6.5,
		Margin:        1.5,
	}
}

func TestValidateCreateErrors(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*createRequest)
		wantField string
	}{
		{"invalid type", func(r *createRequest) { r.Type = "ZZZ" }, "type"},
		{"empty series", func(r *createRequest) { r.Series = "  " }, "series"},
		{"zero face value", func(r *createRequest) { r.FaceValue = 0 }, "face_value"},
		{"negative face value", func(r *createRequest) { r.FaceValue = -10 }, "face_value"},
		{"zero purchase date", func(r *createRequest) { r.PurchaseDate = wire.IsoDate(time.Time{}) }, "purchase_date"},
		{"rate out of range", func(r *createRequest) { r.FirstYearRate = 150 }, "first_year_rate"},
		{"negative margin", func(r *createRequest) { r.Margin = -1 }, "margin"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validCreate()
			tc.mutate(&req)
			vErr := validateCreate(&req)
			if vErr == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
			if vErr.Field != tc.wantField {
				t.Fatalf("field = %q, want %q", vErr.Field, tc.wantField)
			}
		})
	}
}

func TestValidateCreateNormalizesType(t *testing.T) {
	req := validCreate()
	req.Type = " coi "
	if vErr := validateCreate(&req); vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if req.Type != "COI" {
		t.Fatalf("type should be trimmed + upper-cased, got %q", req.Type)
	}
}

func TestBuildUpdatePatchErrors(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantField string
	}{
		{"invalid type", `{"type":"ZZZ"}`, "type"},
		{"empty series", `{"series":"  "}`, "series"},
		{"zero face value", `{"face_value":0}`, "face_value"},
		{"rate out of range", `{"first_year_rate":250}`, "first_year_rate"},
		{"non-integer owner", `{"owner_user_id":"x"}`, "owner_user_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var raw map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tc.body), &raw); err != nil {
				t.Fatalf("bad test JSON: %v", err)
			}
			_, vErr := buildUpdatePatch(raw)
			if vErr == nil || vErr.Field != tc.wantField {
				t.Fatalf("got %+v, want field %q", vErr, tc.wantField)
			}
		})
	}
}

func TestParseIDParamInvalidWrites422(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/bonds/abc", http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	if _, ok := parseIDParam(rec, req); ok {
		t.Fatal("expected ok=false on non-integer id")
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

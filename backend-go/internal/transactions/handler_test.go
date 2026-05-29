package transactions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// rawBody turns a JSON object literal into the map[string]json.RawMessage the
// validation builders consume.
func rawBody(t *testing.T, obj string) map[string]json.RawMessage {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(obj), &raw); err != nil {
		t.Fatalf("bad test JSON: %v", err)
	}
	return raw
}

func TestBuildCreateRequestValidationErrors(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantField string
	}{
		{"missing amount", `{"date":"2026-01-01","owner_user_id":1}`, "amount"},
		{"zero amount", `{"amount":0,"date":"2026-01-01","owner_user_id":1}`, "amount"},
		{"negative amount", `{"amount":-5,"date":"2026-01-01","owner_user_id":1}`, "amount"},
		{"missing date", `{"amount":10,"owner_user_id":1}`, "date"},
		{"bad date format", `{"amount":10,"date":"01-2026","owner_user_id":1}`, "date"},
		{"missing owner field", `{"amount":10,"date":"2026-01-01"}`, "owner_user_id"},
		{"non-integer owner", `{"amount":10,"date":"2026-01-01","owner_user_id":"x"}`, "owner_user_id"},
		{"invalid transaction_type", `{"amount":10,"date":"2026-01-01","owner_user_id":1,"transaction_type":"nope"}`, "transaction_type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, vErr := buildCreateRequest(rawBody(t, tc.body))
			if vErr == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
			if vErr.Field != tc.wantField {
				t.Fatalf("field = %q, want %q", vErr.Field, tc.wantField)
			}
		})
	}
}

func TestBuildCreateRequestValidWithNullOwner(t *testing.T) {
	// Explicit null owner ("Shared") is allowed and yields nil.
	req, vErr := buildCreateRequest(rawBody(t, `{"amount":123.45,"date":"2026-01-01","owner_user_id":null}`))
	if vErr != nil {
		t.Fatalf("unexpected validation error: %+v", vErr)
	}
	if req.OwnerUserID != nil {
		t.Fatalf("null owner should yield nil, got %v", *req.OwnerUserID)
	}
	if req.Amount.String() != "123.45" {
		t.Fatalf("amount = %s, want 123.45", req.Amount.String())
	}
}

func TestParseAccountIDInvalidWrites422(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/accounts/abc/transactions", http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("account_id", "abc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	if _, ok := parseAccountID(rec, req); ok {
		t.Fatal("expected ok=false on non-integer account_id")
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

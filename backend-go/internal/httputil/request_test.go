package httputil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestDecodeJSON(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(
			context.Background(), http.MethodPost, "/", strings.NewReader(`{"name":"x"}`),
		)
		var got struct {
			Name string `json:"name"`
		}
		if !DecodeJSON(rec, req, 1024, &got) {
			t.Fatal("DecodeJSON returned false")
		}
		if got.Name != "x" {
			t.Fatalf("Name = %q", got.Name)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(`{`))
		var got map[string]json.RawMessage
		if DecodeJSON(rec, req, 1024, &got) {
			t.Fatal("DecodeJSON returned true")
		}
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d", rec.Code)
		}
		assertValidationField(t, rec, "body")
	})
}

func TestPathInt(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/accounts/42", http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("account_id", "42")
	req = req.WithContext(contextWithRoute(req, rctx))

	got, ok := PathInt(httptest.NewRecorder(), req, "account_id")
	if !ok {
		t.Fatal("PathInt returned false")
	}
	if got != 42 {
		t.Fatalf("got = %d", got)
	}
}

func TestPathIntInvalid(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/accounts/x", http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("account_id", "x")
	req = req.WithContext(contextWithRoute(req, rctx))
	rec := httptest.NewRecorder()

	if _, ok := PathInt(rec, req, "account_id"); ok {
		t.Fatal("PathInt returned true")
	}
	assertValidationField(t, rec, "account_id")
}

func TestOptionalQueryInt(t *testing.T) {
	rec := httptest.NewRecorder()
	got, ok := OptionalQueryInt(rec, url.Values{"owner_user_id": []string{"7"}}, "owner_user_id")
	if !ok {
		t.Fatal("OptionalQueryInt returned false")
	}
	if got == nil || *got != 7 {
		t.Fatalf("got = %v", got)
	}
}

func TestOptionalQueryIntInvalid(t *testing.T) {
	rec := httptest.NewRecorder()
	if _, ok := OptionalQueryInt(rec, url.Values{"owner_user_id": []string{"x"}}, "owner_user_id"); ok {
		t.Fatal("OptionalQueryInt returned true")
	}
	assertValidationField(t, rec, "owner_user_id")
}

func TestOptionalQueryDate(t *testing.T) {
	rec := httptest.NewRecorder()
	got, ok := OptionalQueryDate(rec, url.Values{"date_from": []string{"2026-05-25"}}, "date_from")
	if !ok {
		t.Fatal("OptionalQueryDate returned false")
	}
	if got == nil || got.Format("2006-01-02") != "2026-05-25" {
		t.Fatalf("got = %v", got)
	}
}

func TestOptionalQueryDateInvalid(t *testing.T) {
	rec := httptest.NewRecorder()
	if _, ok := OptionalQueryDate(rec, url.Values{"date_from": []string{"25/05/2026"}}, "date_from"); ok {
		t.Fatal("OptionalQueryDate returned true")
	}
	assertValidationField(t, rec, "date_from")
}

func contextWithRoute(req *http.Request, rctx *chi.Context) context.Context {
	return context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
}

func assertValidationField(t *testing.T, rec *httptest.ResponseRecorder, field string) {
	t.Helper()
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rec.Code)
	}
	var env struct {
		Detail []struct {
			Loc []string `json:"loc"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Detail) != 1 {
		t.Fatalf("detail len = %d", len(env.Detail))
	}
	if len(env.Detail[0].Loc) != 2 || env.Detail[0].Loc[0] != "body" || env.Detail[0].Loc[1] != field {
		t.Fatalf("loc = %v", env.Detail[0].Loc)
	}
}

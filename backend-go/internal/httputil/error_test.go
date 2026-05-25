package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, map[string]int{"x": 1})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %s", got)
	}
	if got := rec.Body.String(); got != "{\"x\":1}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestWriteDetailError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteDetailError(rec, http.StatusNotFound, "not here")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["detail"] != "not here" {
		t.Fatalf("detail = %q", got["detail"])
	}
}

func TestWriteBodyValidationError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteBodyValidationError(rec, "name", "is required", "x")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rec.Code)
	}
	var env struct {
		Detail []struct {
			Type  string   `json:"type"`
			Msg   string   `json:"msg"`
			Input string   `json:"input"`
			Loc   []string `json:"loc"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Detail) != 1 {
		t.Fatalf("detail len = %d", len(env.Detail))
	}
	d := env.Detail[0]
	if d.Type != "value_error" || d.Msg != "is required" || d.Input != "x" {
		t.Fatalf("detail = %+v", d)
	}
	if len(d.Loc) != 2 || d.Loc[0] != "body" || d.Loc[1] != "name" {
		t.Fatalf("loc = %v", d.Loc)
	}
}

func TestWriteQueryValidationError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteQueryValidationError(rec, "year", "must be int")
	var env struct {
		Detail []map[string]any `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Detail) != 1 {
		t.Fatalf("detail len = %d", len(env.Detail))
	}
	d := env.Detail[0]
	if _, hasInput := d["input"]; hasInput {
		t.Fatalf("query variant must omit 'input', got %v", d)
	}
	loc, _ := d["loc"].([]any)
	if len(loc) != 2 || loc[0] != "query" || loc[1] != "year" {
		t.Fatalf("loc = %v", d["loc"])
	}
}

func TestWritePydanticError(t *testing.T) {
	rec := httptest.NewRecorder()
	WritePydanticError(rec, &ValidationError{Field: "amount", Msg: "must be positive"})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rec.Code)
	}
	var env struct {
		Detail []struct {
			Msg   string   `json:"msg"`
			Input string   `json:"input"`
			Loc   []string `json:"loc"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Detail[0].Msg != "must be positive" || env.Detail[0].Input != "" {
		t.Fatalf("detail = %+v", env.Detail[0])
	}
	if env.Detail[0].Loc[0] != "body" || env.Detail[0].Loc[1] != "amount" {
		t.Fatalf("loc = %v", env.Detail[0].Loc)
	}
}

package httputil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// DecodeJSON reads a size-limited JSON request body and writes the existing
// Pydantic-shaped validation envelope on decode failure.
func DecodeJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, dst any) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, maxBytes)).Decode(dst); err != nil {
		WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return false
	}
	return true
}

// PathInt parses an integer chi path parameter and writes the backend's
// existing validation envelope on parse failure.
func PathInt(w http.ResponseWriter, r *http.Request, param string) (int, bool) {
	return PathIntField(w, r, param, param)
}

// PathIntField parses an integer chi path parameter and writes a validation
// envelope under field. Use when the URL placeholder differs from the legacy
// Pydantic error field that clients/tests already expect.
func PathIntField(w http.ResponseWriter, r *http.Request, param, field string) (int, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.Atoi(raw)
	if err != nil {
		WriteBodyValidationError(w, field, "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// OptionalQueryInt parses an optional integer query parameter. It writes a body
// validation envelope, matching the legacy handlers' wire shape.
func OptionalQueryInt(w http.ResponseWriter, q url.Values, key string) (*int, bool) {
	v := q.Get(key)
	if v == "" {
		return nil, true
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		WriteBodyValidationError(w, key, "must be an integer", v)
		return nil, false
	}
	return &n, true
}

// OptionalQueryDate parses an optional YYYY-MM-DD query parameter. It writes a
// body validation envelope, matching the legacy handlers' wire shape.
func OptionalQueryDate(w http.ResponseWriter, q url.Values, key string) (*time.Time, bool) {
	v := q.Get(key)
	if v == "" {
		return nil, true
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		WriteBodyValidationError(w, key, "must be YYYY-MM-DD", v)
		return nil, false
	}
	return &t, true
}

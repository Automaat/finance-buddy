// Package httputil holds HTTP response helpers shared across handler
// packages. The error/validation shapes mirror what FastAPI/Pydantic v2
// returned, since the bb-test golden snapshots are pinned to that format.
package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ValidationError is a per-field validation failure produced by request
// parsing/validation code. Handlers convert it to a 422 Pydantic-style
// response via WritePydanticError.
type ValidationError struct {
	Field string
	Msg   string
}

// WriteJSON encodes payload as JSON with the given status. Encoder errors
// are logged but not surfaced — by the time we're encoding, headers have
// already been flushed.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode response", "err", err, "status", status)
	}
}

// WriteDetailError sends `{"detail": "..."}` at the given status — the
// FastAPI shape for non-validation errors.
func WriteDetailError(w http.ResponseWriter, status int, detail string) {
	WriteJSON(w, status, map[string]string{"detail": detail})
}

// WriteBodyValidationError sends a 422 Pydantic-shaped error keyed to a
// body field, including the offending raw input for client debugging.
func WriteBodyValidationError(w http.ResponseWriter, field, msg, input string) {
	WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"detail": []map[string]any{
			{
				"type":  "value_error",
				"loc":   []string{"body", field},
				"msg":   msg,
				"input": input,
			},
		},
	})
}

// WriteQueryValidationError sends a 422 Pydantic-shaped error keyed to a
// query parameter. Omits "input" because query handlers historically did
// not echo the raw value.
func WriteQueryValidationError(w http.ResponseWriter, field, msg string) {
	WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"detail": []map[string]any{
			{"type": "value_error", "loc": []string{"query", field}, "msg": msg},
		},
	})
}

// WritePydanticError adapts a ValidationError carrier into a 422 body
// validation response with empty "input" — matches the existing handler
// behavior when the offending value is not readily available.
func WritePydanticError(w http.ResponseWriter, vErr *ValidationError) {
	WriteBodyValidationError(w, vErr.Field, vErr.Msg, "")
}

package config

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON encodes the payload with Content-Type set. Encoding errors are
// logged but otherwise swallowed — by the time we're inside Encode, the
// response status and headers are already on the wire and there's nothing
// useful to send the client beyond a partial body.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode response", "err", err, "status", status)
	}
}

// writeDetailError matches FastAPI's HTTPException shape: {"detail": "..."}.
func writeDetailError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

// writeValidationError emits a single-entry Pydantic-shaped 422 body so the
// frontend's existing error handling keeps working during cutover.
func writeValidationError(w http.ResponseWriter, field, msg, input string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
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

// writePydanticError emits a 422 from a validationError captured during
// request validation.
func writePydanticError(w http.ResponseWriter, vErr *validationError) {
	writeValidationError(w, vErr.Field, vErr.Msg, "")
}

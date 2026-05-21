package config

import (
	"encoding/json"
	"net/http"
)

// writeJSON encodes the payload with Content-Type set. Drop encoding errors
// silently — there's nothing meaningful to do with them at the response
// boundary, and the client will see a partial body.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
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

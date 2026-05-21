package salaries

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type validationError struct {
	Field string
	Msg   string
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode response", "err", err, "status", status)
	}
}

func writeDetailError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

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

func writePydanticError(w http.ResponseWriter, vErr *validationError) {
	writeValidationError(w, vErr.Field, vErr.Msg, "")
}

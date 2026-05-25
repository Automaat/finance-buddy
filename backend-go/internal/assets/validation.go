package assets

import (
	"encoding/json"
	"strings"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

func requireName(raw map[string]json.RawMessage) (string, *httputil.ValidationError) {
	v, ok := raw["name"]
	if !ok || validation.IsNull(v) {
		return "", &httputil.ValidationError{Field: "name", Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &httputil.ValidationError{Field: "name", Msg: "must be a string"}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", &httputil.ValidationError{Field: "name", Msg: "Name cannot be empty"}
	}
	return s, nil
}

// optionalName: absent / explicit null -> no change. Empty string -> 422.
func optionalName(raw map[string]json.RawMessage) (*string, *httputil.ValidationError) {
	v, ok := raw["name"]
	if !ok || validation.IsNull(v) {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return nil, &httputil.ValidationError{Field: "name", Msg: "must be a string"}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, &httputil.ValidationError{Field: "name", Msg: "Name cannot be empty"}
	}
	return &s, nil
}

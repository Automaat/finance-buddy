package assets

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

func requireName(raw map[string]json.RawMessage) (string, *httputil.ValidationError) {
	v, ok := raw["name"]
	if !ok || isNull(v) {
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
	if !ok || isNull(v) {
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

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}

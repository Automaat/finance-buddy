package assets

import (
	"bytes"
	"encoding/json"
	"strings"
)

func requireName(raw map[string]json.RawMessage) (string, *validationError) {
	v, ok := raw["name"]
	if !ok || isNull(v) {
		return "", &validationError{Field: "name", Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: "name", Msg: "must be a string"}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", &validationError{Field: "name", Msg: "Name cannot be empty"}
	}
	return s, nil
}

// optionalName: absent / explicit null -> no change. Empty string -> 422.
func optionalName(raw map[string]json.RawMessage) (*string, *validationError) {
	v, ok := raw["name"]
	if !ok || isNull(v) {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return nil, &validationError{Field: "name", Msg: "must be a string"}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, &validationError{Field: "name", Msg: "Name cannot be empty"}
	}
	return &s, nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}

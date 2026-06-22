package scenarios

import (
	"encoding/json"
	"strings"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

// validKinds enumerates the simulation kinds the frontend can save under.
// Keeping this list closed means a typo in `kind` is a 422 instead of a
// silent bucket nobody loads from.
var validKinds = map[string]struct{}{
	"retirement":         {},
	"mortgage-vs-invest": {},
	"wibor":              {},
	"monte-carlo":        {},
}

const (
	maxNameLen     = 200
	maxInputsBytes = 64 * 1024
)

// validateCreate checks the create/update payload. First-error-wins matches
// the rest of the codebase's Pydantic-style error shape.
func validateCreate(name, kind string, inputs json.RawMessage) *httputil.ValidationError {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return &httputil.ValidationError{Field: "name", Msg: "Name must not be empty"}
	}
	if len(trimmed) > maxNameLen {
		return &httputil.ValidationError{Field: "name", Msg: "Name must be at most 200 characters"}
	}
	if kind == "" {
		return &httputil.ValidationError{Field: "kind", Msg: "Kind is required"}
	}
	if _, ok := validKinds[kind]; !ok {
		return &httputil.ValidationError{Field: "kind", Msg: "Kind must be one of: monte-carlo, mortgage-vs-invest, retirement, wibor"}
	}
	if len(inputs) == 0 || validation.IsNull(inputs) {
		return &httputil.ValidationError{Field: "inputs_json", Msg: "Inputs must be a JSON object"}
	}
	if len(inputs) > maxInputsBytes {
		return &httputil.ValidationError{Field: "inputs_json", Msg: "Inputs payload exceeds 64 KiB"}
	}
	// Require an object — not a bare array/scalar — so the consumer can keep
	// growing the shape without ambiguity at the top level.
	var probe any
	if err := json.Unmarshal(inputs, &probe); err != nil {
		return &httputil.ValidationError{Field: "inputs_json", Msg: "Inputs must be valid JSON"}
	}
	if _, ok := probe.(map[string]any); !ok {
		return &httputil.ValidationError{Field: "inputs_json", Msg: "Inputs must be a JSON object"}
	}
	return nil
}

// validateUpdate is the rename-and-replace-inputs path; kind stays whatever
// the existing row already holds.
func validateUpdate(name string, inputs json.RawMessage) *httputil.ValidationError {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return &httputil.ValidationError{Field: "name", Msg: "Name must not be empty"}
	}
	if len(trimmed) > maxNameLen {
		return &httputil.ValidationError{Field: "name", Msg: "Name must be at most 200 characters"}
	}
	if len(inputs) == 0 || validation.IsNull(inputs) {
		return &httputil.ValidationError{Field: "inputs_json", Msg: "Inputs must be a JSON object"}
	}
	if len(inputs) > maxInputsBytes {
		return &httputil.ValidationError{Field: "inputs_json", Msg: "Inputs payload exceeds 64 KiB"}
	}
	var probe any
	if err := json.Unmarshal(inputs, &probe); err != nil {
		return &httputil.ValidationError{Field: "inputs_json", Msg: "Inputs must be valid JSON"}
	}
	if _, ok := probe.(map[string]any); !ok {
		return &httputil.ValidationError{Field: "inputs_json", Msg: "Inputs must be a JSON object"}
	}
	return nil
}

// validateCloneName accepts an optional override name; empty falls back to
// the handler suffixing the source name in `Clone`. Length cap still applies.
func validateCloneName(name string) *httputil.ValidationError {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) > maxNameLen {
		return &httputil.ValidationError{Field: "name", Msg: "Name must be at most 200 characters"}
	}
	return nil
}

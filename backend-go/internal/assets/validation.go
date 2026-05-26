package assets

import (
	"encoding/json"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

func requireName(raw map[string]json.RawMessage) (string, *httputil.ValidationError) {
	return validation.RequiredTrimmedString(raw, "name", "Field required", "Name cannot be empty")
}

// optionalName: absent / explicit null -> no change. Empty string -> 422.
func optionalName(raw map[string]json.RawMessage) (*string, *httputil.ValidationError) {
	return validation.OptionalTrimmedString(raw, "name", "Name cannot be empty")
}

package snapshots

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

type createRequest struct {
	Date   time.Time
	Notes  *string
	Values []ValueInput
}

func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *httputil.ValidationError) {
	var r createRequest
	t, vErr := validation.RequiredDate(raw, "date")
	if vErr != nil {
		return r, vErr
	}
	r.Date = t

	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}

	vRaw, ok := raw["values"]
	if !ok || validation.IsNull(vRaw) {
		return r, &httputil.ValidationError{Field: "values", Msg: "Field required"}
	}
	values, vErr := parseValues(vRaw)
	if vErr != nil {
		return r, vErr
	}
	if len(values) == 0 {
		return r, &httputil.ValidationError{Field: "values", Msg: "Snapshot must contain at least one account value"}
	}
	r.Values = values
	return r, nil
}

func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *httputil.ValidationError) {
	var p UpdatePatch
	if v, ok := raw["date"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &httputil.ValidationError{Field: "date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return p, &httputil.ValidationError{Field: "date", Msg: "must be YYYY-MM-DD"}
		}
		p.Date = &t
	}
	if v, ok := raw["notes"]; ok {
		p.NotesSet = true
		if validation.IsNull(v) {
			p.Notes = nil
		} else {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return p, &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
			}
			p.Notes = &s
		}
	}
	if v, ok := raw["values"]; ok && !validation.IsNull(v) {
		values, vErr := parseValues(v)
		if vErr != nil {
			return p, vErr
		}
		if len(values) == 0 {
			return p, &httputil.ValidationError{Field: "values", Msg: "Snapshot must contain at least one value"}
		}
		p.ValuesSet = true
		p.Values = values
	}
	return p, nil
}

func parseValues(raw json.RawMessage) ([]ValueInput, *httputil.ValidationError) {
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, &httputil.ValidationError{Field: "values", Msg: "must be an array of objects"}
	}
	out := make([]ValueInput, 0, len(entries))
	for _, e := range entries {
		entry, vErr := parseValueEntry(e)
		if vErr != nil {
			return nil, vErr
		}
		out = append(out, entry)
	}
	return out, nil
}

func parseValueEntry(e map[string]json.RawMessage) (ValueInput, *httputil.ValidationError) {
	var v ValueInput

	assetID, vErr := optionalInt(e, "asset_id")
	if vErr != nil {
		return v, vErr
	}
	v.AssetID = assetID

	accountID, vErr := optionalInt(e, "account_id")
	if vErr != nil {
		return v, vErr
	}
	v.AccountID = accountID

	if v.AssetID == nil && v.AccountID == nil {
		return v, &httputil.ValidationError{Field: "values", Msg: "Either asset_id or account_id must be provided"}
	}
	if v.AssetID != nil && v.AccountID != nil {
		return v, &httputil.ValidationError{Field: "values", Msg: "Only one of asset_id or account_id can be provided"}
	}

	valRaw, ok := e["value"]
	if !ok || validation.IsNull(valRaw) {
		return v, &httputil.ValidationError{Field: "value", Msg: "Field required"}
	}
	d, err := validation.RawDecimal(valRaw)
	if err != nil {
		return v, &httputil.ValidationError{Field: "value", Msg: "must be a number"}
	}
	v.Value = d
	return v, nil
}

func optionalInt(raw map[string]json.RawMessage, key string) (*int, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return nil, nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: fmt.Sprintf("%s must be an integer", key)}
	}
	return &n, nil
}

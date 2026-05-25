// Package wire holds JSON wire types shared across HTTP handlers. They
// preserve the JSON shapes produced by the Python-era backend: ISO date,
// timezone-less ISO timestamp, and Python-style float (always renders a
// decimal point).
package wire

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	isoDateLayout  = "2006-01-02"
	isoNaiveLayout = "2006-01-02T15:04:05.999999"
)

// IsoDate marshals a time.Time as a date-only YYYY-MM-DD string.
type IsoDate time.Time

func (d IsoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format(isoDateLayout) + `"`), nil
}

func (d *IsoDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("expected YYYY-MM-DD string: %w", err)
	}
	t, err := time.Parse(isoDateLayout, s)
	if err != nil {
		return fmt.Errorf("expected YYYY-MM-DD: %w", err)
	}
	*d = IsoDate(t)
	return nil
}

// IsoNaive marshals a time.Time as an ISO-8601 timestamp without a zone
// suffix, matching the Python datetime.isoformat() shape used by the
// legacy API.
type IsoNaive time.Time

func (t IsoNaive) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format(isoNaiveLayout) + `"`), nil
}

// PyFloat marshals as a JSON number that always carries a decimal point,
// mirroring Python's json.dumps behavior for floats (1 → 1.0).
type PyFloat float64

func (f PyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}

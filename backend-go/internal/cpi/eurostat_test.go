package cpi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

const eurostatFixture = `{
  "version": "2.0",
  "value": {"0": 6.6, "1": 4.7, "2": 3.6, "3": null},
  "dimension": {
    "time": {
      "category": {
        "index": {
          "2023-11": 0,
          "2024-11": 1,
          "2025-11": 2,
          "2026-04": 3
        }
      }
    }
  }
}`

func TestParseEurostat(t *testing.T) {
	var body eurostatPayload
	if err := decodeJSONString(eurostatFixture, &body); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	out, err := parseEurostat(&body)
	if err != nil {
		t.Fatalf("parseEurostat: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("rows = %d, want 3 (null skipped)", len(out))
	}
	// Sorted ascending by (year, month).
	if out[0].Year != 2023 || out[0].Month != 11 {
		t.Errorf("row 0 = %v, want 2023-11", out[0])
	}
	// Eurostat reports 6.6 (delta YoY); we store 106.6 to match GUS form.
	if !out[0].YoY.Equal(decimal.RequireFromString("106.6")) {
		t.Errorf("row 0 YoY = %s, want 106.6", out[0].YoY)
	}
	if !out[1].YoY.Equal(decimal.RequireFromString("104.7")) {
		t.Errorf("row 1 YoY = %s, want 104.7", out[1].YoY)
	}
	if !out[2].YoY.Equal(decimal.RequireFromString("103.6")) {
		t.Errorf("row 2 YoY = %s, want 103.6", out[2].YoY)
	}
}

func TestParseEurostatMonth(t *testing.T) {
	cases := []struct {
		in        string
		y, m      int
		shouldErr bool
	}{
		{"2023-11", 2023, 11, false},
		{"2024-01", 2024, 1, false},
		{"2024-13", 0, 0, true},
		{"bad", 0, 0, true},
	}
	for _, c := range cases {
		y, m, err := parseEurostatMonth(c.in)
		if c.shouldErr {
			if err == nil {
				t.Errorf("%q: want error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected err %v", c.in, err)
			continue
		}
		if y != c.y || m != c.m {
			t.Errorf("%q = (%d,%d), want (%d,%d)", c.in, y, m, c.y, c.m)
		}
	}
}

// decodeJSONString is a tiny helper so the test fixture stays inline.
func decodeJSONString(s string, dst any) error {
	return json.NewDecoder(strings.NewReader(s)).Decode(dst)
}

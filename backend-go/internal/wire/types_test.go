package wire

import (
	"encoding/json"
	"testing"
	"time"
)

func TestIsoDateMarshal(t *testing.T) {
	d := IsoDate(time.Date(2025, 3, 14, 9, 30, 0, 0, time.UTC))
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `"2025-03-14"` {
		t.Fatalf("got %s", out)
	}
}

func TestIsoDateUnmarshal(t *testing.T) {
	var d IsoDate
	if err := json.Unmarshal([]byte(`"2025-03-14"`), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := time.Date(2025, 3, 14, 0, 0, 0, 0, time.UTC)
	if !time.Time(d).Equal(want) {
		t.Fatalf("got %v, want %v", time.Time(d), want)
	}
}

func TestIsoDateUnmarshalReject(t *testing.T) {
	var d IsoDate
	if err := json.Unmarshal([]byte(`"2025/03/14"`), &d); err == nil {
		t.Fatalf("expected error for bad format")
	}
}

func TestIsoNaiveMarshal(t *testing.T) {
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{"whole second", time.Date(2025, 3, 14, 9, 30, 15, 0, time.UTC), `"2025-03-14T09:30:15"`},
		{"micros", time.Date(2025, 3, 14, 9, 30, 15, 123456000, time.UTC), `"2025-03-14T09:30:15.123456"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := json.Marshal(IsoNaive(tc.in))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(out) != tc.want {
				t.Fatalf("got %s, want %s", out, tc.want)
			}
		})
	}
}

func TestPyFloatMarshal(t *testing.T) {
	cases := []struct {
		in   PyFloat
		want string
	}{
		{0, "0.0"},
		{1, "1.0"},
		{-3, "-3.0"},
		{1.5, "1.5"},
		{1234.5678, "1234.5678"},
		{0.1, "0.1"},
	}
	for _, tc := range cases {
		out, err := json.Marshal(tc.in)
		if err != nil {
			t.Fatalf("marshal %v: %v", tc.in, err)
		}
		if string(out) != tc.want {
			t.Fatalf("PyFloat(%v) -> %s, want %s", float64(tc.in), out, tc.want)
		}
	}
}

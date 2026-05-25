package simulations

import (
	"encoding/json"
	"testing"
)

func TestOwnerLabel_NilIsWspolne(t *testing.T) {
	if got := ownerLabel(map[int]string{}, nil); got != "Wspólne" {
		t.Errorf("nil id: want Wspólne, got %q", got)
	}
}

func TestOwnerLabel_KnownReturnsName(t *testing.T) {
	names := map[int]string{1: "Marcin", 2: "Ewa"}
	id := 1
	if got := ownerLabel(names, &id); got != "Marcin" {
		t.Errorf("known id: want Marcin, got %q", got)
	}
}

func TestOwnerLabel_UnknownReturnsDash(t *testing.T) {
	id := 99
	if got := ownerLabel(map[int]string{1: "Marcin"}, &id); got != "—" {
		t.Errorf("unknown id: want em-dash, got %q", got)
	}
}

func TestPyFloatMarshal(t *testing.T) {
	cases := []struct {
		in   pyFloat
		want string
	}{
		{0, "0.0"},
		{100, "100.0"},
		{1.5, "1.5"},
		{-2.25, "-2.25"},
	}
	for _, c := range cases {
		got, err := json.Marshal(c.in)
		if err != nil {
			t.Fatalf("marshal %v: %v", c.in, err)
		}
		if string(got) != c.want {
			t.Errorf("Marshal(%v): want %s, got %s", c.in, c.want, got)
		}
	}
}

func TestAccountSimToWire_PassesNullableSalaryAndReturn(t *testing.T) {
	salary := 5000.0
	rate := 0.07
	sim := AccountSimulation{
		AccountName: "Test",
		YearlyProjections: []YearlyProjection{
			{Year: 2025, Age: 40, MonthlySalary: &salary, ReturnRate: &rate},
			{Year: 2026, Age: 41}, // both nil
		},
	}
	out := accountSimToWire(&sim)
	rows, ok := out["yearly_projections"].([]map[string]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("expected 2 projections, got %+v", out["yearly_projections"])
	}
	if rows[0]["monthly_salary"] == nil {
		t.Errorf("year 0: monthly_salary set but came back nil")
	}
	if rows[1]["monthly_salary"] != nil {
		t.Errorf("year 1: monthly_salary nil but came back %+v", rows[1]["monthly_salary"])
	}
	if rows[0]["return_rate"] == nil {
		t.Errorf("year 0: return_rate set but came back nil")
	}
	if rows[1]["return_rate"] != nil {
		t.Errorf("year 1: return_rate nil but came back %+v", rows[1]["return_rate"])
	}
}

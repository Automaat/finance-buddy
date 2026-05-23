package retirement

import (
	"math"
	"testing"
)

func TestMarginalPITRate(t *testing.T) {
	cases := []struct {
		name        string
		annualGross float64
		want        float64
	}{
		{"below kwota wolna is zero", 25_000, 0},
		{"at kwota wolna boundary stays zero", 30_000, 0},
		{"just above kwota wolna is 12%", 30_001, 0.12},
		{"at threshold stays 12%", 120_000, 0.12},
		{"just above threshold is 32%", 120_001, 0.32},
		{"between threshold and 1M is 32%", 250_000, 0.32},
		{"at 1M solidarity boundary stays 32%", 1_000_000, 0.32},
		{"above 1M adds 4% solidarity", 1_500_000, 0.36},
		{"negative income treated as below kwota wolna", -100, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MarginalPITRate(tc.annualGross)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("MarginalPITRate(%v) = %v, want %v", tc.annualGross, got, tc.want)
			}
		})
	}
}

func TestEstimatePITSavings(t *testing.T) {
	cases := []struct {
		name         string
		contribution float64
		annualGross  float64
		want         float64
	}{
		{"low bracket: 12% of contribution", 10_000, 80_000, 1200},
		{"high bracket: 32% of contribution", 10_000, 150_000, 3200},
		{"with solidarity: 36% of contribution", 10_000, 1_500_000, 3600},
		{"zero contribution returns zero", 0, 150_000, 0},
		{"negative contribution returns zero", -5_000, 150_000, 0},
		{"zero gross returns zero", 10_000, 0, 0},
		{"income below kwota wolna returns zero savings", 5_000, 25_000, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimatePITSavings(tc.contribution, tc.annualGross)
			if math.Abs(got-tc.want) > 1e-6 {
				t.Errorf("EstimatePITSavings(%v, %v) = %v, want %v", tc.contribution, tc.annualGross, got, tc.want)
			}
		})
	}
}

package simulations

import (
	"math"
	"testing"
)

func TestSimulateWiborScenarios_BaselineMatchesAnnuityFormula(t *testing.T) {
	in := WiborScenarioInputs{
		RemainingPrincipal: 300000,
		BaseAnnualRate:     6.5,
		RemainingMonths:    240,
		BasePayment:        2237.40,
	}
	out := SimulateWiborScenarios(in, []float64{0})
	if len(out.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(out.Scenarios))
	}
	s := out.Scenarios[0]
	// 300000 @ 6.5% / 240 months -> ~2236.72
	if math.Abs(s.MonthlyPayment-2236.72) > 1.0 {
		t.Errorf("baseline payment = %f, expected ~2236.72", s.MonthlyPayment)
	}
	expectedInterest := s.MonthlyPayment*float64(in.RemainingMonths) - in.RemainingPrincipal
	if math.Abs(s.TotalInterest-round(expectedInterest, 2)) > 1.0 {
		t.Errorf("total interest = %f, expected %f", s.TotalInterest, expectedInterest)
	}
	if s.TermMonths != in.RemainingMonths {
		t.Errorf("term at base payment = %d, expected %d", s.TermMonths, in.RemainingMonths)
	}
}

func TestSimulateWiborScenarios_PaymentRisesWithRate(t *testing.T) {
	in := WiborScenarioInputs{
		RemainingPrincipal: 300000,
		BaseAnnualRate:     6.5,
		RemainingMonths:    240,
		BasePayment:        2237.40,
	}
	out := SimulateWiborScenarios(in, []float64{-1, 0, 1, 2, 3})
	if len(out.Scenarios) != 5 {
		t.Fatalf("expected 5 scenarios, got %d", len(out.Scenarios))
	}
	for i := 1; i < len(out.Scenarios); i++ {
		if out.Scenarios[i].MonthlyPayment <= out.Scenarios[i-1].MonthlyPayment {
			t.Errorf("payment should rise with rate, got %f then %f",
				out.Scenarios[i-1].MonthlyPayment, out.Scenarios[i].MonthlyPayment)
		}
	}
}

func TestSimulateWiborScenarios_FixedPaymentExtendsTerm(t *testing.T) {
	in := WiborScenarioInputs{
		RemainingPrincipal: 300000,
		BaseAnnualRate:     6.5,
		RemainingMonths:    240,
		BasePayment:        2237.40,
	}
	out := SimulateWiborScenarios(in, []float64{2})
	if len(out.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(out.Scenarios))
	}
	if out.Scenarios[0].TermMonths <= in.RemainingMonths {
		t.Errorf("term should extend at higher rate when payment fixed, got %d (orig %d)",
			out.Scenarios[0].TermMonths, in.RemainingMonths)
	}
}

func TestSimulateWiborScenarios_CapsTermWhenPaymentBelowInterest(t *testing.T) {
	in := WiborScenarioInputs{
		RemainingPrincipal: 1000000,
		BaseAnnualRate:     5,
		RemainingMonths:    240,
		BasePayment:        100, // way below monthly interest
	}
	out := SimulateWiborScenarios(in, []float64{5})
	if out.Scenarios[0].TermMonths != 1200 {
		t.Errorf("expected term capped to 1200, got %d", out.Scenarios[0].TermMonths)
	}
}

func TestSimulateWiborScenarios_ZeroBasePaymentKeepsOriginalTerm(t *testing.T) {
	in := WiborScenarioInputs{
		RemainingPrincipal: 300000,
		BaseAnnualRate:     6.5,
		RemainingMonths:    240,
	}
	out := SimulateWiborScenarios(in, []float64{0, 2})
	for _, s := range out.Scenarios {
		if s.TermMonths != in.RemainingMonths {
			t.Errorf("with no BasePayment, term should equal RemainingMonths, got %d", s.TermMonths)
		}
	}
}

func TestSimulateWiborScenarios_FloorsRateAtPositive(t *testing.T) {
	in := WiborScenarioInputs{
		RemainingPrincipal: 100000,
		BaseAnnualRate:     0.5,
		RemainingMonths:    120,
	}
	out := SimulateWiborScenarios(in, []float64{-5})
	if out.Scenarios[0].AnnualRate <= 0 {
		t.Errorf("rate should be floored above 0, got %f", out.Scenarios[0].AnnualRate)
	}
	if !out.Scenarios[0].RateFloored {
		t.Errorf("RateFloored should flag the clamp, got false")
	}
}

func TestSimulateWiborScenarios_BalanceConvergesToZero(t *testing.T) {
	in := WiborScenarioInputs{
		RemainingPrincipal: 100000,
		BaseAnnualRate:     5,
		RemainingMonths:    120,
	}
	out := SimulateWiborScenarios(in, []float64{0})
	balances := out.Scenarios[0].YearlyBalances
	if len(balances) == 0 {
		t.Fatalf("expected non-empty yearly balances")
	}
	final := balances[len(balances)-1]
	if final > 1.0 {
		t.Errorf("end-of-term balance should be ~0, got %f", final)
	}
	for i := 1; i < len(balances); i++ {
		if balances[i] > balances[i-1] {
			t.Errorf("balance should decrease, year %d > year %d (%f > %f)",
				i, i-1, balances[i], balances[i-1])
		}
	}
}

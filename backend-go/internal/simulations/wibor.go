package simulations

import "math"

// WiborScenarioInputs is the validated payload for /api/simulations/wibor.
//
// BaseAnnualRate is the current effective rate (WIBOR + bank margin) on the
// loan. Scenarios are applied as additive percentage-point shocks to this
// base; e.g. +2 pp means a 200-bps rise in the effective rate.
type WiborScenarioInputs struct {
	RemainingPrincipal float64
	BaseAnnualRate     float64
	RemainingMonths    int
	BasePayment        float64
}

// WiborScenarioRow is one rate shock's projection.
type WiborScenarioRow struct {
	DeltaPP        float64
	AnnualRate     float64
	MonthlyPayment float64
	TotalInterest  float64
	TermMonths     int
	// RateFloored is true when the requested shock would push the effective
	// rate at or below zero, and the simulator clamped it to a positive
	// floor (0.01%) so the math remains valid.
	RateFloored bool
	// YearlyBalances are end-of-year mortgage balances under the
	// rescheduled-payment scenario (bank recalcs payment to keep the
	// original term). Length == ceil(RemainingMonths/12).
	YearlyBalances []float64
}

// WiborScenarioResult bundles the base payment with each scenario.
type WiborScenarioResult struct {
	BasePayment float64
	Scenarios   []WiborScenarioRow
}

// DefaultWiborDeltas are the standard shocks per #371.
var DefaultWiborDeltas = []float64{-1, 0, 1, 2, 3}

// SimulateWiborScenarios computes, for each rate shock:
//   - the new monthly payment that fully amortizes the remaining principal over
//     the original term at the shocked rate;
//   - the cumulative interest paid over that term;
//   - the term required if the borrower keeps paying BasePayment at the new
//     rate (clamped to 1200 months to bound runaway scenarios where the
//     base payment can't cover monthly interest).
//
// Pure function — same inputs always produce same outputs.
func SimulateWiborScenarios(in WiborScenarioInputs, deltas []float64) WiborScenarioResult {
	rows := make([]WiborScenarioRow, 0, len(deltas))
	for _, d := range deltas {
		requested := in.BaseAnnualRate + d
		rate := requested
		floored := false
		if rate <= 0 {
			rate = 0.01
			floored = true
		}
		monthlyRate := rate / 100 / 12
		n := float64(in.RemainingMonths)
		var payment float64
		if monthlyRate == 0 {
			payment = in.RemainingPrincipal / n
		} else {
			factor := math.Pow(1+monthlyRate, n)
			payment = in.RemainingPrincipal * (monthlyRate * factor) / (factor - 1)
		}
		totalInterest := payment*n - in.RemainingPrincipal

		termMonths := in.RemainingMonths
		if in.BasePayment > 0 {
			termMonths = termAtFixedPayment(in.RemainingPrincipal, monthlyRate, in.BasePayment)
		}

		rows = append(rows, WiborScenarioRow{
			DeltaPP:        d,
			AnnualRate:     round(rate, 4),
			MonthlyPayment: round(payment, 2),
			TotalInterest:  round(totalInterest, 2),
			TermMonths:     termMonths,
			RateFloored:    floored,
			YearlyBalances: yearlyAmortization(in.RemainingPrincipal, monthlyRate, payment, in.RemainingMonths),
		})
	}
	return WiborScenarioResult{BasePayment: round(in.BasePayment, 2), Scenarios: rows}
}

// yearlyAmortization returns the end-of-year mortgage balance for each year
// of the loan under the rescheduled-payment scenario. Element [0] is the
// balance at the end of year 1.
func yearlyAmortization(principal, monthlyRate, payment float64, months int) []float64 {
	years := (months + 11) / 12
	out := make([]float64, 0, years)
	balance := principal
	for m := 1; m <= months; m++ {
		interest := balance * monthlyRate
		principalPaid := payment - interest
		balance -= principalPaid
		if balance < 0 {
			balance = 0
		}
		if m%12 == 0 || m == months {
			out = append(out, round(balance, 2))
		}
	}
	return out
}

// termAtFixedPayment returns the number of months needed to amortize
// `principal` at monthly rate `r` while paying `payment` each month. Returns
// the loan's original term cap (1200 months / 100 years) when payment doesn't
// cover monthly interest — the loan would never amortize.
func termAtFixedPayment(principal, monthlyRate, payment float64) int {
	const maxTermMonths = 1200
	if monthlyRate <= 0 {
		months := math.Ceil(principal / payment)
		if months > maxTermMonths {
			return maxTermMonths
		}
		return int(months)
	}
	monthlyInterestAtStart := principal * monthlyRate
	if payment <= monthlyInterestAtStart {
		return maxTermMonths
	}
	denom := 1 - monthlyRate*principal/payment
	if denom <= 0 {
		return maxTermMonths
	}
	months := -math.Log(denom) / math.Log(1+monthlyRate)
	if months > maxTermMonths {
		return maxTermMonths
	}
	return int(math.Ceil(months))
}

// Package simulations implements /api/simulations — mortgage-vs-invest
// comparison, retirement projection, and form prefill.
//
// All projection math is float64, matching Python's float-everywhere
// behavior including the round(x, 2) at each stored value.
package simulations

import (
	"math"
)

const belkaRate = 0.19

// MortgageInputs mirrors MortgageVsInvestInputs after validation.
type MortgageInputs struct {
	RemainingPrincipal   float64
	AnnualInterestRate   float64
	RemainingMonths      int
	TotalMonthlyBudget   float64
	ExpectedAnnualReturn float64
	InflationRate        float64
	EnableVariableRate   bool
}

// MortgageYearlyRow mirrors MortgageVsInvestYearlyRow.
type MortgageYearlyRow struct {
	Year                         int
	AnnualRate                   float64
	ScenarioAMortgageBalance     float64
	ScenarioARealMortgageBalance float64
	ScenarioACumulativeInterest  float64
	ScenarioAInvestmentBalance   float64
	ScenarioAAfterTaxPortfolio   float64
	ScenarioARealPortfolio       float64
	ScenarioAPaidOff             bool
	ScenarioBMortgageBalance     float64
	ScenarioBRealMortgageBalance float64
	ScenarioBInvestmentBalance   float64
	ScenarioBAfterTaxPortfolio   float64
	ScenarioBRealPortfolio       float64
	ScenarioBCumulativeInterest  float64
	NetAdvantageInvest           float64
}

// MortgageSummary mirrors MortgageVsInvestSummary.
type MortgageSummary struct {
	RegularMonthlyPayment    float64
	TotalInterestA           float64
	TotalInterestB           float64
	InterestSaved            float64
	FinalInvestmentPortfolio float64
	FinalPortfolioAReal      float64
	FinalPortfolioBReal      float64
	MonthsSaved              int
	WinningStrategy          string
	NetAdvantage             float64
	BreakEvenGrossReturn     float64
}

// MortgageResult bundles the projection + summary.
type MortgageResult struct {
	Yearly  []MortgageYearlyRow
	Summary MortgageSummary
}

// BudgetTooLowError signals the monthly budget can't even cover the regular
// mortgage payment. Handler maps it to 400 with the exact Python message.
type BudgetTooLowError struct {
	Budget  float64
	Payment float64
}

func (e *BudgetTooLowError) Error() string { return "budget below required payment" }

// SimulateMortgageVsInvest is the pure-functions port of
// backend/app/services/simulations/mortgage.simulate_mortgage_vs_invest.
func SimulateMortgageVsInvest(in MortgageInputs) (MortgageResult, error) {
	monthlyRate := in.AnnualInterestRate / 100 / 12
	monthlyInvestRate := in.ExpectedAnnualReturn * (1 - belkaRate) / 100 / 12
	n := in.RemainingMonths
	p := in.RemainingPrincipal

	regularPayment := p * (monthlyRate * math.Pow(1+monthlyRate, float64(n))) /
		(math.Pow(1+monthlyRate, float64(n)) - 1)

	if in.TotalMonthlyBudget < regularPayment {
		return MortgageResult{}, &BudgetTooLowError{
			Budget: in.TotalMonthlyBudget, Payment: regularPayment,
		}
	}

	balanceA := p
	cumulativeInterestA := 0.0
	investmentA := 0.0
	payoffMonthA := n

	balanceB := p
	investmentB := 0.0
	cumulativeInterestB := 0.0

	yearly := []MortgageYearlyRow{}
	currentAnnualRate := in.AnnualInterestRate

	for month := 1; month <= n; month++ {
		remainingMonths := n - month + 1

		if in.EnableVariableRate {
			currentAnnualRate = variableRate(in.AnnualInterestRate, month)
		}
		currentMonthlyRate := currentAnnualRate / 100 / 12

		// Scenario A
		if balanceA > 0 {
			interestA := balanceA * currentMonthlyRate
			cumulativeInterestA += interestA
			amountToClear := balanceA + interestA
			actualPaymentA := math.Min(in.TotalMonthlyBudget, amountToClear)
			surplusA := in.TotalMonthlyBudget - actualPaymentA
			balanceA = math.Max(0, balanceA-(actualPaymentA-interestA))
			if balanceA == 0 && payoffMonthA == n {
				payoffMonthA = month
			}
			if surplusA > 0 {
				investmentA = (investmentA + surplusA) * (1 + monthlyInvestRate)
			}
		} else {
			investmentA = (investmentA + in.TotalMonthlyBudget) * (1 + monthlyInvestRate)
		}

		// Scenario B
		interestB := balanceB * currentMonthlyRate
		cumulativeInterestB += interestB
		minPaymentB := balanceB *
			(currentMonthlyRate * math.Pow(1+currentMonthlyRate, float64(remainingMonths))) /
			(math.Pow(1+currentMonthlyRate, float64(remainingMonths)) - 1)
		principalPaymentB := minPaymentB - interestB
		balanceB = math.Max(0, balanceB-principalPaymentB)
		extraB := math.Max(0, in.TotalMonthlyBudget-minPaymentB)
		investmentB = (investmentB + extraB) * (1 + monthlyInvestRate)

		if month%12 == 0 {
			year := month / 12
			inflationFactor := math.Pow(1+in.InflationRate/100, float64(year))
			realA := investmentA / inflationFactor
			realB := investmentB / inflationFactor
			realBalanceA := balanceA / inflationFactor
			realBalanceB := balanceB / inflationFactor
			netAdvantage := (realB - realBalanceB) - (realA - realBalanceA)
			yearly = append(yearly, MortgageYearlyRow{
				Year:                         year,
				AnnualRate:                   round(currentAnnualRate, 4),
				ScenarioAMortgageBalance:     round(balanceA, 2),
				ScenarioARealMortgageBalance: round(realBalanceA, 2),
				ScenarioACumulativeInterest:  round(cumulativeInterestA, 2),
				ScenarioAInvestmentBalance:   round(investmentA, 2),
				ScenarioAAfterTaxPortfolio:   round(investmentA, 2),
				ScenarioARealPortfolio:       round(realA, 2),
				ScenarioAPaidOff:             balanceA == 0,
				ScenarioBMortgageBalance:     round(balanceB, 2),
				ScenarioBRealMortgageBalance: round(realBalanceB, 2),
				ScenarioBInvestmentBalance:   round(investmentB, 2),
				ScenarioBAfterTaxPortfolio:   round(investmentB, 2),
				ScenarioBRealPortfolio:       round(realB, 2),
				ScenarioBCumulativeInterest:  round(cumulativeInterestB, 2),
				NetAdvantageInvest:           round(netAdvantage, 2),
			})
		}
	}

	summary := mortgageSummary(in, regularPayment, finalState{
		cumulativeInterestA: cumulativeInterestA,
		cumulativeInterestB: cumulativeInterestB,
		investmentA:         investmentA,
		investmentB:         investmentB,
		payoffMonthA:        payoffMonthA,
	})
	return MortgageResult{Yearly: yearly, Summary: summary}, nil
}

// finalState bundles the running totals SimulateMortgageVsInvest holds when
// the month loop ends — passed to mortgageSummary to keep that function
// under the funlen budget.
type finalState struct {
	cumulativeInterestA float64
	cumulativeInterestB float64
	investmentA         float64
	investmentB         float64
	payoffMonthA        int
}

func mortgageSummary(in MortgageInputs, regularPayment float64, s finalState) MortgageSummary {
	interestSaved := s.cumulativeInterestB - s.cumulativeInterestA
	monthsSaved := in.RemainingMonths - s.payoffMonthA
	finalInflationFactor := math.Pow(1+in.InflationRate/100, float64(in.RemainingMonths)/12)
	finalRealA := s.investmentA / finalInflationFactor
	finalRealB := s.investmentB / finalInflationFactor
	netAdvantageFinal := finalRealB - finalRealA

	winningStrategy := "inwestycja"
	netAdvantage := netAdvantageFinal
	if netAdvantageFinal < 0 {
		winningStrategy = "nadpłata"
		netAdvantage = finalRealA - finalRealB
	}
	return MortgageSummary{
		RegularMonthlyPayment:    round(regularPayment, 2),
		TotalInterestA:           round(s.cumulativeInterestA, 2),
		TotalInterestB:           round(s.cumulativeInterestB, 2),
		InterestSaved:            round(interestSaved, 2),
		FinalInvestmentPortfolio: round(s.investmentB, 2),
		FinalPortfolioAReal:      round(finalRealA, 2),
		FinalPortfolioBReal:      round(finalRealB, 2),
		MonthsSaved:              monthsSaved,
		WinningStrategy:          winningStrategy,
		NetAdvantage:             round(netAdvantage, 2),
		BreakEvenGrossReturn:     round(in.AnnualInterestRate/(1-belkaRate), 4),
	}
}

// variableRate reproduces the cyclical rate model — calibrated to Polish
// rate history (1% COVID low, 8% 2022 peak), phase derived from the start
// rate so the trajectory begins descending.
func variableRate(startRate float64, month int) float64 {
	const longTermMean = 4.5
	const cycleAmplitude = 3.5
	const cyclePeriod = 10.0
	sinVal := (startRate - longTermMean) / cycleAmplitude
	sinVal = math.Max(-1, math.Min(1, sinVal))
	phase := math.Pi - math.Asin(sinVal)
	year := float64(month-1) / 12
	angle := 2*math.Pi*year/cyclePeriod + phase
	return math.Max(1, longTermMean+cycleAmplitude*math.Sin(angle))
}

func round(v float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(v*scale) / scale
}

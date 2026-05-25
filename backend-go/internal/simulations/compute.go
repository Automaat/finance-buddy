package simulations

import (
	"math"

	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

// ownerLabel resolves an owner_user_id to a display name for AccountName.
// A nil id (jointly owned) renders as "Wspólne".
func ownerLabel(names map[int]string, id *int) string {
	if id == nil {
		return "Wspólne"
	}
	if n, ok := names[*id]; ok {
		return n
	}
	return "—"
}

// buildSimulationResponse turns the raw inputs + computed account simulations
// into the wire-shape map the /api/simulations/retirement endpoint serves.
// Pure — no store access, no logging.
func buildSimulationResponse(in simulationInputs, sims []AccountSimulation) map[string]any {
	years := in.RetirementAge - in.CurrentAge
	totalFinal, totalContrib, totalReturns := 0.0, 0.0, 0.0
	totalTaxSavings, totalSubsidies := 0.0, 0.0
	for _, s := range sims {
		totalFinal += s.FinalBalance
		totalContrib += s.TotalContributions
		totalReturns += s.TotalReturns
		totalTaxSavings += s.TotalTaxSavings
		totalSubsidies += s.TotalSubsidies
	}
	monthlyIncome := totalFinal * safeWithdrawalRate / 12
	inflationFactor := math.Pow(1+in.InflationRate/100, float64(years))
	monthlyIncomeToday := monthlyIncome / inflationFactor

	simWire := make([]map[string]any, 0, len(sims))
	for i := range sims {
		simWire = append(simWire, accountSimToWire(&sims[i]))
	}
	return map[string]any{
		"inputs":      in.echo(),
		"simulations": simWire,
		"summary": map[string]any{
			"total_final_balance":            wire.PyFloat(totalFinal),
			"total_contributions":            wire.PyFloat(totalContrib),
			"total_returns":                  wire.PyFloat(totalReturns),
			"total_tax_savings":              wire.PyFloat(totalTaxSavings),
			"total_subsidies":                wire.PyFloat(totalSubsidies),
			"estimated_monthly_income":       wire.PyFloat(monthlyIncome),
			"estimated_monthly_income_today": wire.PyFloat(monthlyIncomeToday),
			"years_until_retirement":         years,
		},
	}
}

func accountSimToWire(s *AccountSimulation) map[string]any {
	projections := make([]map[string]any, 0, len(s.YearlyProjections))
	for i := range s.YearlyProjections {
		p := &s.YearlyProjections[i]
		row := map[string]any{
			"year":                     p.Year,
			"age":                      p.Age,
			"annual_contribution":      wire.PyFloat(p.AnnualContribution),
			"balance_end_of_year":      wire.PyFloat(p.BalanceEndOfYear),
			"cumulative_contributions": wire.PyFloat(p.CumulativeContributions),
			"cumulative_returns":       wire.PyFloat(p.CumulativeReturns),
			"annual_limit":             wire.PyFloat(p.AnnualLimit),
			"limit_utilized_pct":       wire.PyFloat(p.LimitUtilizedPct),
			"tax_savings":              wire.PyFloat(p.TaxSavings),
			"government_subsidies":     wire.PyFloat(p.GovernmentSubsidies),
			"monthly_salary":           nil,
			"return_rate":              nil,
		}
		if p.MonthlySalary != nil {
			row["monthly_salary"] = wire.PyFloat(*p.MonthlySalary)
		}
		if p.ReturnRate != nil {
			row["return_rate"] = wire.PyFloat(*p.ReturnRate)
		}
		projections = append(projections, row)
	}
	return map[string]any{
		"account_name":        s.AccountName,
		"starting_balance":    wire.PyFloat(s.StartingBalance),
		"total_contributions": wire.PyFloat(s.TotalContributions),
		"total_returns":       wire.PyFloat(s.TotalReturns),
		"total_tax_savings":   wire.PyFloat(s.TotalTaxSavings),
		"total_subsidies":     wire.PyFloat(s.TotalSubsidies),
		"final_balance":       wire.PyFloat(s.FinalBalance),
		"yearly_projections":  projections,
	}
}

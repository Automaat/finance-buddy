package simulations

import (
	"math"
	"strconv"
	"strings"
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
			"total_final_balance":            pyFloat(totalFinal),
			"total_contributions":            pyFloat(totalContrib),
			"total_returns":                  pyFloat(totalReturns),
			"total_tax_savings":              pyFloat(totalTaxSavings),
			"total_subsidies":                pyFloat(totalSubsidies),
			"estimated_monthly_income":       pyFloat(monthlyIncome),
			"estimated_monthly_income_today": pyFloat(monthlyIncomeToday),
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
			"annual_contribution":      pyFloat(p.AnnualContribution),
			"balance_end_of_year":      pyFloat(p.BalanceEndOfYear),
			"cumulative_contributions": pyFloat(p.CumulativeContributions),
			"cumulative_returns":       pyFloat(p.CumulativeReturns),
			"annual_limit":             pyFloat(p.AnnualLimit),
			"limit_utilized_pct":       pyFloat(p.LimitUtilizedPct),
			"tax_savings":              pyFloat(p.TaxSavings),
			"government_subsidies":     pyFloat(p.GovernmentSubsidies),
			"monthly_salary":           nil,
			"return_rate":              nil,
		}
		if p.MonthlySalary != nil {
			row["monthly_salary"] = pyFloat(*p.MonthlySalary)
		}
		if p.ReturnRate != nil {
			row["return_rate"] = pyFloat(*p.ReturnRate)
		}
		projections = append(projections, row)
	}
	return map[string]any{
		"account_name":        s.AccountName,
		"starting_balance":    pyFloat(s.StartingBalance),
		"total_contributions": pyFloat(s.TotalContributions),
		"total_returns":       pyFloat(s.TotalReturns),
		"total_tax_savings":   pyFloat(s.TotalTaxSavings),
		"total_subsidies":     pyFloat(s.TotalSubsidies),
		"final_balance":       pyFloat(s.FinalBalance),
		"yearly_projections":  projections,
	}
}

type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}

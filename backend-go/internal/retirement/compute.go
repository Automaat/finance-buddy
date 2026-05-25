package retirement

import (
	"math"

	"github.com/shopspring/decimal"
)

// computeYearlyStat folds the contribution totals against the configured
// limit into the wire-shape yearly stat. The IKZE PIT fields are left zero —
// callers add them after looking up the owner's salary.
//
// Untyped-remainder rule: when typed buckets sum to less than the total,
// the gap is attributed to the employee bucket (matches the legacy Python
// behavior the bb-tests pin).
//
// isWarning uses the unrounded percentage on purpose: 89.95% must NOT flip
// the badge even though it renders as 90.0 in percentage_used.
func computeYearlyStat(year int, wrapper string, ownerUserID *int, totals ContributionTotals, limit decimal.Decimal) yearlyStat {
	totalF, _ := totals.Total.Float64()
	employeeF, _ := totals.Employee.Float64()
	employerF, _ := totals.Employer.Float64()
	if totalF > employeeF+employerF {
		employeeF += totalF - (employeeF + employerF)
	}
	limitF, _ := limit.Float64()
	remaining := limitF - totalF
	if remaining < 0 {
		remaining = 0
	}
	percentage := 0.0
	if limitF > 0 {
		percentage = totalF / limitF * 100
	}
	lAmt := pyFloat(limitF)
	rem := pyFloat(remaining)
	pct := pyFloat(math.Round(percentage*10) / 10)
	return yearlyStat{
		Year:                year,
		AccountWrapper:      wrapper,
		OwnerUserID:         ownerUserID,
		LimitAmount:         &lAmt,
		TotalContributed:    pyFloat(totalF),
		EmployeeContributed: pyFloat(employeeF),
		EmployerContributed: pyFloat(employerF),
		Remaining:           &rem,
		PercentageUsed:      &pct,
		IsWarning:           percentage >= 90,
	}
}

// computePPKStat collapses contribution totals + latest snapshot value into
// the PPK wire stat. Same untyped-remainder rule as computeYearlyStat: any
// gap between Total and Employee+Employer+Government is attributed to the
// employee bucket.
func computePPKStat(ownerUserID *int, totals ContributionTotals, latest decimal.Decimal) ppkStat {
	totalF, _ := totals.Total.Float64()
	employeeF, _ := totals.Employee.Float64()
	employerF, _ := totals.Employer.Float64()
	governmentF, _ := totals.Government.Float64()
	if totalF > employeeF+employerF+governmentF {
		employeeF += totalF - (employeeF + employerF + governmentF)
	}
	totalValue, _ := latest.Float64()
	returns := totalValue - totalF
	roi := 0.0
	if totalF > 0 {
		roi = returns / totalF * 100
	}
	roiRounded := math.Round(roi*100) / 100
	return ppkStat{
		OwnerUserID:           ownerUserID,
		TotalValue:            pyFloat(totalValue),
		EmployeeContributed:   pyFloat(employeeF),
		EmployerContributed:   pyFloat(employerF),
		GovernmentContributed: pyFloat(governmentF),
		TotalContributed:      pyFloat(totalF),
		Returns:               pyFloat(returns),
		ROIPercentage:         pyFloat(roiRounded),
	}
}

// computePPKContributionAmounts returns (employee, employer) PPK contribution
// amounts: gross salary multiplied by each rate, divided by 100.
func computePPKContributionAmounts(gross, employeeRate, employerRate decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	hundred := decimal.NewFromInt(100)
	return gross.Mul(employeeRate).Div(hundred), gross.Mul(employerRate).Div(hundred)
}

// computePPKGenerateResponse assembles the wire response for a generated
// month of PPK contributions, including any welcome/annual subsidies that
// were actually inserted by the store.
func computePPKGenerateResponse(
	req generateRequest,
	gross, employeeAmt, employerAmt decimal.Decimal,
	subsidy SubsidyConfig,
	result PPKContributionResult,
) ppkGenerateResponse {
	grossF, _ := gross.Float64()
	empF, _ := employeeAmt.Float64()
	emprF, _ := employerAmt.Float64()
	govF := 0.0
	welcomeApplied := result.WelcomeID != nil
	annualApplied := result.AnnualID != nil
	if welcomeApplied {
		w, _ := subsidy.WelcomeAmount.Float64()
		govF += w
	}
	if annualApplied {
		a, _ := subsidy.AnnualAmount.Float64()
		govF += a
	}
	ids := []int{result.EmployeeID, result.EmployerID}
	if result.WelcomeID != nil {
		ids = append(ids, *result.WelcomeID)
	}
	if result.AnnualID != nil {
		ids = append(ids, *result.AnnualID)
	}
	return ppkGenerateResponse{
		OwnerUserID:         req.OwnerUserID,
		Month:               req.Month,
		Year:                req.Year,
		GrossSalary:         pyFloat(grossF),
		EmployeeAmount:      pyFloat(empF),
		EmployerAmount:      pyFloat(emprF),
		GovernmentAmount:    pyFloat(govF),
		WelcomeApplied:      welcomeApplied,
		AnnualApplied:       annualApplied,
		TotalAmount:         pyFloat(empF + emprF + govF),
		TransactionsCreated: ids,
	}
}

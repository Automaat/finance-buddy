package simulations

import "math"

// MIN_ANNUAL_CONTRIBUTION_2026 — PPK minimum for annual subsidy eligibility.
const ppkMinAnnualContribution2026 = 1009.26

// YearlyProjection mirrors the schema's YearlyProjection. MonthlySalary and
// ReturnRate are pointers so PPK rows emit them and others emit JSON null.
type YearlyProjection struct {
	Year                    int
	Age                     int
	AnnualContribution      float64
	BalanceEndOfYear        float64
	CumulativeContributions float64
	CumulativeReturns       float64
	AnnualLimit             float64
	LimitUtilizedPct        float64
	TaxSavings              float64
	GovernmentSubsidies     float64
	MonthlySalary           *float64
	ReturnRate              *float64
}

// AccountSimulation mirrors the schema's AccountSimulation.
type AccountSimulation struct {
	AccountName        string
	StartingBalance    float64
	TotalContributions float64
	TotalReturns       float64
	TotalTaxSavings    float64
	TotalSubsidies     float64
	FinalBalance       float64
	YearlyProjections  []YearlyProjection
}

// IkeIkzeParams is the per-account input for simulateAccount. BaseLimit is
// resolved by the caller via the retirement_limits table (DB read kept out
// of the pure math).
type IkeIkzeParams struct {
	Wrapper             string
	Owner               string
	StartingBalance     float64
	AutoFillLimit       bool
	MonthlyContribution float64
	TaxRate             float64
	BaseLimit           float64
}

// SimulateAccount ports simulate_account — IKE/IKZE year-by-year projection.
func SimulateAccount(p IkeIkzeParams, yearsToRetirement, currentAge, currentYear int, annualReturnRate, limitGrowthRate float64) AccountSimulation {
	projections := make([]YearlyProjection, 0, yearsToRetirement)
	balance := p.StartingBalance
	cumulativeContributions := 0.0
	cumulativeTaxSavings := 0.0
	cumulativeReturns := 0.0

	for yearOffset := range yearsToRetirement {
		year := currentYear + yearOffset + 1
		age := currentAge + yearOffset + 1
		limit := p.BaseLimit * math.Pow(1+limitGrowthRate/100, float64(yearOffset+1))

		contribution := p.MonthlyContribution * 12
		if p.AutoFillLimit {
			contribution = limit
		} else if contribution > limit {
			contribution = limit
		}

		taxSavings := 0.0
		if p.Wrapper == "IKZE" {
			taxSavings = contribution * (p.TaxRate / 100)
		}

		balance *= 1 + annualReturnRate/100
		balance += contribution
		cumulativeContributions += contribution
		cumulativeTaxSavings += taxSavings
		cumulativeReturns = balance - p.StartingBalance - cumulativeContributions

		limitUtilized := 0.0
		if limit > 0 {
			limitUtilized = contribution / limit * 100
		}
		projections = append(projections, YearlyProjection{
			Year: year, Age: age,
			AnnualContribution:      contribution,
			BalanceEndOfYear:        balance,
			CumulativeContributions: cumulativeContributions,
			CumulativeReturns:       cumulativeReturns,
			AnnualLimit:             limit,
			LimitUtilizedPct:        limitUtilized,
			TaxSavings:              taxSavings,
		})
	}

	return AccountSimulation{
		AccountName:        p.Wrapper + " (" + p.Owner + ")",
		StartingBalance:    p.StartingBalance,
		TotalContributions: cumulativeContributions,
		TotalReturns:       cumulativeReturns,
		TotalTaxSavings:    cumulativeTaxSavings,
		FinalBalance:       balance,
		YearlyProjections:  projections,
	}
}

// BrokerageParams is the per-account input for simulateBrokerageAccount.
type BrokerageParams struct {
	Owner               string
	StartingBalance     float64
	MonthlyContribution float64
}

// SimulateBrokerageAccount ports simulate_brokerage_account — taxable account
// with 19% capital-gains tax on positive annual returns.
func SimulateBrokerageAccount(p BrokerageParams, currentAge, retirementAge, currentYear int, annualReturnRate float64) AccountSimulation {
	const capitalGainsTaxRate = 19.0
	years := retirementAge - currentAge
	balance := p.StartingBalance
	projections := make([]YearlyProjection, 0, years)
	cumulativeContributions := 0.0
	cumulativeReturns := 0.0

	for yearOffset := range years {
		yearAge := currentAge + yearOffset + 1
		grossReturns := balance * (annualReturnRate / 100)
		netReturns := grossReturns
		if grossReturns > 0 {
			netReturns = grossReturns * (1 - capitalGainsTaxRate/100)
		}
		balance += netReturns
		cumulativeReturns += netReturns
		annualContribution := p.MonthlyContribution * 12
		balance += annualContribution
		cumulativeContributions += annualContribution

		projections = append(projections, YearlyProjection{
			Year: currentYear + yearOffset + 1, Age: yearAge,
			AnnualContribution:      annualContribution,
			BalanceEndOfYear:        balance,
			CumulativeContributions: cumulativeContributions,
			CumulativeReturns:       cumulativeReturns,
		})
	}

	return AccountSimulation{
		AccountName:        "Rachunek maklerski (" + p.Owner + ")",
		StartingBalance:    p.StartingBalance,
		TotalContributions: cumulativeContributions,
		TotalReturns:       cumulativeReturns,
		FinalBalance:       balance,
		YearlyProjections:  projections,
	}
}

// PPKParams is the per-account input for simulatePPKAccount.
type PPKParams struct {
	Owner                string
	StartingBalance      float64
	MonthlyGrossSalary   float64
	EmployeeRate         float64
	EmployerRate         float64
	SalaryBelowThreshold bool
	IncludeWelcomeBonus  bool
	IncludeAnnualSubsidy bool
}

// PPKReturnForAge ports get_ppk_return_for_age — lifecycle allocation.
func PPKReturnForAge(age int) float64 {
	switch {
	case age < 40:
		return 7.0
	case age < 50:
		return 6.0
	case age < 60:
		return 5.0
	default:
		return 4.0
	}
}

// SimulatePPKAccount ports simulate_ppk_account.
func SimulatePPKAccount(p PPKParams, currentAge, retirementAge int, salaryGrowth float64) AccountSimulation {
	years := retirementAge - currentAge
	balance := p.StartingBalance
	monthlySalary := p.MonthlyGrossSalary
	projections := make([]YearlyProjection, 0, years)
	totalContributions := 0.0
	totalSubsidies := 0.0
	welcomeBonusAdded := false
	monthsParticipated := 0

	for year := range years {
		yearAge := currentAge + year
		annualContrib := 0.0
		yearSubsidies := 0.0

		annualReturn := PPKReturnForAge(yearAge)
		netAnnualReturn := annualReturn - 0.6
		monthlyReturn := netAnnualReturn / 12 / 100

		for range 12 {
			contrib := monthlySalary * (p.EmployeeRate + p.EmployerRate) / 100
			balance += contrib
			annualContrib += contrib
			totalContributions += contrib
			monthsParticipated++
			balance *= 1 + monthlyReturn
		}

		if p.IncludeWelcomeBonus && !welcomeBonusAdded && monthsParticipated >= 3 {
			balance += 250
			yearSubsidies += 250
			welcomeBonusAdded = true
		}
		if p.IncludeAnnualSubsidy && p.SalaryBelowThreshold && annualContrib >= ppkMinAnnualContribution2026 {
			balance += 240
			yearSubsidies += 240
		}
		totalSubsidies += yearSubsidies

		salaryCopy := monthlySalary
		returnCopy := annualReturn
		projections = append(projections, YearlyProjection{
			Year: year, Age: yearAge,
			AnnualContribution:      annualContrib,
			BalanceEndOfYear:        balance,
			CumulativeContributions: totalContributions,
			CumulativeReturns:       balance - totalContributions - totalSubsidies,
			GovernmentSubsidies:     yearSubsidies,
			MonthlySalary:           &salaryCopy,
			ReturnRate:              &returnCopy,
		})

		monthlySalary *= 1 + salaryGrowth/100
	}

	return AccountSimulation{
		AccountName:        "PPK (" + p.Owner + ")",
		StartingBalance:    p.StartingBalance,
		TotalContributions: totalContributions,
		TotalReturns:       balance - totalContributions - totalSubsidies,
		TotalSubsidies:     totalSubsidies,
		FinalBalance:       balance,
		YearlyProjections:  projections,
	}
}

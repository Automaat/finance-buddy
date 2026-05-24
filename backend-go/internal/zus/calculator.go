// Package zus implements /api/zus — pension projection + prefill.
//
// calculator.go is a pure-functions port of
// backend/app/services/zus_calculator.py. Math intentionally uses float64
// throughout to match Python's behavior byte-for-byte (every round is the
// same round(x, 2)).
package zus

import (
	"math"

	"github.com/Automaat/finance-buddy/backend-go/internal/rules"
)

// ZUS contribution rates (% of gross salary).
const (
	contributionRateKonto         = 0.1222
	contributionRateSubkontoNoOFE = 0.0730
	contributionRateSubkontoOFE   = 0.0438
)

// ZUS-specific contribution-base constants. Centralized PIT and ZUS-cap
// values come from the rules package (#545); growth + lookahead horizons
// stay local since they're tuning constants, not legal thresholds.
const (
	capGrowthRate  = 0.05
	healthRate     = 0.09
	baseCapYear    = 2026
	monthsPerYear  = 12.0
	suwakLookahead = 10
)

var (
	cap30x2026    = rules.MustFloat64("zus_cap_30x_2026")
	pitRate       = rules.MustFloat64("pit_rate_first_2026")
	annualFreeThr = rules.MustFloat64("pit_free_amount_2026")
)

// lifeExpectancyMonths — GUS 2025 dalsze trwanie życia at retirement age.
var lifeExpectancyMonths = map[int]float64{
	55: 318.0, 56: 312.0, 57: 306.0, 58: 300.0, 59: 294.0,
	60: 266.4, 61: 260.4, 62: 254.4, 63: 248.4, 64: 234.6,
	65: 220.8, 66: 214.8, 67: 208.8, 68: 202.8, 69: 196.8,
	70: 190.8,
}

// Inputs mirrors the Pydantic ZusCalculatorInputs after validation.
// OwnerUserID is the owning household member; nil means jointly owned.
type Inputs struct {
	OwnerUserID               *int
	BirthYear                 int
	Gender                    string
	RetirementAge             int
	CurrentGrossMonthlySalary float64
	SalaryGrowthRate          float64
	InflationRate             float64
	ValorizationRateKonto     float64
	ValorizationRateSubkonto  float64
	HasOFE                    bool
	KapitalPoczatkowy         float64
	WorkStartYear             int
	SalaryHistory             map[int]float64
}

// YearlyProjection mirrors ZusYearlyProjection.
type YearlyProjection struct {
	Year                 int
	Age                  int
	AnnualGrossSalary    float64
	SalaryCapped         bool
	ContributionKonto    float64
	ContributionSubkonto float64
	KontoBalance         float64
	SubkontoBalance      float64
	TotalBalance         float64
}

// SensitivityScenario mirrors ZusSensitivityScenario.
type SensitivityScenario struct {
	Label                string
	ValorizationKonto    float64
	ValorizationSubkonto float64
	MonthlyPensionGross  float64
	MonthlyPensionNet    float64
	ReplacementRate      float64
}

// Result mirrors ZusCalculatorResponse minus the echoed inputs (handler
// re-attaches them).
type Result struct {
	YearlyProjections          []YearlyProjection
	LifeExpectancyMonths       float64
	KontoAtRetirement          float64
	SubkontoAtRetirement       float64
	KapitalPoczatkowyValorized float64
	TotalCapital               float64
	MonthlyPensionGross        float64
	MonthlyPensionNet          float64
	ReplacementRate            float64
	LastGrossSalary            float64
	Sensitivity                []SensitivityScenario
}

// Calculate runs the ZUS projection. Pure function — no I/O.
func Calculate(in Inputs, currentYear int) Result {
	retirementYear := in.BirthYear + in.RetirementAge
	subkontoRate := contributionRateSubkontoNoOFE
	if in.HasOFE {
		subkontoRate = contributionRateSubkontoOFE
	}

	state := projectionState{
		kapitalValorized: in.KapitalPoczatkowy,
		lastSalary:       in.CurrentGrossMonthlySalary * monthsPerYear,
	}
	projections := make([]YearlyProjection, 0, retirementYear-in.WorkStartYear)
	for year := in.WorkStartYear; year < retirementYear; year++ {
		p := state.step(in, subkontoRate, year, currentYear, retirementYear)
		projections = append(projections, p)
	}

	life := lifeExpectancy(in.RetirementAge)
	totalCapital := state.kapitalValorized + state.konto + state.subkonto
	monthlyGross := totalCapital / life
	monthlyNet := applyPensionTax(monthlyGross)
	replacementRate := 0.0
	if state.lastSalary > 0 {
		replacementRate = monthlyGross / (state.lastSalary / monthsPerYear) * 100
	}
	sensitivity := buildSensitivity(in, retirementYear, currentYear, life)

	return Result{
		YearlyProjections:          projections,
		LifeExpectancyMonths:       life,
		KontoAtRetirement:          round2(state.konto),
		SubkontoAtRetirement:       round2(state.subkonto),
		KapitalPoczatkowyValorized: round2(state.kapitalValorized),
		TotalCapital:               round2(totalCapital),
		MonthlyPensionGross:        round2(monthlyGross),
		MonthlyPensionNet:          round2(monthlyNet),
		ReplacementRate:            round2(replacementRate),
		LastGrossSalary:            round2(state.lastSalary / monthsPerYear),
		Sensitivity:                sensitivity,
	}
}

type projectionState struct {
	konto            float64
	subkonto         float64
	kapitalValorized float64
	lastSalary       float64
}

func (s *projectionState) step(
	in Inputs, subkontoRate float64,
	year, currentYear, retirementYear int,
) YearlyProjection {
	annualSalary := salaryForYear(in, year, currentYear)
	s.lastSalary = annualSalary

	yearsFromBase := max(0, year-baseCapYear)
	yearCap := cap30x2026 * math.Pow(1+capGrowthRate, float64(yearsFromBase))
	capped := math.Min(annualSalary, yearCap)
	salaryCapped := annualSalary > yearCap

	cKonto := capped * contributionRateKonto
	cSubkonto := capped * subkontoRate
	if retirementYear-year <= suwakLookahead {
		cKonto += cSubkonto
		cSubkonto = 0
	}

	s.konto *= 1 + in.ValorizationRateKonto/100
	s.subkonto *= 1 + in.ValorizationRateSubkonto/100
	s.kapitalValorized *= 1 + in.ValorizationRateKonto/100

	s.konto += cKonto
	s.subkonto += cSubkonto

	return YearlyProjection{
		Year:                 year,
		Age:                  year - in.BirthYear,
		AnnualGrossSalary:    round2(annualSalary),
		SalaryCapped:         salaryCapped,
		ContributionKonto:    round2(cKonto),
		ContributionSubkonto: round2(cSubkonto),
		KontoBalance:         round2(s.konto),
		SubkontoBalance:      round2(s.subkonto),
		TotalBalance:         round2(s.konto + s.subkonto + s.kapitalValorized),
	}
}

func salaryForYear(in Inputs, year, currentYear int) float64 {
	if v, ok := in.SalaryHistory[year]; ok {
		return v
	}
	if year <= currentYear {
		return in.CurrentGrossMonthlySalary * monthsPerYear
	}
	yearsAhead := float64(year - currentYear)
	return in.CurrentGrossMonthlySalary * monthsPerYear * math.Pow(1+in.SalaryGrowthRate/100, yearsAhead)
}

func buildSensitivity(in Inputs, retirementYear, currentYear int, life float64) []SensitivityScenario {
	deltas := []struct {
		label string
		delta float64
	}{
		{"Pesymistyczny", -2.0},
		{"Bazowy", 0.0},
		{"Optymistyczny", 2.0},
	}
	results := make([]SensitivityScenario, 0, len(deltas))
	for _, d := range deltas {
		valKonto := in.ValorizationRateKonto + d.delta
		valSubkonto := in.ValorizationRateSubkonto + d.delta
		total := quickCalculate(in, retirementYear, currentYear, valKonto, valSubkonto)
		gross := total / life
		net := applyPensionTax(gross)
		lastSalary := projectedLastSalary(in, retirementYear, currentYear)
		rr := 0.0
		if lastSalary > 0 {
			rr = gross / lastSalary * 100
		}
		results = append(results, SensitivityScenario{
			Label:                d.label,
			ValorizationKonto:    valKonto,
			ValorizationSubkonto: valSubkonto,
			MonthlyPensionGross:  round2(gross),
			MonthlyPensionNet:    round2(net),
			ReplacementRate:      round2(rr),
		})
	}
	return results
}

func quickCalculate(in Inputs, retirementYear, currentYear int, valKonto, valSubkonto float64) float64 {
	subkontoRate := contributionRateSubkontoNoOFE
	if in.HasOFE {
		subkontoRate = contributionRateSubkontoOFE
	}
	konto := 0.0
	subkonto := 0.0
	kapital := in.KapitalPoczatkowy
	for year := in.WorkStartYear; year < retirementYear; year++ {
		annualSalary := salaryForYear(in, year, currentYear)
		yearsFromBase := max(0, year-baseCapYear)
		yearCap := cap30x2026 * math.Pow(1+capGrowthRate, float64(yearsFromBase))
		capped := math.Min(annualSalary, yearCap)
		cKonto := capped * contributionRateKonto
		cSubkonto := capped * subkontoRate
		if retirementYear-year <= suwakLookahead {
			cKonto += cSubkonto
			cSubkonto = 0
		}
		konto *= 1 + valKonto/100
		subkonto *= 1 + valSubkonto/100
		kapital *= 1 + valKonto/100
		konto += cKonto
		subkonto += cSubkonto
	}
	return kapital + konto + subkonto
}

func projectedLastSalary(in Inputs, retirementYear, currentYear int) float64 {
	yearsAhead := retirementYear - 1 - currentYear
	if yearsAhead <= 0 {
		return in.CurrentGrossMonthlySalary
	}
	return in.CurrentGrossMonthlySalary * math.Pow(1+in.SalaryGrowthRate/100, float64(yearsAhead))
}

func lifeExpectancy(retirementAge int) float64 {
	if v, ok := lifeExpectancyMonths[retirementAge]; ok {
		return v
	}
	if retirementAge < 55 {
		return lifeExpectancyMonths[55]
	}
	if retirementAge > 70 {
		return lifeExpectancyMonths[70]
	}
	// Linear interpolation between the two nearest known ages.
	// Initialize lower below the floor and upper above the ceiling so any
	// matching key in the table will displace the sentinel.
	lower := -1
	upper := 1 << 30
	for k := range lifeExpectancyMonths {
		if k <= retirementAge && k > lower {
			lower = k
		}
		if k >= retirementAge && k < upper {
			upper = k
		}
	}
	if lower == upper {
		return lifeExpectancyMonths[lower]
	}
	frac := float64(retirementAge-lower) / float64(upper-lower)
	return lifeExpectancyMonths[lower] + frac*(lifeExpectancyMonths[upper]-lifeExpectancyMonths[lower])
}

// applyPensionTax mirrors Python's apply_pension_tax (12% PIT with 30k free
// threshold + 9% health insurance).
func applyPensionTax(monthlyGross float64) float64 {
	annualGross := monthlyGross * monthsPerYear
	annualHealth := annualGross * healthRate
	taxable := math.Max(0, annualGross-annualFreeThr)
	annualPIT := taxable * pitRate
	annualNet := annualGross - annualHealth - annualPIT
	return math.Max(0, annualNet/monthsPerYear)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

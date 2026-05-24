package retirement

import "github.com/Automaat/finance-buddy/backend-go/internal/rules"

// Polish PIT brackets (kwota wolna 30k, próg 120k, plus solidarity
// surcharge on annual income above PLN 1M). All values sourced from the
// centralized rules table (#545) — the wrapping vars exist so callers
// keep using local Go identifiers.
var (
	pitFreeAmount       = rules.MustFloat64("pit_free_amount_2026")
	pitThresholdAnnual  = rules.MustFloat64("pit_threshold_first_2026")
	pitLowRate          = rules.MustFloat64("pit_rate_first_2026")
	pitHighRate         = rules.MustFloat64("pit_rate_second_2026")
	solidarityThreshold = rules.MustFloat64("pit_solidarity_threshold_2026")
	solidarityRate      = rules.MustFloat64("pit_solidarity_rate_2026")
)

// MarginalPITRate returns the applicable marginal rate for the next zloty
// earned at a given annual gross income. Below the kwota wolna (PLN 30k)
// the rate is zero; above PLN 1M the 4% solidarity surcharge is added.
func MarginalPITRate(annualGross float64) float64 {
	if annualGross <= pitFreeAmount {
		return 0
	}
	rate := pitLowRate
	if annualGross > pitThresholdAnnual {
		rate = pitHighRate
	}
	if annualGross > solidarityThreshold {
		rate += solidarityRate
	}
	return rate
}

// EstimatePITSavings approximates the income-tax saved by deducting a
// year's IKZE contributions from taxable income. Uses the marginal rate
// at the *pre*-deduction income level — accurate when the contribution
// doesn't straddle a bracket boundary, slightly overstates savings when
// it does (the part of the deduction below the next bracket would
// actually save at the lower rate). Display estimate, not a filing.
func EstimatePITSavings(contribution, annualGross float64) float64 {
	if contribution <= 0 || annualGross <= 0 {
		return 0
	}
	rate := MarginalPITRate(annualGross)
	return contribution * rate
}

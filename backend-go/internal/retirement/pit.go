package retirement

// Polish PIT brackets (2024+ — kwota wolna 30k, próg 120k, plus solidarity
// surcharge on annual income above PLN 1M). Mirrors src/lib/utils/pl_tax.ts.
const (
	pitFreeAmount       = 30_000.0
	pitThresholdAnnual  = 120_000.0
	pitLowRate          = 0.12
	pitHighRate         = 0.32
	solidarityThreshold = 1_000_000.0
	solidarityRate      = 0.04
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

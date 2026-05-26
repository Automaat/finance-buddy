package simulations

// EffectiveTaxRate returns the share of each gross PLN withdrawn that
// goes to tax at the given age. Branches:
//   - Taxable brokerage: 19% Belka on the gain portion only.
//   - IKE: tax-free post-60, taxed like brokerage if withdrawn earlier.
//   - IKZE: 10% PIT post-65, taxed like brokerage if withdrawn earlier.
//   - ZUS: already taxed at source (treated as 0% here for the portfolio).
func (m MonteCarloAccountMix) EffectiveTaxRate(age int) float64 {
	gain := m.TaxableGainPct / 100
	taxable := m.TaxablePct / 100
	ike := m.IkePct / 100
	ikze := m.IkzePct / 100
	rate := taxable * gain * BelkaRate
	if age < IkeFreeAge {
		// IKE pre-60: early withdrawal taxed like brokerage gains.
		rate += ike * gain * BelkaRate
	}
	if age >= IkzeFreeAge {
		rate += ikze * IkzePostAgeRate
	} else {
		rate += ikze * gain * BelkaRate
	}
	return rate
}

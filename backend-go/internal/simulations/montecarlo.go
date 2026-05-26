package simulations

import (
	"math"
	"math/rand/v2"
	"sort"
)

// DeriveAssumptions returns the expected return and volatility implied by
// an allocation: a value-weighted mean for return, and the variance of a
// zero-correlation portfolio for volatility (a deliberate simplification
// that captures diversification without requiring a full correlation
// matrix).
func DeriveAssumptions(a MonteCarloAllocation) (float64, float64) {
	ws := a.StocksPct / 100
	wb := a.BondsPct / 100
	wc := a.CashPct / 100
	expectedReturn := ws*AssetReturnStocksPct + wb*AssetReturnBondsPct + wc*AssetReturnCashPct
	variance := ws*ws*AssetVolStocksPct*AssetVolStocksPct +
		wb*wb*AssetVolBondsPct*AssetVolBondsPct +
		wc*wc*AssetVolCashPct*AssetVolCashPct
	return expectedReturn, math.Sqrt(variance)
}

// RunMonteCarlo simulates `params.Paths` retirement paths. Returns drawn
// from N(ExpectedReturn, Volatility). Each year-end: grow, then add an
// AnnualContribution if year-end age <= RetirementAge (the retirement
// year still contributes), otherwise subtract AnnualWithdrawal. Balance
// is clamped at zero, so once a path busts it stays busted.
func RunMonteCarlo(rng *rand.Rand, p MonteCarloInputs) MonteCarloResult {
	paths := p.Paths
	if paths <= 0 {
		paths = 1000
	}
	years := p.LifeExpectancy - p.CurrentAge
	if years <= 0 {
		return MonteCarloResult{Paths: paths, Assumptions: emptyAssumptions(p)}
	}

	expectedReturn := p.ExpectedReturn
	volatility := p.Volatility
	source := "manual"
	var allocOut *MonteCarloAllocationOut
	if p.Allocation != nil {
		expectedReturn, volatility = DeriveAssumptions(*p.Allocation)
		source = "allocation"
		allocOut = &MonteCarloAllocationOut{
			StocksPct: p.Allocation.StocksPct,
			BondsPct:  p.Allocation.BondsPct,
			CashPct:   p.Allocation.CashPct,
		}
	}
	sim := simulatePaths(rng, p, paths, years, expectedReturn, volatility)
	bands := buildBands(sim, paths, years, p.CurrentAge)

	var taxSummary *MonteCarloTaxSummary
	var mixOut *MonteCarloAccountMixOut
	if p.AccountMix != nil {
		mixOut = &MonteCarloAccountMixOut{
			TaxablePct:     p.AccountMix.TaxablePct,
			IkePct:         p.AccountMix.IkePct,
			IkzePct:        p.AccountMix.IkzePct,
			ZusPct:         p.AccountMix.ZusPct,
			TaxableGainPct: p.AccountMix.TaxableGainPct,
		}
		taxSummary = &MonteCarloTaxSummary{
			GrossWithdrawalsTotal: sim.grossTotal,
			TaxTotal:              sim.taxTotal,
			EffectiveLifetimeRate: safeDiv(sim.taxTotal, sim.grossTotal),
		}
	}

	return MonteCarloResult{
		SuccessRate: float64(sim.survived) / float64(paths),
		Bands:       bands,
		Paths:       paths,
		Assumptions: MonteCarloAssumptions{
			ExpectedReturn:      expectedReturn,
			Volatility:          volatility,
			Source:              source,
			Allocation:          allocOut,
			InflationMean:       p.InflationMean,
			InflationVolatility: p.InflationVolatility,
			AccountMix:          mixOut,
		},
		Tax: taxSummary,
	}
}

// pathMatrix bundles per-path per-year trajectories so RunMonteCarlo can
// stay short and the loop body stays linear. netNominal/netReal are nil
// when no AccountMix was supplied; consumers must mirror gross in that
// case (saves an allocation + per-band sort on the common no-tax path).
type pathMatrix struct {
	nominal      [][]float64
	realBalance  [][]float64
	netNominal   [][]float64
	netReal      [][]float64
	spending     [][]float64
	spendingReal [][]float64
	survived     int
	grossTotal   float64
	taxTotal     float64
}

func simulatePaths(rng *rand.Rand, p MonteCarloInputs, paths, years int, expectedReturn, volatility float64) pathMatrix {
	mean := expectedReturn / 100.0
	sd := volatility / 100.0
	infMean := p.InflationMean / 100.0
	infSd := p.InflationVolatility / 100.0
	hasTax := p.AccountMix != nil
	m := pathMatrix{
		nominal:      make([][]float64, paths),
		realBalance:  make([][]float64, paths),
		spending:     make([][]float64, paths),
		spendingReal: make([][]float64, paths),
	}
	if hasTax {
		m.netNominal = make([][]float64, paths)
		m.netReal = make([][]float64, paths)
	}
	for i := range paths {
		balance := p.CurrentPortfolio
		nomRow := make([]float64, years)
		realRow := make([]float64, years)
		// Always allocate the row slices to keep nilaway happy; the per-year
		// math below is still gated on hasTax so the no-tax common case
		// avoids the EffectiveTaxRate call. Real savings come from the
		// buildBands gate that skips the extra sorts.
		netRow := make([]float64, years)
		netRealRow := make([]float64, years)
		spendRow := make([]float64, years)
		spendRealRow := make([]float64, years)
		cumInflation := 1.0
		for y := range years {
			r := rng.NormFloat64()*sd + mean
			inf := rng.NormFloat64()*infSd + infMean
			// Clamp the annual factor so a sampled deflation tail can't flip
			// cumInflation negative — that would invert real balances and let
			// retirees "spend" negative withdrawals (i.e. receive money).
			factor := 1 + inf
			if factor < 0.01 {
				factor = 0.01
			}
			cumInflation *= factor
			age := p.CurrentAge + y + 1
			balance *= 1 + r
			if balance < 0 {
				balance = 0
			}
			withdrawal := 0.0
			if age <= p.RetirementAge {
				balance += p.AnnualContribution
			} else {
				desiredNet := p.AnnualWithdrawal * cumInflation
				// Gross up the portfolio debit to cover the tax that this
				// year's withdrawal mix incurs. With no AccountMix, taxRate
				// is 0 and gross == desiredNet (back-compat).
				taxRate := 0.0
				if p.AccountMix != nil {
					taxRate = p.AccountMix.EffectiveTaxRate(age)
				}
				gross := desiredNet
				if taxRate > 0 && taxRate < 1 {
					gross = desiredNet / (1 - taxRate)
				}
				if gross > balance {
					// Path busts this year: retiree withdraws whatever the
					// portfolio still holds, pays the tax on that gross, and
					// consumes the remainder net.
					gross = balance
					withdrawal = gross * (1 - taxRate)
					balance = 0
				} else {
					withdrawal = desiredNet
					balance -= gross
				}
				m.grossTotal += gross
				m.taxTotal += gross - withdrawal
			}
			nomRow[y] = balance
			realRow[y] = safeDiv(balance, cumInflation)
			if hasTax {
				netBalance := balance * (1 - p.AccountMix.EffectiveTaxRate(age))
				netRow[y] = netBalance
				netRealRow[y] = safeDiv(netBalance, cumInflation)
			}
			spendRow[y] = withdrawal
			spendRealRow[y] = safeDiv(withdrawal, cumInflation)
		}
		m.nominal[i] = nomRow
		m.realBalance[i] = realRow
		if hasTax {
			m.netNominal[i] = netRow
			m.netReal[i] = netRealRow
		}
		m.spending[i] = spendRow
		m.spendingReal[i] = spendRealRow
		if balance > 0 {
			m.survived++
		}
	}
	return m
}

func buildBands(m pathMatrix, paths, years, currentAge int) []MonteCarloPercentileBand {
	hasTax := m.netNominal != nil
	bands := make([]MonteCarloPercentileBand, years)
	col := make([]float64, paths)
	colReal := make([]float64, paths)
	colNet := make([]float64, paths)
	colNetReal := make([]float64, paths)
	for y := range years {
		for i := range paths {
			col[i] = m.nominal[i][y]
			colReal[i] = m.realBalance[i][y]
			if hasTax {
				colNet[i] = m.netNominal[i][y]
				colNetReal[i] = m.netReal[i][y]
			}
		}
		sort.Float64s(col)
		sort.Float64s(colReal)
		b := MonteCarloPercentileBand{
			Age:          currentAge + y + 1,
			P5:           percentile(col, 5),
			P50:          percentile(col, 50),
			P95:          percentile(col, 95),
			P5Real:       percentile(colReal, 5),
			P50Real:      percentile(colReal, 50),
			P95Real:      percentile(colReal, 95),
			Spending:     meanColumn(m.spending, y),
			SpendingReal: meanColumn(m.spendingReal, y),
		}
		if hasTax {
			sort.Float64s(colNet)
			sort.Float64s(colNetReal)
			b.P5Net = percentile(colNet, 5)
			b.P50Net = percentile(colNet, 50)
			b.P95Net = percentile(colNet, 95)
			b.P5NetReal = percentile(colNetReal, 5)
			b.P50NetReal = percentile(colNetReal, 50)
			b.P95NetReal = percentile(colNetReal, 95)
		} else {
			b.P5Net = b.P5
			b.P50Net = b.P50
			b.P95Net = b.P95
			b.P5NetReal = b.P5Real
			b.P50NetReal = b.P50Real
			b.P95NetReal = b.P95Real
		}
		bands[y] = b
	}
	return bands
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func meanColumn(m [][]float64, y int) float64 {
	if len(m) == 0 {
		return 0
	}
	sum := 0.0
	for i := range m {
		sum += m[i][y]
	}
	return sum / float64(len(m))
}

// emptyAssumptions echoes the chosen ER/vol pair when years <= 0 so the
// response shape stays consistent with the happy path.
func emptyAssumptions(p MonteCarloInputs) MonteCarloAssumptions {
	a := MonteCarloAssumptions{
		ExpectedReturn:      p.ExpectedReturn,
		Volatility:          p.Volatility,
		Source:              "manual",
		InflationMean:       p.InflationMean,
		InflationVolatility: p.InflationVolatility,
	}
	if p.Allocation != nil {
		r, v := DeriveAssumptions(*p.Allocation)
		a.ExpectedReturn = r
		a.Volatility = v
		a.Source = "allocation"
		a.Allocation = &MonteCarloAllocationOut{
			StocksPct: p.Allocation.StocksPct,
			BondsPct:  p.Allocation.BondsPct,
			CashPct:   p.Allocation.CashPct,
		}
	}
	if p.AccountMix != nil {
		a.AccountMix = &MonteCarloAccountMixOut{
			TaxablePct:     p.AccountMix.TaxablePct,
			IkePct:         p.AccountMix.IkePct,
			IkzePct:        p.AccountMix.IkzePct,
			ZusPct:         p.AccountMix.ZusPct,
			TaxableGainPct: p.AccountMix.TaxableGainPct,
		}
	}
	return a
}

// percentile picks pct (0..100) using linear interpolation, matching
// NumPy's default `linear` method on a sorted slice. Pure for tests.
func percentile(sorted []float64, pct float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	rank := pct / 100.0 * float64(n-1)
	lo := int(rank)
	hi := lo + 1
	if hi >= n {
		return sorted[n-1]
	}
	frac := rank - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}

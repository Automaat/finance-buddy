package simulations

import (
	"math"
	"math/rand/v2"
	"sort"
)

// Per-asset long-term nominal assumptions (annual, percent). These are
// the canonical reference numbers used when the user supplies a portfolio
// allocation instead of an explicit expected_return/volatility pair.
const (
	AssetReturnStocksPct = 7.0
	AssetReturnBondsPct  = 3.5
	AssetReturnCashPct   = 2.0
	AssetVolStocksPct    = 18.0
	AssetVolBondsPct     = 6.0
	AssetVolCashPct      = 1.0
)

// MonteCarloAllocation captures the stocks/bonds/cash split (percent,
// must sum to 100). When set on MonteCarloInputs, the simulation derives
// ExpectedReturn and Volatility from the per-asset constants above.
type MonteCarloAllocation struct {
	StocksPct float64
	BondsPct  float64
	CashPct   float64
}

// MonteCarloInputs is the validated user payload for a retirement Monte
// Carlo run. Volatility is the std-dev of annual returns (percent).
// Allocation, when non-nil, overrides ExpectedReturn/Volatility.
//
// AnnualWithdrawal is the user's spending in today's PLN; the simulation
// inflates it by the realized inflation path each year so the user keeps
// the same real lifestyle. With InflationMean=0 and InflationVolatility=0
// this collapses to a constant nominal withdrawal and real == nominal.
type MonteCarloInputs struct {
	CurrentPortfolio    float64
	AnnualContribution  float64
	ExpectedReturn      float64 // percent
	Volatility          float64 // percent (standard deviation)
	CurrentAge          int
	RetirementAge       int
	LifeExpectancy      int
	AnnualWithdrawal    float64 // today's PLN; inflated each year
	Paths               int     // defaults to 1000
	Allocation          *MonteCarloAllocation
	InflationMean       float64 // percent, annual
	InflationVolatility float64 // percent, std-dev of annual inflation
}

// MonteCarloPercentileBand is one age slice of the fan chart: p5/p50/p95
// of end-of-year balance across all paths, both in nominal PLN and in
// real (today's PLN) terms after dividing by the realized cumulative
// inflation factor on each path.
type MonteCarloPercentileBand struct {
	Age     int     `json:"age"`
	P5      float64 `json:"p5"`
	P50     float64 `json:"p50"`
	P95     float64 `json:"p95"`
	P5Real  float64 `json:"p5_real"`
	P50Real float64 `json:"p50_real"`
	P95Real float64 `json:"p95_real"`
	// Spending is the average realized nominal withdrawal at this age
	// (zero in the accumulation phase), and SpendingReal is that same
	// value deflated by the path's cumulative inflation factor — both
	// averaged across paths so the chart can show one curve per series.
	Spending     float64 `json:"spending"`
	SpendingReal float64 `json:"spending_real"`
}

// MonteCarloAssumptions echoes the return/volatility pair actually used
// by the simulation. When inputs carried an Allocation, Source is
// "allocation" and the percentages used to derive the numbers are
// included; otherwise Source is "manual" and Allocation is nil.
// InflationMean / InflationVolatility echo the inflation path parameters.
type MonteCarloAssumptions struct {
	ExpectedReturn      float64                  `json:"expected_return"`
	Volatility          float64                  `json:"volatility"`
	Source              string                   `json:"source"`
	Allocation          *MonteCarloAllocationOut `json:"allocation,omitempty"`
	InflationMean       float64                  `json:"inflation_mean"`
	InflationVolatility float64                  `json:"inflation_volatility"`
}

// MonteCarloAllocationOut is the wire echo of the allocation split.
type MonteCarloAllocationOut struct {
	StocksPct float64 `json:"stocks_pct"`
	BondsPct  float64 `json:"bonds_pct"`
	CashPct   float64 `json:"cash_pct"`
}

// MonteCarloResult is what the handler ships back. SuccessRate is the
// share of paths with non-negative ending balance at life_expectancy.
type MonteCarloResult struct {
	SuccessRate float64                    `json:"success_rate"`
	Bands       []MonteCarloPercentileBand `json:"bands"`
	Paths       int                        `json:"paths"`
	Assumptions MonteCarloAssumptions      `json:"assumptions"`
}

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
		},
	}
}

// pathMatrix bundles per-path per-year trajectories so RunMonteCarlo can
// stay short and the loop body stays linear.
type pathMatrix struct {
	nominal      [][]float64
	realBalance  [][]float64
	spending     [][]float64
	spendingReal [][]float64
	survived     int
}

func simulatePaths(rng *rand.Rand, p MonteCarloInputs, paths, years int, expectedReturn, volatility float64) pathMatrix {
	mean := expectedReturn / 100.0
	sd := volatility / 100.0
	infMean := p.InflationMean / 100.0
	infSd := p.InflationVolatility / 100.0
	m := pathMatrix{
		nominal:      make([][]float64, paths),
		realBalance:  make([][]float64, paths),
		spending:     make([][]float64, paths),
		spendingReal: make([][]float64, paths),
	}
	for i := range paths {
		balance := p.CurrentPortfolio
		nomRow := make([]float64, years)
		realRow := make([]float64, years)
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
				desired := p.AnnualWithdrawal * cumInflation
				if desired > balance {
					// Path busts this year: retiree withdraws whatever the
					// portfolio still holds, not the desired nominal amount.
					withdrawal = balance
					balance = 0
				} else {
					withdrawal = desired
					balance -= desired
				}
			}
			nomRow[y] = balance
			realRow[y] = safeDiv(balance, cumInflation)
			spendRow[y] = withdrawal
			spendRealRow[y] = safeDiv(withdrawal, cumInflation)
		}
		m.nominal[i] = nomRow
		m.realBalance[i] = realRow
		m.spending[i] = spendRow
		m.spendingReal[i] = spendRealRow
		if balance > 0 {
			m.survived++
		}
	}
	return m
}

func buildBands(m pathMatrix, paths, years, currentAge int) []MonteCarloPercentileBand {
	bands := make([]MonteCarloPercentileBand, years)
	col := make([]float64, paths)
	colReal := make([]float64, paths)
	for y := range years {
		for i := range paths {
			col[i] = m.nominal[i][y]
			colReal[i] = m.realBalance[i][y]
		}
		sort.Float64s(col)
		sort.Float64s(colReal)
		bands[y] = MonteCarloPercentileBand{
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

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
type MonteCarloInputs struct {
	CurrentPortfolio   float64
	AnnualContribution float64
	ExpectedReturn     float64 // percent
	Volatility         float64 // percent (standard deviation)
	CurrentAge         int
	RetirementAge      int
	LifeExpectancy     int
	AnnualWithdrawal   float64
	Paths              int // defaults to 1000
	Allocation         *MonteCarloAllocation
}

// MonteCarloPercentileBand is one age slice of the fan chart: p5/p50/p95
// of end-of-year balance across all paths.
type MonteCarloPercentileBand struct {
	Age int     `json:"age"`
	P5  float64 `json:"p5"`
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
}

// MonteCarloAssumptions echoes the return/volatility pair actually used
// by the simulation. When inputs carried an Allocation, Source is
// "allocation" and the percentages used to derive the numbers are
// included; otherwise Source is "manual" and Allocation is nil.
type MonteCarloAssumptions struct {
	ExpectedReturn float64                  `json:"expected_return"`
	Volatility     float64                  `json:"volatility"`
	Source         string                   `json:"source"`
	Allocation     *MonteCarloAllocationOut `json:"allocation,omitempty"`
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
	mean := expectedReturn / 100.0
	sd := volatility / 100.0

	// trajectories[pathIdx][yearIdx] = balance at end of year.
	trajectories := make([][]float64, paths)
	survived := 0
	for i := range paths {
		balance := p.CurrentPortfolio
		path := make([]float64, years)
		for y := range years {
			r := rng.NormFloat64()*sd + mean
			age := p.CurrentAge + y + 1
			balance *= 1 + r
			if age <= p.RetirementAge {
				balance += p.AnnualContribution
			} else {
				balance -= p.AnnualWithdrawal
			}
			if balance < 0 {
				balance = 0
			}
			path[y] = balance
		}
		trajectories[i] = path
		if balance > 0 {
			survived++
		}
	}

	bands := make([]MonteCarloPercentileBand, years)
	col := make([]float64, paths)
	for y := range years {
		for i := range paths {
			col[i] = trajectories[i][y]
		}
		sort.Float64s(col)
		bands[y] = MonteCarloPercentileBand{
			Age: p.CurrentAge + y + 1,
			P5:  percentile(col, 5),
			P50: percentile(col, 50),
			P95: percentile(col, 95),
		}
	}

	return MonteCarloResult{
		SuccessRate: float64(survived) / float64(paths),
		Bands:       bands,
		Paths:       paths,
		Assumptions: MonteCarloAssumptions{
			ExpectedReturn: expectedReturn,
			Volatility:     volatility,
			Source:         source,
			Allocation:     allocOut,
		},
	}
}

// emptyAssumptions echoes the chosen ER/vol pair when years <= 0 so the
// response shape stays consistent with the happy path.
func emptyAssumptions(p MonteCarloInputs) MonteCarloAssumptions {
	if p.Allocation == nil {
		return MonteCarloAssumptions{
			ExpectedReturn: p.ExpectedReturn,
			Volatility:     p.Volatility,
			Source:         "manual",
		}
	}
	r, v := DeriveAssumptions(*p.Allocation)
	return MonteCarloAssumptions{
		ExpectedReturn: r,
		Volatility:     v,
		Source:         "allocation",
		Allocation: &MonteCarloAllocationOut{
			StocksPct: p.Allocation.StocksPct,
			BondsPct:  p.Allocation.BondsPct,
			CashPct:   p.Allocation.CashPct,
		},
	}
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

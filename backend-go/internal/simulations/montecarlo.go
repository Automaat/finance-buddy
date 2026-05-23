package simulations

import (
	"math/rand/v2"
	"sort"
)

// MonteCarloInputs is the validated user payload for a retirement Monte
// Carlo run. Volatility is the std-dev of annual returns (percent).
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
}

// MonteCarloPercentileBand is one age slice of the fan chart: p5/p50/p95
// of end-of-year balance across all paths.
type MonteCarloPercentileBand struct {
	Age int     `json:"age"`
	P5  float64 `json:"p5"`
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
}

// MonteCarloResult is what the handler ships back. SuccessRate is the
// share of paths with non-negative ending balance at life_expectancy.
type MonteCarloResult struct {
	SuccessRate float64                    `json:"success_rate"`
	Bands       []MonteCarloPercentileBand `json:"bands"`
	Paths       int                        `json:"paths"`
}

// RunMonteCarlo simulates `params.Paths` retirement paths. Returns drawn
// from N(ExpectedReturn, Volatility). During accumulation (age <
// RetirementAge) contributions are added each year; during decumulation
// withdrawals are subtracted (balance is clamped at 0).
func RunMonteCarlo(rng *rand.Rand, p MonteCarloInputs) MonteCarloResult {
	paths := p.Paths
	if paths <= 0 {
		paths = 1000
	}
	years := p.LifeExpectancy - p.CurrentAge
	if years <= 0 {
		return MonteCarloResult{Paths: paths}
	}

	mean := p.ExpectedReturn / 100.0
	sd := p.Volatility / 100.0

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

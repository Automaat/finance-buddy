package simulations

import (
	"math"
	"math/rand/v2"
	"testing"
)

func newSeededRng() *rand.Rand {
	// Deterministic seed for reproducible tests.
	return rand.New(rand.NewPCG(42, 99))
}

func TestRunMonteCarlo_DefaultPathCount(t *testing.T) {
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio: 100000,
		CurrentAge:       30,
		RetirementAge:    65,
		LifeExpectancy:   90,
		ExpectedReturn:   6,
		Volatility:       15,
		AnnualWithdrawal: 30000,
	})
	if got.Paths != 1000 {
		t.Fatalf("default paths = %d, want 1000", got.Paths)
	}
	if len(got.Bands) != 60 {
		t.Fatalf("bands len = %d, want 60", len(got.Bands))
	}
}

func TestRunMonteCarlo_ZeroOrNegativeYearsReturnsEmpty(t *testing.T) {
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio: 100000,
		CurrentAge:       90,
		LifeExpectancy:   90,
	})
	if len(got.Bands) != 0 {
		t.Errorf("expected no bands, got %d", len(got.Bands))
	}
}

func TestRunMonteCarlo_ZeroVolatilityYieldsDeterministicPath(t *testing.T) {
	// With sd=0, every path produces the same trajectory: 100k → growth +
	// contribution every year, no draws.
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio:   100000,
		AnnualContribution: 10000,
		ExpectedReturn:     5,
		Volatility:         0,
		CurrentAge:         30,
		RetirementAge:      60,
		LifeExpectancy:     32,
		AnnualWithdrawal:   0,
		Paths:              50,
	})
	if got.SuccessRate != 1.0 {
		t.Errorf("success_rate = %v, want 1.0 (zero volatility, accumulating)", got.SuccessRate)
	}
	// All bands should collapse: p5 == p50 == p95.
	for _, b := range got.Bands {
		if b.P5 != b.P50 || b.P50 != b.P95 {
			t.Errorf("age %d: bands diverge with zero volatility (p5=%v, p50=%v, p95=%v)", b.Age, b.P5, b.P50, b.P95)
			break
		}
	}
}

func TestRunMonteCarlo_AgeAxisIsCurrentAgePlusOneToLifeExpectancy(t *testing.T) {
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio: 10000,
		ExpectedReturn:   5,
		Volatility:       10,
		CurrentAge:       40,
		RetirementAge:    65,
		LifeExpectancy:   45,
		Paths:            10,
	})
	if got.Bands[0].Age != 41 {
		t.Errorf("first band age = %d, want 41", got.Bands[0].Age)
	}
	if got.Bands[len(got.Bands)-1].Age != 45 {
		t.Errorf("last band age = %d, want 45", got.Bands[len(got.Bands)-1].Age)
	}
}

func TestRunMonteCarlo_BalanceClampedAtZero(t *testing.T) {
	// Aggressive withdrawal + tiny portfolio guarantees most paths bust.
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio: 1000,
		AnnualWithdrawal: 5000,
		ExpectedReturn:   3,
		Volatility:       10,
		CurrentAge:       65,
		RetirementAge:    65,
		LifeExpectancy:   90,
		Paths:            200,
	})
	for _, b := range got.Bands {
		if b.P5 < 0 || b.P50 < 0 || b.P95 < 0 {
			t.Errorf("age %d: negative band detected (p5=%v p50=%v p95=%v)", b.Age, b.P5, b.P50, b.P95)
		}
	}
	if got.SuccessRate < 0 || got.SuccessRate > 1 {
		t.Errorf("success_rate out of [0,1]: %v", got.SuccessRate)
	}
}

func TestPercentile_KnownValues(t *testing.T) {
	cases := []struct {
		name   string
		sorted []float64
		pct    float64
		want   float64
	}{
		{"empty returns 0", []float64{}, 50, 0},
		{"single element", []float64{42}, 50, 42},
		{"p0 = min", []float64{1, 2, 3, 4, 5}, 0, 1},
		{"p100 = max", []float64{1, 2, 3, 4, 5}, 100, 5},
		{"p50 of 1..5 = 3 (linear interp)", []float64{1, 2, 3, 4, 5}, 50, 3},
		{"p25 of 1..5 = 2.0", []float64{1, 2, 3, 4, 5}, 25, 2.0},
		{"p75 of 1..5 = 4.0", []float64{1, 2, 3, 4, 5}, 75, 4.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := percentile(tc.sorted, tc.pct)
			if got != tc.want {
				t.Errorf("percentile(%v, %v) = %v, want %v", tc.sorted, tc.pct, got, tc.want)
			}
		})
	}
}

func TestDeriveAssumptions_AllStocksMatchesStockConstants(t *testing.T) {
	r, v := DeriveAssumptions(MonteCarloAllocation{StocksPct: 100})
	if r != AssetReturnStocksPct {
		t.Errorf("100%% stocks return = %v, want %v", r, AssetReturnStocksPct)
	}
	if v != AssetVolStocksPct {
		t.Errorf("100%% stocks vol = %v, want %v", v, AssetVolStocksPct)
	}
}

func TestDeriveAssumptions_AllCashMatchesCashConstants(t *testing.T) {
	r, v := DeriveAssumptions(MonteCarloAllocation{CashPct: 100})
	if r != AssetReturnCashPct {
		t.Errorf("100%% cash return = %v, want %v", r, AssetReturnCashPct)
	}
	if v != AssetVolCashPct {
		t.Errorf("100%% cash vol = %v, want %v", v, AssetVolCashPct)
	}
}

func TestDeriveAssumptions_BalancedDiversifies(t *testing.T) {
	// 50/50 stocks/bonds: return is the simple weighted mean; volatility
	// (zero-corr) is strictly less than the weighted mean of vols thanks
	// to diversification.
	r, v := DeriveAssumptions(MonteCarloAllocation{StocksPct: 50, BondsPct: 50})
	wantR := 0.5*AssetReturnStocksPct + 0.5*AssetReturnBondsPct
	if r != wantR {
		t.Errorf("50/50 return = %v, want %v", r, wantR)
	}
	weightedVol := 0.5*AssetVolStocksPct + 0.5*AssetVolBondsPct
	if v >= weightedVol {
		t.Errorf("50/50 vol = %v, expected < weighted mean %v (diversification)", v, weightedVol)
	}
}

func TestRunMonteCarlo_AllocationOverridesManualAssumptions(t *testing.T) {
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio: 100000,
		ExpectedReturn:   99, // should be ignored
		Volatility:       99, // should be ignored
		CurrentAge:       30,
		RetirementAge:    65,
		LifeExpectancy:   90,
		AnnualWithdrawal: 30000,
		Paths:            100,
		Allocation:       &MonteCarloAllocation{StocksPct: 60, BondsPct: 30, CashPct: 10},
	})
	if got.Assumptions.Source != "allocation" {
		t.Errorf("source = %q, want %q", got.Assumptions.Source, "allocation")
	}
	wantR, wantV := DeriveAssumptions(MonteCarloAllocation{StocksPct: 60, BondsPct: 30, CashPct: 10})
	if math.Abs(got.Assumptions.ExpectedReturn-wantR) > 1e-9 {
		t.Errorf("ER = %v, want %v", got.Assumptions.ExpectedReturn, wantR)
	}
	if math.Abs(got.Assumptions.Volatility-wantV) > 1e-9 {
		t.Errorf("vol = %v, want %v", got.Assumptions.Volatility, wantV)
	}
	if got.Assumptions.Allocation == nil || got.Assumptions.Allocation.StocksPct != 60 {
		t.Errorf("allocation echo missing or wrong: %+v", got.Assumptions.Allocation)
	}
}

func TestRunMonteCarlo_ManualSourceWhenNoAllocation(t *testing.T) {
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio: 100000,
		ExpectedReturn:   6,
		Volatility:       15,
		CurrentAge:       30,
		RetirementAge:    65,
		LifeExpectancy:   90,
		AnnualWithdrawal: 30000,
		Paths:            10,
	})
	if got.Assumptions.Source != "manual" {
		t.Errorf("source = %q, want %q", got.Assumptions.Source, "manual")
	}
	if got.Assumptions.ExpectedReturn != 6 || got.Assumptions.Volatility != 15 {
		t.Errorf("assumptions = %+v, want ER=6 vol=15", got.Assumptions)
	}
	if got.Assumptions.Allocation != nil {
		t.Errorf("allocation should be nil, got %+v", got.Assumptions.Allocation)
	}
}

func TestRunMonteCarlo_PercentilesAreSorted(t *testing.T) {
	// Across reasonable inputs p5 <= p50 <= p95 at every age.
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio:   100000,
		AnnualContribution: 5000,
		ExpectedReturn:     6,
		Volatility:         15,
		CurrentAge:         30,
		RetirementAge:      65,
		LifeExpectancy:     90,
		AnnualWithdrawal:   30000,
		Paths:              500,
	})
	for _, b := range got.Bands {
		if b.P5 > b.P50 || b.P50 > b.P95 {
			t.Fatalf("age %d: percentile ordering broken (p5=%v p50=%v p95=%v)", b.Age, b.P5, b.P50, b.P95)
		}
	}
}

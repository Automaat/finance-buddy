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

func TestRunMonteCarlo_ZeroInflation_RealEqualsNominal(t *testing.T) {
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio:    100000,
		AnnualContribution:  10000,
		ExpectedReturn:      5,
		Volatility:          10,
		CurrentAge:          30,
		RetirementAge:       60,
		LifeExpectancy:      40,
		AnnualWithdrawal:    20000,
		Paths:               50,
		InflationMean:       0,
		InflationVolatility: 0,
	})
	for _, b := range got.Bands {
		if math.Abs(b.P5-b.P5Real) > 1e-9 || math.Abs(b.P50-b.P50Real) > 1e-9 || math.Abs(b.P95-b.P95Real) > 1e-9 {
			t.Errorf("age %d: zero-inflation should make real == nominal (got p5=%v vs %v, p50=%v vs %v, p95=%v vs %v)",
				b.Age, b.P5, b.P5Real, b.P50, b.P50Real, b.P95, b.P95Real)
		}
		if math.Abs(b.Spending-b.SpendingReal) > 1e-9 {
			t.Errorf("age %d: zero-inflation spending real (%v) should equal nominal (%v)", b.Age, b.SpendingReal, b.Spending)
		}
	}
}

func TestRunMonteCarlo_FixedInflation_DeflatesGeometrically(t *testing.T) {
	// Zero return volatility AND zero inflation volatility => deterministic
	// path. Real balance at year Y should equal nominal / (1+inf)^Y exactly.
	infMean := 3.0
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio:    100000,
		AnnualContribution:  0,
		ExpectedReturn:      5,
		Volatility:          0,
		CurrentAge:          60,
		RetirementAge:       60,
		LifeExpectancy:      65,
		AnnualWithdrawal:    0,
		Paths:               10,
		InflationMean:       infMean,
		InflationVolatility: 0,
	})
	for y, b := range got.Bands {
		factor := math.Pow(1+infMean/100, float64(y+1))
		wantReal := b.P50 / factor
		if math.Abs(b.P50Real-wantReal) > 1e-6 {
			t.Errorf("age %d: real = %v, want %v (nominal %v / %v)", b.Age, b.P50Real, wantReal, b.P50, factor)
		}
	}
}

func TestRunMonteCarlo_FixedInflation_InflatesNominalWithdrawals(t *testing.T) {
	// Zero return vol + zero inflation vol => every path is identical. The
	// retiree's real spending stays at the supplied AnnualWithdrawal; the
	// nominal spending grows by (1+inf)^yearsSinceRetirement.
	infMean := 3.0
	withdrawal := 40000.0
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio:    10_000_000,
		AnnualContribution:  0,
		ExpectedReturn:      0,
		Volatility:          0,
		CurrentAge:          60,
		RetirementAge:       60,
		LifeExpectancy:      65,
		AnnualWithdrawal:    withdrawal,
		Paths:               5,
		InflationMean:       infMean,
		InflationVolatility: 0,
	})
	for y, b := range got.Bands {
		wantNominal := withdrawal * math.Pow(1+infMean/100, float64(y+1))
		if math.Abs(b.Spending-wantNominal) > 1e-6 {
			t.Errorf("age %d: nominal spend = %v, want %v", b.Age, b.Spending, wantNominal)
		}
		if math.Abs(b.SpendingReal-withdrawal) > 1e-6 {
			t.Errorf("age %d: real spend = %v, want constant %v", b.Age, b.SpendingReal, withdrawal)
		}
	}
}

func TestRunMonteCarlo_NegativeInflation_RealOutpacesNominal(t *testing.T) {
	// Deterministic deflation of 2% per year: nominal balance is unchanged
	// (zero return), but real balance grows because each PLN is worth more.
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio:    100000,
		AnnualContribution:  0,
		ExpectedReturn:      0,
		Volatility:          0,
		CurrentAge:          60,
		RetirementAge:       60,
		LifeExpectancy:      65,
		AnnualWithdrawal:    0,
		Paths:               5,
		InflationMean:       -2,
		InflationVolatility: 0,
	})
	last := got.Bands[len(got.Bands)-1]
	if last.P50Real <= last.P50 {
		t.Errorf("deflation should make real (%v) > nominal (%v)", last.P50Real, last.P50)
	}
}

func TestRunMonteCarlo_BustedPath_SpendingCapped(t *testing.T) {
	// Tiny portfolio + giant withdrawal forces every path to bust quickly.
	// After bust, recorded spending must never exceed the portfolio's
	// pre-bust real balance — i.e. SpendingReal cannot exceed AnnualWithdrawal
	// (which is the real per-year ask).
	withdrawal := 50000.0
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio:    1000,
		AnnualContribution:  0,
		ExpectedReturn:      0,
		Volatility:          0,
		CurrentAge:          65,
		RetirementAge:       65,
		LifeExpectancy:      75,
		AnnualWithdrawal:    withdrawal,
		Paths:               5,
		InflationMean:       3,
		InflationVolatility: 0,
	})
	for _, b := range got.Bands {
		if b.SpendingReal > withdrawal+1e-6 {
			t.Errorf("age %d: real spending %v exceeds the requested %v on a busted path",
				b.Age, b.SpendingReal, withdrawal)
		}
		if b.P50 < 0 || b.P50Real < 0 {
			t.Errorf("age %d: negative balance band p50=%v p50_real=%v", b.Age, b.P50, b.P50Real)
		}
	}
}

func TestRunMonteCarlo_AssumptionsEchoesInflation(t *testing.T) {
	got := RunMonteCarlo(newSeededRng(), MonteCarloInputs{
		CurrentPortfolio:    100000,
		CurrentAge:          30,
		RetirementAge:       65,
		LifeExpectancy:      90,
		ExpectedReturn:      6,
		Volatility:          15,
		AnnualWithdrawal:    30000,
		Paths:               10,
		InflationMean:       3.2,
		InflationVolatility: 1.5,
	})
	if got.Assumptions.InflationMean != 3.2 || got.Assumptions.InflationVolatility != 1.5 {
		t.Errorf("inflation assumptions echo = %+v, want mean=3.2 vol=1.5", got.Assumptions)
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

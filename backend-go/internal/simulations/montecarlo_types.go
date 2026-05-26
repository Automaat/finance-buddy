package simulations

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

// Polish withdrawal-phase tax rates (as fractions, not percent).
const (
	BelkaRate       = 0.19 // 19% on realized capital gains (taxable brokerage)
	IkzePostAgeRate = 0.10 // 10% PIT on IKZE withdrawals post-65
	IkeFreeAge      = 60   // IKE withdrawals tax-free from this age
	IkzeFreeAge     = 65   // IKZE switches from Belka-like penalty to 10% PIT
)

// MonteCarloAllocation captures the stocks/bonds/cash split (percent,
// must sum to 100). When set on MonteCarloInputs, the simulation derives
// ExpectedReturn and Volatility from the per-asset constants above.
type MonteCarloAllocation struct {
	StocksPct float64
	BondsPct  float64
	CashPct   float64
}

// MonteCarloAccountMix expresses what share of each PLN withdrawn comes
// from which Polish wrapper, plus the gain fraction used for Belka. Pcts
// must sum to 100. When set, the simulation grosses up withdrawals to
// cover tax and reports both gross and net-of-tax bands.
//
// TaxableGainPct is the gain fraction (vs cost basis) of the taxable
// brokerage balance. The same fraction is used as a proxy for IKE
// withdrawals before age 60 and IKZE withdrawals before age 65 - early
// withdrawals from those wrappers are taxed like brokerage gains in our
// model, so they need a gain-fraction assumption too.
type MonteCarloAccountMix struct {
	TaxablePct     float64
	IkePct         float64
	IkzePct        float64
	ZusPct         float64
	TaxableGainPct float64
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
	// AccountMix, when non-nil, makes AnnualWithdrawal a net (after-tax)
	// target and grosses up the portfolio debit by the per-age effective
	// tax rate. Also enables the net-of-tax balance bands and the
	// effective-lifetime-tax-rate metric.
	AccountMix *MonteCarloAccountMix
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
	// Net* are the balance bands after subtracting the estimated tax
	// owed on a full liquidation at this age (per the AccountMix). When
	// AccountMix is nil they mirror the gross bands.
	P5Net      float64 `json:"p5_net"`
	P50Net     float64 `json:"p50_net"`
	P95Net     float64 `json:"p95_net"`
	P5NetReal  float64 `json:"p5_net_real"`
	P50NetReal float64 `json:"p50_net_real"`
	P95NetReal float64 `json:"p95_net_real"`
	// Spending is the average realized nominal withdrawal (net to the
	// retiree) at this age (zero in the accumulation phase). SpendingReal
	// is that same value deflated by the path's cumulative inflation
	// factor. Both averaged across paths so the chart can show one curve.
	Spending     float64 `json:"spending"`
	SpendingReal float64 `json:"spending_real"`
}

// MonteCarloTaxSummary aggregates Belka / IKZE PIT across all paths and
// all retirement years. EffectiveLifetimeRate = TaxTotal / GrossTotal,
// i.e. the average tax wedge per gross PLN drawn from the portfolio.
type MonteCarloTaxSummary struct {
	GrossWithdrawalsTotal float64 `json:"gross_withdrawals_total"`
	TaxTotal              float64 `json:"tax_total"`
	EffectiveLifetimeRate float64 `json:"effective_lifetime_rate"`
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
	AccountMix          *MonteCarloAccountMixOut `json:"account_mix,omitempty"`
}

// MonteCarloAccountMixOut echoes the account mix supplied in the request.
type MonteCarloAccountMixOut struct {
	TaxablePct     float64 `json:"taxable_pct"`
	IkePct         float64 `json:"ike_pct"`
	IkzePct        float64 `json:"ikze_pct"`
	ZusPct         float64 `json:"zus_pct"`
	TaxableGainPct float64 `json:"taxable_gain_pct"`
}

// MonteCarloAllocationOut is the wire echo of the allocation split.
type MonteCarloAllocationOut struct {
	StocksPct float64 `json:"stocks_pct"`
	BondsPct  float64 `json:"bonds_pct"`
	CashPct   float64 `json:"cash_pct"`
}

// MonteCarloResult is what the handler ships back. SuccessRate is the
// share of paths with non-negative ending balance at life_expectancy.
// Tax is present only when AccountMix was supplied.
type MonteCarloResult struct {
	SuccessRate float64                    `json:"success_rate"`
	Bands       []MonteCarloPercentileBand `json:"bands"`
	Paths       int                        `json:"paths"`
	Assumptions MonteCarloAssumptions      `json:"assumptions"`
	Tax         *MonteCarloTaxSummary      `json:"tax,omitempty"`
}

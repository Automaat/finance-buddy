package simulations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

const safeWithdrawalRate = 0.04

// Handler is the HTTP boundary for /api/simulations.
type Handler struct {
	store  *Store
	logger *slog.Logger
	now    func() time.Time
}

// NewHandler wires the store + logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger, now: time.Now}
}

// --- mortgage wire types ---

type mortgageInputsWire struct {
	RemainingPrincipal   wire.PyFloat `json:"remaining_principal"`
	AnnualInterestRate   wire.PyFloat `json:"annual_interest_rate"`
	RemainingMonths      int          `json:"remaining_months"`
	TotalMonthlyBudget   wire.PyFloat `json:"total_monthly_budget"`
	ExpectedAnnualReturn wire.PyFloat `json:"expected_annual_return"`
	InflationRate        wire.PyFloat `json:"inflation_rate"`
	EnableVariableRate   bool         `json:"enable_variable_rate"`
}

type mortgageYearlyWire struct {
	Year                         int          `json:"year"`
	AnnualRate                   wire.PyFloat `json:"annual_rate"`
	ScenarioAMortgageBalance     wire.PyFloat `json:"scenario_a_mortgage_balance"`
	ScenarioARealMortgageBalance wire.PyFloat `json:"scenario_a_real_mortgage_balance"`
	ScenarioACumulativeInterest  wire.PyFloat `json:"scenario_a_cumulative_interest"`
	ScenarioAInvestmentBalance   wire.PyFloat `json:"scenario_a_investment_balance"`
	ScenarioAAfterTaxPortfolio   wire.PyFloat `json:"scenario_a_after_tax_portfolio"`
	ScenarioARealPortfolio       wire.PyFloat `json:"scenario_a_real_portfolio"`
	ScenarioAPaidOff             bool         `json:"scenario_a_paid_off"`
	ScenarioBMortgageBalance     wire.PyFloat `json:"scenario_b_mortgage_balance"`
	ScenarioBRealMortgageBalance wire.PyFloat `json:"scenario_b_real_mortgage_balance"`
	ScenarioBInvestmentBalance   wire.PyFloat `json:"scenario_b_investment_balance"`
	ScenarioBAfterTaxPortfolio   wire.PyFloat `json:"scenario_b_after_tax_portfolio"`
	ScenarioBRealPortfolio       wire.PyFloat `json:"scenario_b_real_portfolio"`
	ScenarioBCumulativeInterest  wire.PyFloat `json:"scenario_b_cumulative_interest"`
	NetAdvantageInvest           wire.PyFloat `json:"net_advantage_invest"`
}

type mortgageSummaryWire struct {
	RegularMonthlyPayment    wire.PyFloat `json:"regular_monthly_payment"`
	TotalInterestA           wire.PyFloat `json:"total_interest_a"`
	TotalInterestB           wire.PyFloat `json:"total_interest_b"`
	InterestSaved            wire.PyFloat `json:"interest_saved"`
	FinalInvestmentPortfolio wire.PyFloat `json:"final_investment_portfolio"`
	BelkaTaxA                wire.PyFloat `json:"belka_tax_a"`
	BelkaTaxB                wire.PyFloat `json:"belka_tax_b"`
	FinalPortfolioAReal      wire.PyFloat `json:"final_portfolio_a_real"`
	FinalPortfolioBReal      wire.PyFloat `json:"final_portfolio_b_real"`
	MonthsSaved              int          `json:"months_saved"`
	WinningStrategy          string       `json:"winning_strategy"`
	NetAdvantage             wire.PyFloat `json:"net_advantage"`
	BreakEvenGrossReturn     wire.PyFloat `json:"break_even_gross_return"`
}

type mortgageResponse struct {
	Inputs            mortgageInputsWire   `json:"inputs"`
	YearlyProjections []mortgageYearlyWire `json:"yearly_projections"`
	Summary           mortgageSummaryWire  `json:"summary"`
}

// MortgageVsInvest serves POST /api/simulations/mortgage-vs-invest.
func (h *Handler) MortgageVsInvest(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	in, vErr := buildMortgageInputs(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	result, err := SimulateMortgageVsInvest(in)
	if err != nil {
		var budgetErr *BudgetTooLowError
		if errors.As(err, &budgetErr) {
			httputil.WriteDetailError(w, http.StatusBadRequest,
				fmt.Sprintf("Total monthly budget (%.2f PLN) is less than "+
					"the required mortgage payment (%.2f PLN)",
					budgetErr.Budget, budgetErr.Payment))
			return
		}
		h.logger.Error("mortgage sim", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resp := mortgageResponse{
		Inputs: mortgageInputsWire{
			RemainingPrincipal:   wire.PyFloat(in.RemainingPrincipal),
			AnnualInterestRate:   wire.PyFloat(in.AnnualInterestRate),
			RemainingMonths:      in.RemainingMonths,
			TotalMonthlyBudget:   wire.PyFloat(in.TotalMonthlyBudget),
			ExpectedAnnualReturn: wire.PyFloat(in.ExpectedAnnualReturn),
			InflationRate:        wire.PyFloat(in.InflationRate),
			EnableVariableRate:   in.EnableVariableRate,
		},
		Summary: mortgageSummaryWire{
			RegularMonthlyPayment:    wire.PyFloat(result.Summary.RegularMonthlyPayment),
			TotalInterestA:           wire.PyFloat(result.Summary.TotalInterestA),
			TotalInterestB:           wire.PyFloat(result.Summary.TotalInterestB),
			InterestSaved:            wire.PyFloat(result.Summary.InterestSaved),
			FinalInvestmentPortfolio: wire.PyFloat(result.Summary.FinalInvestmentPortfolio),
			FinalPortfolioAReal:      wire.PyFloat(result.Summary.FinalPortfolioAReal),
			FinalPortfolioBReal:      wire.PyFloat(result.Summary.FinalPortfolioBReal),
			MonthsSaved:              result.Summary.MonthsSaved,
			WinningStrategy:          result.Summary.WinningStrategy,
			NetAdvantage:             wire.PyFloat(result.Summary.NetAdvantage),
			BreakEvenGrossReturn:     wire.PyFloat(result.Summary.BreakEvenGrossReturn),
		},
	}
	resp.YearlyProjections = make([]mortgageYearlyWire, 0, len(result.Yearly))
	for i := range result.Yearly {
		y := &result.Yearly[i]
		resp.YearlyProjections = append(resp.YearlyProjections, mortgageYearlyWire{
			Year: y.Year, AnnualRate: wire.PyFloat(y.AnnualRate),
			ScenarioAMortgageBalance:     wire.PyFloat(y.ScenarioAMortgageBalance),
			ScenarioARealMortgageBalance: wire.PyFloat(y.ScenarioARealMortgageBalance),
			ScenarioACumulativeInterest:  wire.PyFloat(y.ScenarioACumulativeInterest),
			ScenarioAInvestmentBalance:   wire.PyFloat(y.ScenarioAInvestmentBalance),
			ScenarioAAfterTaxPortfolio:   wire.PyFloat(y.ScenarioAAfterTaxPortfolio),
			ScenarioARealPortfolio:       wire.PyFloat(y.ScenarioARealPortfolio),
			ScenarioAPaidOff:             y.ScenarioAPaidOff,
			ScenarioBMortgageBalance:     wire.PyFloat(y.ScenarioBMortgageBalance),
			ScenarioBRealMortgageBalance: wire.PyFloat(y.ScenarioBRealMortgageBalance),
			ScenarioBInvestmentBalance:   wire.PyFloat(y.ScenarioBInvestmentBalance),
			ScenarioBAfterTaxPortfolio:   wire.PyFloat(y.ScenarioBAfterTaxPortfolio),
			ScenarioBRealPortfolio:       wire.PyFloat(y.ScenarioBRealPortfolio),
			ScenarioBCumulativeInterest:  wire.PyFloat(y.ScenarioBCumulativeInterest),
			NetAdvantageInvest:           wire.PyFloat(y.NetAdvantageInvest),
		})
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}

// --- WIBOR scenarios wire types ---

type wiborInputsWire struct {
	RemainingPrincipal wire.PyFloat `json:"remaining_principal"`
	BaseAnnualRate     wire.PyFloat `json:"base_annual_rate"`
	RemainingMonths    int          `json:"remaining_months"`
	BasePayment        wire.PyFloat `json:"base_payment"`
}

type wiborScenarioWire struct {
	DeltaPP        wire.PyFloat   `json:"delta_pp"`
	AnnualRate     wire.PyFloat   `json:"annual_rate"`
	MonthlyPayment wire.PyFloat   `json:"monthly_payment"`
	TotalInterest  wire.PyFloat   `json:"total_interest"`
	TermMonths     int            `json:"term_months"`
	RateFloored    bool           `json:"rate_floored"`
	YearlyBalances []wire.PyFloat `json:"yearly_balances"`
}

type wiborResponse struct {
	Inputs    wiborInputsWire     `json:"inputs"`
	Scenarios []wiborScenarioWire `json:"scenarios"`
}

// WiborScenarios serves POST /api/simulations/wibor.
func (h *Handler) WiborScenarios(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	in, vErr := buildWiborInputs(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	result := SimulateWiborScenarios(in, DefaultWiborDeltas)
	resp := wiborResponse{
		Inputs: wiborInputsWire{
			RemainingPrincipal: wire.PyFloat(in.RemainingPrincipal),
			BaseAnnualRate:     wire.PyFloat(in.BaseAnnualRate),
			RemainingMonths:    in.RemainingMonths,
			BasePayment:        wire.PyFloat(result.BasePayment),
		},
	}
	resp.Scenarios = make([]wiborScenarioWire, 0, len(result.Scenarios))
	for _, s := range result.Scenarios {
		balances := make([]wire.PyFloat, 0, len(s.YearlyBalances))
		for _, b := range s.YearlyBalances {
			balances = append(balances, wire.PyFloat(b))
		}
		resp.Scenarios = append(resp.Scenarios, wiborScenarioWire{
			DeltaPP:        wire.PyFloat(s.DeltaPP),
			AnnualRate:     wire.PyFloat(s.AnnualRate),
			MonthlyPayment: wire.PyFloat(s.MonthlyPayment),
			TotalInterest:  wire.PyFloat(s.TotalInterest),
			TermMonths:     s.TermMonths,
			RateFloored:    s.RateFloored,
			YearlyBalances: balances,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}

// Prefill serves GET /api/simulations/prefill.
func (h *Handler) Prefill(w http.ResponseWriter, r *http.Request) {
	data, err := h.store.LoadPrefill(r.Context())
	if err != nil {
		h.logger.Error("simulations prefill", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	balances := map[string]wire.PyFloat{}
	for k, v := range data.Balances {
		balances[k] = wire.PyFloat(v)
	}
	ppkBalances := map[string]wire.PyFloat{}
	for k, v := range data.PPKBalances {
		ppkBalances[k] = wire.PyFloat(v)
	}
	ppkRates := map[string]map[string]wire.PyFloat{}
	for owner, rates := range data.PPKRates {
		ppkRates[owner] = map[string]wire.PyFloat{
			"employee": wire.PyFloat(rates["employee"]),
			"employer": wire.PyFloat(rates["employer"]),
		}
	}
	salaries := map[string]*wire.PyFloat{}
	for k, v := range data.MonthlySalaries {
		if v == nil {
			salaries[k] = nil
			continue
		}
		pf := wire.PyFloat(*v)
		salaries[k] = &pf
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"current_age":      data.CurrentAge,
		"retirement_age":   data.RetirementAge,
		"balances":         balances,
		"ppk_rates":        ppkRates,
		"monthly_salaries": salaries,
		"ppk_balances":     ppkBalances,
	})
}

// Retirement serves POST /api/simulations/retirement.
func (h *Handler) Retirement(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<18)).Decode(&raw); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	in, vErr := buildSimulationInputs(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	sims, err := h.runSimulation(r.Context(), in)
	if err != nil {
		h.logger.Error("retirement sim", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, buildSimulationResponse(in, sims))
}

// monteCarloInputsWire mirrors MonteCarloInputs over JSON.
type monteCarloInputsWire struct {
	CurrentPortfolio    wire.PyFloat              `json:"current_portfolio"`
	AnnualContribution  wire.PyFloat              `json:"annual_contribution"`
	ExpectedReturn      wire.PyFloat              `json:"expected_return"`
	Volatility          wire.PyFloat              `json:"volatility"`
	CurrentAge          int                       `json:"current_age"`
	RetirementAge       int                       `json:"retirement_age"`
	LifeExpectancy      int                       `json:"life_expectancy"`
	AnnualWithdrawal    wire.PyFloat              `json:"annual_withdrawal"`
	Paths               int                       `json:"paths,omitempty"`
	Allocation          *monteCarloAllocationWire `json:"allocation,omitempty"`
	InflationMean       wire.PyFloat              `json:"inflation_mean"`
	InflationVolatility wire.PyFloat              `json:"inflation_volatility"`
	AccountMix          *monteCarloAccountMixWire `json:"account_mix,omitempty"`
}

type monteCarloAllocationWire struct {
	StocksPct wire.PyFloat `json:"stocks_pct"`
	BondsPct  wire.PyFloat `json:"bonds_pct"`
	CashPct   wire.PyFloat `json:"cash_pct"`
}

type monteCarloAccountMixWire struct {
	TaxablePct     wire.PyFloat `json:"taxable_pct"`
	IkePct         wire.PyFloat `json:"ike_pct"`
	IkzePct        wire.PyFloat `json:"ikze_pct"`
	ZusPct         wire.PyFloat `json:"zus_pct"`
	TaxableGainPct wire.PyFloat `json:"taxable_gain_pct"`
}

// MonteCarlo serves POST /api/simulations/monte-carlo. It runs `paths`
// Monte Carlo trajectories using N(expected_return, volatility) annual
// returns and reports the success rate plus 5/50/95 percentile bands.
func (h *Handler) MonteCarlo(w http.ResponseWriter, r *http.Request) {
	var in monteCarloInputsWire
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<18)).Decode(&in); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	if in.CurrentAge <= 0 || in.LifeExpectancy <= in.CurrentAge {
		httputil.WriteBodyValidationError(w, "life_expectancy", "life_expectancy must exceed current_age", "")
		return
	}
	if in.RetirementAge < in.CurrentAge || in.RetirementAge > in.LifeExpectancy {
		httputil.WriteBodyValidationError(w, "retirement_age", "retirement_age must be between current_age and life_expectancy", "")
		return
	}
	if in.CurrentPortfolio < 0 || in.AnnualContribution < 0 || in.AnnualWithdrawal < 0 {
		httputil.WriteBodyValidationError(w, "amount", "monetary inputs must be non-negative", "")
		return
	}
	if in.Paths < 0 || in.Paths > 10000 {
		httputil.WriteBodyValidationError(w, "paths", "paths must be in [0, 10000]", "")
		return
	}

	var allocation *MonteCarloAllocation
	if in.Allocation != nil {
		stocks := float64(in.Allocation.StocksPct)
		bonds := float64(in.Allocation.BondsPct)
		cash := float64(in.Allocation.CashPct)
		if stocks < 0 || bonds < 0 || cash < 0 {
			httputil.WriteBodyValidationError(w, "allocation", "allocation percentages must be non-negative", "")
			return
		}
		sum := stocks + bonds + cash
		if math.Abs(sum-100) > 0.01 {
			httputil.WriteBodyValidationError(w, "allocation",
				fmt.Sprintf("allocation must sum to 100 (got %.2f)", sum), "")
			return
		}
		allocation = &MonteCarloAllocation{StocksPct: stocks, BondsPct: bonds, CashPct: cash}
	} else if in.Volatility < 0 {
		httputil.WriteBodyValidationError(w, "volatility", "volatility must be non-negative", "")
		return
	}
	if in.InflationVolatility < 0 {
		httputil.WriteBodyValidationError(w, "inflation_volatility", "inflation_volatility must be non-negative", "")
		return
	}
	if in.InflationMean < -50 || in.InflationMean > 50 {
		httputil.WriteBodyValidationError(w, "inflation_mean", "inflation_mean must be within [-50, 50]", "")
		return
	}

	var accountMix *MonteCarloAccountMix
	if in.AccountMix != nil {
		taxable := float64(in.AccountMix.TaxablePct)
		ike := float64(in.AccountMix.IkePct)
		ikze := float64(in.AccountMix.IkzePct)
		zus := float64(in.AccountMix.ZusPct)
		gain := float64(in.AccountMix.TaxableGainPct)
		if taxable < 0 || ike < 0 || ikze < 0 || zus < 0 {
			httputil.WriteBodyValidationError(w, "account_mix", "account_mix percentages must be non-negative", "")
			return
		}
		if gain < 0 || gain > 100 {
			httputil.WriteBodyValidationError(w, "account_mix",
				"taxable_gain_pct must be within [0, 100]", "")
			return
		}
		sum := taxable + ike + ikze + zus
		if math.Abs(sum-100) > 0.01 {
			httputil.WriteBodyValidationError(w, "account_mix",
				fmt.Sprintf("account_mix must sum to 100 (got %.2f)", sum), "")
			return
		}
		accountMix = &MonteCarloAccountMix{
			TaxablePct: taxable, IkePct: ike, IkzePct: ikze, ZusPct: zus,
			TaxableGainPct: gain,
		}
	}

	rng := rand.New(rand.NewPCG(uint64(h.now().UnixNano()), 0xdeadbeef))
	result := RunMonteCarlo(rng, MonteCarloInputs{
		CurrentPortfolio:    float64(in.CurrentPortfolio),
		AnnualContribution:  float64(in.AnnualContribution),
		ExpectedReturn:      float64(in.ExpectedReturn),
		Volatility:          float64(in.Volatility),
		CurrentAge:          in.CurrentAge,
		RetirementAge:       in.RetirementAge,
		LifeExpectancy:      in.LifeExpectancy,
		AnnualWithdrawal:    float64(in.AnnualWithdrawal),
		Paths:               in.Paths,
		Allocation:          allocation,
		InflationMean:       float64(in.InflationMean),
		InflationVolatility: float64(in.InflationVolatility),
		AccountMix:          accountMix,
	})
	httputil.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) runSimulation(ctx context.Context, in simulationInputs) ([]AccountSimulation, error) {
	yearsToRetirement := in.RetirementAge - in.CurrentAge
	currentYear := h.now().UTC().Year()
	userNames, err := h.store.UserNames(ctx)
	if err != nil {
		return nil, err
	}
	sims := []AccountSimulation{}

	for _, acc := range in.IkeIkzeAccounts {
		if !acc.Enabled {
			continue
		}
		baseLimit, err := h.store.LimitForYear(ctx, currentYear, acc.Wrapper, acc.OwnerUserID)
		if err != nil {
			return nil, err
		}
		sims = append(sims, SimulateAccount(IkeIkzeParams{
			Wrapper:         acc.Wrapper,
			OwnerUserID:     acc.OwnerUserID,
			OwnerName:       ownerLabel(userNames, acc.OwnerUserID),
			StartingBalance: acc.Balance, AutoFillLimit: acc.AutoFillLimit,
			MonthlyContribution: acc.MonthlyContribution, TaxRate: acc.TaxRate,
			BaseLimit: baseLimit,
		}, yearsToRetirement, in.CurrentAge, currentYear, in.AnnualReturnRate, in.LimitGrowthRate))
	}
	for _, ppk := range in.PPKAccounts {
		if !ppk.Enabled {
			continue
		}
		sims = append(sims, SimulatePPKAccount(PPKParams{
			OwnerUserID:        ppk.OwnerUserID,
			OwnerName:          ownerLabel(userNames, ppk.OwnerUserID),
			StartingBalance:    ppk.StartingBalance,
			MonthlyGrossSalary: ppk.MonthlyGrossSalary,
			EmployeeRate:       ppk.EmployeeRate, EmployerRate: ppk.EmployerRate,
			SalaryBelowThreshold: ppk.SalaryBelowThreshold,
			IncludeWelcomeBonus:  ppk.IncludeWelcomeBonus,
			IncludeAnnualSubsidy: ppk.IncludeAnnualSubsidy,
		}, in.CurrentAge, in.RetirementAge, in.ExpectedSalaryGrowth))
	}
	for _, brk := range in.BrokerageAccounts {
		if !brk.Enabled {
			continue
		}
		sims = append(sims, SimulateBrokerageAccount(BrokerageParams{
			OwnerUserID:         brk.OwnerUserID,
			OwnerName:           ownerLabel(userNames, brk.OwnerUserID),
			StartingBalance:     brk.Balance,
			MonthlyContribution: brk.MonthlyContribution,
		}, in.CurrentAge, in.RetirementAge, currentYear, in.AnnualReturnRate))
	}
	return sims, nil
}

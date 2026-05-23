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
	"strconv"
	"strings"
	"time"
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
	RemainingPrincipal   pyFloat `json:"remaining_principal"`
	AnnualInterestRate   pyFloat `json:"annual_interest_rate"`
	RemainingMonths      int     `json:"remaining_months"`
	TotalMonthlyBudget   pyFloat `json:"total_monthly_budget"`
	ExpectedAnnualReturn pyFloat `json:"expected_annual_return"`
	InflationRate        pyFloat `json:"inflation_rate"`
	EnableVariableRate   bool    `json:"enable_variable_rate"`
}

type mortgageYearlyWire struct {
	Year                         int     `json:"year"`
	AnnualRate                   pyFloat `json:"annual_rate"`
	ScenarioAMortgageBalance     pyFloat `json:"scenario_a_mortgage_balance"`
	ScenarioARealMortgageBalance pyFloat `json:"scenario_a_real_mortgage_balance"`
	ScenarioACumulativeInterest  pyFloat `json:"scenario_a_cumulative_interest"`
	ScenarioAInvestmentBalance   pyFloat `json:"scenario_a_investment_balance"`
	ScenarioAAfterTaxPortfolio   pyFloat `json:"scenario_a_after_tax_portfolio"`
	ScenarioARealPortfolio       pyFloat `json:"scenario_a_real_portfolio"`
	ScenarioAPaidOff             bool    `json:"scenario_a_paid_off"`
	ScenarioBMortgageBalance     pyFloat `json:"scenario_b_mortgage_balance"`
	ScenarioBRealMortgageBalance pyFloat `json:"scenario_b_real_mortgage_balance"`
	ScenarioBInvestmentBalance   pyFloat `json:"scenario_b_investment_balance"`
	ScenarioBAfterTaxPortfolio   pyFloat `json:"scenario_b_after_tax_portfolio"`
	ScenarioBRealPortfolio       pyFloat `json:"scenario_b_real_portfolio"`
	ScenarioBCumulativeInterest  pyFloat `json:"scenario_b_cumulative_interest"`
	NetAdvantageInvest           pyFloat `json:"net_advantage_invest"`
}

type mortgageSummaryWire struct {
	RegularMonthlyPayment    pyFloat `json:"regular_monthly_payment"`
	TotalInterestA           pyFloat `json:"total_interest_a"`
	TotalInterestB           pyFloat `json:"total_interest_b"`
	InterestSaved            pyFloat `json:"interest_saved"`
	FinalInvestmentPortfolio pyFloat `json:"final_investment_portfolio"`
	BelkaTaxA                pyFloat `json:"belka_tax_a"`
	BelkaTaxB                pyFloat `json:"belka_tax_b"`
	FinalPortfolioAReal      pyFloat `json:"final_portfolio_a_real"`
	FinalPortfolioBReal      pyFloat `json:"final_portfolio_b_real"`
	MonthsSaved              int     `json:"months_saved"`
	WinningStrategy          string  `json:"winning_strategy"`
	NetAdvantage             pyFloat `json:"net_advantage"`
	BreakEvenGrossReturn     pyFloat `json:"break_even_gross_return"`
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
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	in, vErr := buildMortgageInputs(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	result, err := SimulateMortgageVsInvest(in)
	if err != nil {
		var budgetErr *BudgetTooLowError
		if errors.As(err, &budgetErr) {
			writeDetailError(w, http.StatusBadRequest,
				fmt.Sprintf("Total monthly budget (%.2f PLN) is less than "+
					"the required mortgage payment (%.2f PLN)",
					budgetErr.Budget, budgetErr.Payment))
			return
		}
		h.logger.Error("mortgage sim", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resp := mortgageResponse{
		Inputs: mortgageInputsWire{
			RemainingPrincipal:   pyFloat(in.RemainingPrincipal),
			AnnualInterestRate:   pyFloat(in.AnnualInterestRate),
			RemainingMonths:      in.RemainingMonths,
			TotalMonthlyBudget:   pyFloat(in.TotalMonthlyBudget),
			ExpectedAnnualReturn: pyFloat(in.ExpectedAnnualReturn),
			InflationRate:        pyFloat(in.InflationRate),
			EnableVariableRate:   in.EnableVariableRate,
		},
		Summary: mortgageSummaryWire{
			RegularMonthlyPayment:    pyFloat(result.Summary.RegularMonthlyPayment),
			TotalInterestA:           pyFloat(result.Summary.TotalInterestA),
			TotalInterestB:           pyFloat(result.Summary.TotalInterestB),
			InterestSaved:            pyFloat(result.Summary.InterestSaved),
			FinalInvestmentPortfolio: pyFloat(result.Summary.FinalInvestmentPortfolio),
			FinalPortfolioAReal:      pyFloat(result.Summary.FinalPortfolioAReal),
			FinalPortfolioBReal:      pyFloat(result.Summary.FinalPortfolioBReal),
			MonthsSaved:              result.Summary.MonthsSaved,
			WinningStrategy:          result.Summary.WinningStrategy,
			NetAdvantage:             pyFloat(result.Summary.NetAdvantage),
			BreakEvenGrossReturn:     pyFloat(result.Summary.BreakEvenGrossReturn),
		},
	}
	resp.YearlyProjections = make([]mortgageYearlyWire, 0, len(result.Yearly))
	for i := range result.Yearly {
		y := &result.Yearly[i]
		resp.YearlyProjections = append(resp.YearlyProjections, mortgageYearlyWire{
			Year: y.Year, AnnualRate: pyFloat(y.AnnualRate),
			ScenarioAMortgageBalance:     pyFloat(y.ScenarioAMortgageBalance),
			ScenarioARealMortgageBalance: pyFloat(y.ScenarioARealMortgageBalance),
			ScenarioACumulativeInterest:  pyFloat(y.ScenarioACumulativeInterest),
			ScenarioAInvestmentBalance:   pyFloat(y.ScenarioAInvestmentBalance),
			ScenarioAAfterTaxPortfolio:   pyFloat(y.ScenarioAAfterTaxPortfolio),
			ScenarioARealPortfolio:       pyFloat(y.ScenarioARealPortfolio),
			ScenarioAPaidOff:             y.ScenarioAPaidOff,
			ScenarioBMortgageBalance:     pyFloat(y.ScenarioBMortgageBalance),
			ScenarioBRealMortgageBalance: pyFloat(y.ScenarioBRealMortgageBalance),
			ScenarioBInvestmentBalance:   pyFloat(y.ScenarioBInvestmentBalance),
			ScenarioBAfterTaxPortfolio:   pyFloat(y.ScenarioBAfterTaxPortfolio),
			ScenarioBRealPortfolio:       pyFloat(y.ScenarioBRealPortfolio),
			ScenarioBCumulativeInterest:  pyFloat(y.ScenarioBCumulativeInterest),
			NetAdvantageInvest:           pyFloat(y.NetAdvantageInvest),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// Prefill serves GET /api/simulations/prefill.
func (h *Handler) Prefill(w http.ResponseWriter, r *http.Request) {
	data, err := h.store.LoadPrefill(r.Context())
	if err != nil {
		h.logger.Error("simulations prefill", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	balances := map[string]pyFloat{}
	for k, v := range data.Balances {
		balances[k] = pyFloat(v)
	}
	ppkBalances := map[string]pyFloat{}
	for k, v := range data.PPKBalances {
		ppkBalances[k] = pyFloat(v)
	}
	ppkRates := map[string]map[string]pyFloat{}
	for owner, rates := range data.PPKRates {
		ppkRates[owner] = map[string]pyFloat{
			"employee": pyFloat(rates["employee"]),
			"employer": pyFloat(rates["employer"]),
		}
	}
	salaries := map[string]*pyFloat{}
	for k, v := range data.MonthlySalaries {
		if v == nil {
			salaries[k] = nil
			continue
		}
		pf := pyFloat(*v)
		salaries[k] = &pf
	}
	writeJSON(w, http.StatusOK, map[string]any{
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
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	in, vErr := buildSimulationInputs(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	sims, err := h.runSimulation(r.Context(), in)
	if err != nil {
		h.logger.Error("retirement sim", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, h.buildSimulationResponse(in, sims))
}

// monteCarloInputsWire mirrors MonteCarloInputs over JSON.
type monteCarloInputsWire struct {
	CurrentPortfolio   pyFloat `json:"current_portfolio"`
	AnnualContribution pyFloat `json:"annual_contribution"`
	ExpectedReturn     pyFloat `json:"expected_return"`
	Volatility         pyFloat `json:"volatility"`
	CurrentAge         int     `json:"current_age"`
	RetirementAge      int     `json:"retirement_age"`
	LifeExpectancy     int     `json:"life_expectancy"`
	AnnualWithdrawal   pyFloat `json:"annual_withdrawal"`
	Paths              int     `json:"paths,omitempty"`
}

// MonteCarlo serves POST /api/simulations/monte-carlo. It runs `paths`
// Monte Carlo trajectories using N(expected_return, volatility) annual
// returns and reports the success rate plus 5/50/95 percentile bands.
func (h *Handler) MonteCarlo(w http.ResponseWriter, r *http.Request) {
	var in monteCarloInputsWire
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<18)).Decode(&in); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	if in.CurrentAge <= 0 || in.LifeExpectancy <= in.CurrentAge {
		writeValidationError(w, "life_expectancy", "life_expectancy must exceed current_age", "")
		return
	}
	if in.RetirementAge < in.CurrentAge || in.RetirementAge > in.LifeExpectancy {
		writeValidationError(w, "retirement_age", "retirement_age must be between current_age and life_expectancy", "")
		return
	}
	if in.Volatility < 0 {
		writeValidationError(w, "volatility", "volatility must be non-negative", "")
		return
	}
	if in.CurrentPortfolio < 0 || in.AnnualContribution < 0 || in.AnnualWithdrawal < 0 {
		writeValidationError(w, "amount", "monetary inputs must be non-negative", "")
		return
	}
	if in.Paths < 0 || in.Paths > 10000 {
		writeValidationError(w, "paths", "paths must be in [0, 10000]", "")
		return
	}

	rng := rand.New(rand.NewPCG(uint64(h.now().UnixNano()), 0xdeadbeef))
	result := RunMonteCarlo(rng, MonteCarloInputs{
		CurrentPortfolio:   float64(in.CurrentPortfolio),
		AnnualContribution: float64(in.AnnualContribution),
		ExpectedReturn:     float64(in.ExpectedReturn),
		Volatility:         float64(in.Volatility),
		CurrentAge:         in.CurrentAge,
		RetirementAge:      in.RetirementAge,
		LifeExpectancy:     in.LifeExpectancy,
		AnnualWithdrawal:   float64(in.AnnualWithdrawal),
		Paths:              in.Paths,
	})
	writeJSON(w, http.StatusOK, result)
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

// ownerLabel resolves an owner_user_id to a display name for AccountName.
// A nil id (jointly owned) renders as "Wspólne".
func ownerLabel(names map[int]string, id *int) string {
	if id == nil {
		return "Wspólne"
	}
	if n, ok := names[*id]; ok {
		return n
	}
	return "—"
}

func (h *Handler) buildSimulationResponse(in simulationInputs, sims []AccountSimulation) map[string]any {
	years := in.RetirementAge - in.CurrentAge
	totalFinal, totalContrib, totalReturns := 0.0, 0.0, 0.0
	totalTaxSavings, totalSubsidies := 0.0, 0.0
	for _, s := range sims {
		totalFinal += s.FinalBalance
		totalContrib += s.TotalContributions
		totalReturns += s.TotalReturns
		totalTaxSavings += s.TotalTaxSavings
		totalSubsidies += s.TotalSubsidies
	}
	monthlyIncome := totalFinal * safeWithdrawalRate / 12
	inflationFactor := math.Pow(1+in.InflationRate/100, float64(years))
	monthlyIncomeToday := monthlyIncome / inflationFactor

	simWire := make([]map[string]any, 0, len(sims))
	for i := range sims {
		simWire = append(simWire, accountSimToWire(&sims[i]))
	}
	return map[string]any{
		"inputs":      in.echo(),
		"simulations": simWire,
		"summary": map[string]any{
			"total_final_balance":            pyFloat(totalFinal),
			"total_contributions":            pyFloat(totalContrib),
			"total_returns":                  pyFloat(totalReturns),
			"total_tax_savings":              pyFloat(totalTaxSavings),
			"total_subsidies":                pyFloat(totalSubsidies),
			"estimated_monthly_income":       pyFloat(monthlyIncome),
			"estimated_monthly_income_today": pyFloat(monthlyIncomeToday),
			"years_until_retirement":         years,
		},
	}
}

func accountSimToWire(s *AccountSimulation) map[string]any {
	projections := make([]map[string]any, 0, len(s.YearlyProjections))
	for i := range s.YearlyProjections {
		p := &s.YearlyProjections[i]
		row := map[string]any{
			"year":                     p.Year,
			"age":                      p.Age,
			"annual_contribution":      pyFloat(p.AnnualContribution),
			"balance_end_of_year":      pyFloat(p.BalanceEndOfYear),
			"cumulative_contributions": pyFloat(p.CumulativeContributions),
			"cumulative_returns":       pyFloat(p.CumulativeReturns),
			"annual_limit":             pyFloat(p.AnnualLimit),
			"limit_utilized_pct":       pyFloat(p.LimitUtilizedPct),
			"tax_savings":              pyFloat(p.TaxSavings),
			"government_subsidies":     pyFloat(p.GovernmentSubsidies),
			"monthly_salary":           nil,
			"return_rate":              nil,
		}
		if p.MonthlySalary != nil {
			row["monthly_salary"] = pyFloat(*p.MonthlySalary)
		}
		if p.ReturnRate != nil {
			row["return_rate"] = pyFloat(*p.ReturnRate)
		}
		projections = append(projections, row)
	}
	return map[string]any{
		"account_name":        s.AccountName,
		"starting_balance":    pyFloat(s.StartingBalance),
		"total_contributions": pyFloat(s.TotalContributions),
		"total_returns":       pyFloat(s.TotalReturns),
		"total_tax_savings":   pyFloat(s.TotalTaxSavings),
		"total_subsidies":     pyFloat(s.TotalSubsidies),
		"final_balance":       pyFloat(s.FinalBalance),
		"yearly_projections":  projections,
	}
}

// --- shared response helpers ---

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode response", "err", err, "status", status)
	}
}

func writeDetailError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

func writeValidationError(w http.ResponseWriter, field, msg, input string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"detail": []map[string]any{
			{"type": "value_error", "loc": []string{"body", field}, "msg": msg, "input": input},
		},
	})
}

func writePydanticError(w http.ResponseWriter, vErr *validationError) {
	writeValidationError(w, vErr.Field, vErr.Msg, "")
}

type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}

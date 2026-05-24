package simulations

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Automaat/finance-buddy/backend-go/internal/rules"
)

type validationError struct {
	Field string
	Msg   string
}

// ppkBelowThresholdSalary is the wage cap (1.2× minimum wage) below which a
// PPK participant may lower their employee contribution from 2% to 0.5%.
// Centralized via the rules package (#545) so the validator + frontend +
// error message all read the same number.
var ppkBelowThresholdSalary = rules.MustFloat64("ppk_below_threshold_2026")

// --- mortgage ---

func buildMortgageInputs(raw map[string]json.RawMessage) (MortgageInputs, *validationError) {
	var in MortgageInputs
	var vErr *validationError

	in.RemainingPrincipal, vErr = requireFloat(raw, "remaining_principal")
	if vErr != nil {
		return in, vErr
	}
	if in.RemainingPrincipal <= 0 {
		return in, &validationError{Field: "remaining_principal", Msg: "Remaining principal must be positive"}
	}
	in.AnnualInterestRate, vErr = requireFloat(raw, "annual_interest_rate")
	if vErr != nil {
		return in, vErr
	}
	if in.AnnualInterestRate <= 0 || in.AnnualInterestRate > 30 {
		return in, &validationError{Field: "annual_interest_rate", Msg: "Annual interest rate must be between 0 and 30%"}
	}
	in.RemainingMonths, vErr = requireInt(raw, "remaining_months")
	if vErr != nil {
		return in, vErr
	}
	if in.RemainingMonths < 1 || in.RemainingMonths > 600 {
		return in, &validationError{Field: "remaining_months", Msg: "Remaining months must be between 1 and 600"}
	}
	in.TotalMonthlyBudget, vErr = requireFloat(raw, "total_monthly_budget")
	if vErr != nil {
		return in, vErr
	}
	if in.TotalMonthlyBudget <= 0 {
		return in, &validationError{Field: "total_monthly_budget", Msg: "Total monthly budget must be positive"}
	}
	in.ExpectedAnnualReturn, vErr = requireFloat(raw, "expected_annual_return")
	if vErr != nil {
		return in, vErr
	}
	if in.ExpectedAnnualReturn < 0 || in.ExpectedAnnualReturn > 50 {
		return in, &validationError{Field: "expected_annual_return", Msg: "Expected annual return must be between 0 and 50%"}
	}
	in.InflationRate = 3.0
	if v, ok := raw["inflation_rate"]; ok && !isNullRaw(v) {
		f, err := floatFromRaw(v)
		if err != nil {
			return in, &validationError{Field: "inflation_rate", Msg: "must be a number"}
		}
		if f < 0 || f > 20 {
			return in, &validationError{Field: "inflation_rate", Msg: "Inflation rate must be between 0 and 20%"}
		}
		in.InflationRate = f
	}
	if v, ok := raw["enable_variable_rate"]; ok && !isNullRaw(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return in, &validationError{Field: "enable_variable_rate", Msg: "must be a boolean"}
		}
		in.EnableVariableRate = b
	}
	return in, nil
}

// --- WIBOR scenarios ---

func buildWiborInputs(raw map[string]json.RawMessage) (WiborScenarioInputs, *validationError) {
	var in WiborScenarioInputs
	var vErr *validationError

	in.RemainingPrincipal, vErr = requireFloat(raw, "remaining_principal")
	if vErr != nil {
		return in, vErr
	}
	if in.RemainingPrincipal <= 0 {
		return in, &validationError{Field: "remaining_principal", Msg: "Remaining principal must be positive"}
	}
	in.BaseAnnualRate, vErr = requireFloat(raw, "base_annual_rate")
	if vErr != nil {
		return in, vErr
	}
	if in.BaseAnnualRate <= 0 || in.BaseAnnualRate > 30 {
		return in, &validationError{Field: "base_annual_rate", Msg: "Base annual rate must be > 0 and <= 30%"}
	}
	in.RemainingMonths, vErr = requireInt(raw, "remaining_months")
	if vErr != nil {
		return in, vErr
	}
	if in.RemainingMonths < 1 || in.RemainingMonths > 600 {
		return in, &validationError{Field: "remaining_months", Msg: "Remaining months must be between 1 and 600"}
	}
	if v, ok := raw["base_payment"]; ok && !isNullRaw(v) {
		f, err := floatFromRaw(v)
		if err != nil {
			return in, &validationError{Field: "base_payment", Msg: "must be a number"}
		}
		if f < 0 {
			return in, &validationError{Field: "base_payment", Msg: "Base payment cannot be negative"}
		}
		in.BasePayment = f
	}
	return in, nil
}

// --- retirement ---

type ikeIkzeInput struct {
	Enabled             bool
	Wrapper             string
	OwnerUserID         *int
	Balance             float64
	AutoFillLimit       bool
	MonthlyContribution float64
	TaxRate             float64
}

type ppkInput struct {
	OwnerUserID          *int
	Enabled              bool
	StartingBalance      float64
	MonthlyGrossSalary   float64
	EmployeeRate         float64
	EmployerRate         float64
	SalaryBelowThreshold bool
	IncludeWelcomeBonus  bool
	IncludeAnnualSubsidy bool
}

type brokerageInput struct {
	Enabled             bool
	OwnerUserID         *int
	Balance             float64
	MonthlyContribution float64
}

type simulationInputs struct {
	CurrentAge           int
	RetirementAge        int
	IkeIkzeAccounts      []ikeIkzeInput
	PPKAccounts          []ppkInput
	BrokerageAccounts    []brokerageInput
	AnnualReturnRate     float64
	LimitGrowthRate      float64
	ExpectedSalaryGrowth float64
	InflationRate        float64
}

// echo reproduces the inputs block of the response from the typed values —
// money fields go out as pyFloat so a whole-number input like 46100 still
// serializes as 46100.0 (Pydantic float semantics).
func (s simulationInputs) echo() map[string]any {
	ike := make([]map[string]any, 0, len(s.IkeIkzeAccounts))
	for _, a := range s.IkeIkzeAccounts {
		ike = append(ike, map[string]any{
			"enabled":              a.Enabled,
			"wrapper":              a.Wrapper,
			"owner_user_id":        a.OwnerUserID,
			"balance":              pyFloat(a.Balance),
			"auto_fill_limit":      a.AutoFillLimit,
			"monthly_contribution": pyFloat(a.MonthlyContribution),
			"tax_rate":             pyFloat(a.TaxRate),
		})
	}
	ppk := make([]map[string]any, 0, len(s.PPKAccounts))
	for _, a := range s.PPKAccounts {
		ppk = append(ppk, map[string]any{
			"owner_user_id":          a.OwnerUserID,
			"enabled":                a.Enabled,
			"starting_balance":       pyFloat(a.StartingBalance),
			"monthly_gross_salary":   pyFloat(a.MonthlyGrossSalary),
			"employee_rate":          pyFloat(a.EmployeeRate),
			"employer_rate":          pyFloat(a.EmployerRate),
			"salary_below_threshold": a.SalaryBelowThreshold,
			"include_welcome_bonus":  a.IncludeWelcomeBonus,
			"include_annual_subsidy": a.IncludeAnnualSubsidy,
		})
	}
	brk := make([]map[string]any, 0, len(s.BrokerageAccounts))
	for _, a := range s.BrokerageAccounts {
		brk = append(brk, map[string]any{
			"enabled":              a.Enabled,
			"owner_user_id":        a.OwnerUserID,
			"balance":              pyFloat(a.Balance),
			"monthly_contribution": pyFloat(a.MonthlyContribution),
		})
	}
	return map[string]any{
		"current_age":            s.CurrentAge,
		"retirement_age":         s.RetirementAge,
		"ike_ikze_accounts":      ike,
		"ppk_accounts":           ppk,
		"brokerage_accounts":     brk,
		"annual_return_rate":     pyFloat(s.AnnualReturnRate),
		"limit_growth_rate":      pyFloat(s.LimitGrowthRate),
		"expected_salary_growth": pyFloat(s.ExpectedSalaryGrowth),
		"inflation_rate":         pyFloat(s.InflationRate),
	}
}

func buildSimulationInputs(raw map[string]json.RawMessage) (simulationInputs, *validationError) {
	var in simulationInputs
	var vErr *validationError

	in.CurrentAge, vErr = requireAge(raw, "current_age")
	if vErr != nil {
		return in, vErr
	}
	in.RetirementAge, vErr = requireAge(raw, "retirement_age")
	if vErr != nil {
		return in, vErr
	}
	if in.RetirementAge <= in.CurrentAge {
		return in, &validationError{Field: "retirement_age", Msg: "Retirement age must be greater than current age"}
	}
	if vErr := parseSimAssumptions(raw, &in); vErr != nil {
		return in, vErr
	}
	if vErr := parseSimAccounts(raw, &in); vErr != nil {
		return in, vErr
	}
	hasEnabled := false
	for _, a := range in.IkeIkzeAccounts {
		hasEnabled = hasEnabled || a.Enabled
	}
	for _, a := range in.PPKAccounts {
		hasEnabled = hasEnabled || a.Enabled
	}
	for _, a := range in.BrokerageAccounts {
		hasEnabled = hasEnabled || a.Enabled
	}
	if !hasEnabled {
		return in, &validationError{Field: "accounts", Msg: "At least one account must be selected for simulation"}
	}
	return in, nil
}

func parseSimAssumptions(raw map[string]json.RawMessage, in *simulationInputs) *validationError {
	in.AnnualReturnRate = 7.0
	in.LimitGrowthRate = 5.0
	in.ExpectedSalaryGrowth = 3.0
	in.InflationRate = 3.0
	rates := []struct {
		key      string
		dest     *float64
		lo, hi   float64
		errField string
	}{
		{"annual_return_rate", &in.AnnualReturnRate, -50, 50, "Return rate must be -50% to 50%"},
		{"limit_growth_rate", &in.LimitGrowthRate, 0, 20, "Limit growth rate must be between 0% and 20%"},
		{"expected_salary_growth", &in.ExpectedSalaryGrowth, -10, 20, "Salary growth must be -10% to 20%"},
		{"inflation_rate", &in.InflationRate, 0, 20, "Inflation rate must be between 0% and 20%"},
	}
	for _, rt := range rates {
		v, ok := raw[rt.key]
		if !ok || isNullRaw(v) {
			continue
		}
		f, err := floatFromRaw(v)
		if err != nil {
			return &validationError{Field: rt.key, Msg: "must be a number"}
		}
		if f < rt.lo || f > rt.hi {
			return &validationError{Field: rt.key, Msg: rt.errField}
		}
		*rt.dest = f
	}
	return nil
}

func parseSimAccounts(raw map[string]json.RawMessage, in *simulationInputs) *validationError {
	if v, ok := raw["ike_ikze_accounts"]; ok && !isNullRaw(v) {
		ike, vErr := parseIkeIkze(v)
		if vErr != nil {
			return vErr
		}
		in.IkeIkzeAccounts = ike
	}
	if v, ok := raw["ppk_accounts"]; ok && !isNullRaw(v) {
		ppk, vErr := parsePPK(v)
		if vErr != nil {
			return vErr
		}
		in.PPKAccounts = ppk
	}
	if v, ok := raw["brokerage_accounts"]; ok && !isNullRaw(v) {
		brk, vErr := parseBrokerage(v)
		if vErr != nil {
			return vErr
		}
		in.BrokerageAccounts = brk
	}
	return nil
}

func parseIkeIkze(raw json.RawMessage) ([]ikeIkzeInput, *validationError) {
	var entries []map[string]any
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, &validationError{Field: "ike_ikze_accounts", Msg: "must be an array"}
	}
	out := make([]ikeIkzeInput, 0, len(entries))
	for _, e := range entries {
		var a ikeIkzeInput
		a.Enabled = boolField(e, "enabled")
		wrapper, vErr := requiredStringField(e, "wrapper")
		if vErr != nil {
			return nil, vErr
		}
		a.Wrapper = wrapper
		owner, vErr := requiredIntOrNullField(e, "owner_user_id")
		if vErr != nil {
			return nil, vErr
		}
		a.OwnerUserID = owner
		a.Balance = floatField(e, "balance")
		a.AutoFillLimit = boolField(e, "auto_fill_limit")
		a.MonthlyContribution = floatField(e, "monthly_contribution")
		a.TaxRate = floatField(e, "tax_rate")
		if a.Balance < 0 {
			return nil, &validationError{Field: "balance", Msg: "Account balance cannot be negative"}
		}
		if a.MonthlyContribution < 0 {
			return nil, &validationError{Field: "monthly_contribution", Msg: "Monthly contribution cannot be negative"}
		}
		if a.TaxRate < 0 || a.TaxRate > 100 {
			return nil, &validationError{Field: "tax_rate", Msg: "Tax rate must be between 0 and 100"}
		}
		out = append(out, a)
	}
	return out, nil
}

func parsePPK(raw json.RawMessage) ([]ppkInput, *validationError) {
	var entries []map[string]any
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, &validationError{Field: "ppk_accounts", Msg: "must be an array"}
	}
	out := make([]ppkInput, 0, len(entries))
	for _, e := range entries {
		var a ppkInput
		owner, vErr := requiredIntOrNullField(e, "owner_user_id")
		if vErr != nil {
			return nil, vErr
		}
		a.OwnerUserID = owner
		a.Enabled = boolField(e, "enabled")
		a.StartingBalance = floatField(e, "starting_balance")
		a.MonthlyGrossSalary = floatField(e, "monthly_gross_salary")
		a.EmployeeRate = floatField(e, "employee_rate")
		a.EmployerRate = floatField(e, "employer_rate")
		a.SalaryBelowThreshold = boolField(e, "salary_below_threshold")
		a.IncludeWelcomeBonus = boolFieldDefault(e, "include_welcome_bonus", true)
		a.IncludeAnnualSubsidy = boolFieldDefault(e, "include_annual_subsidy", true)
		if a.EmployeeRate < 0.5 || a.EmployeeRate > 4.0 {
			return nil, &validationError{Field: "employee_rate", Msg: "Employee rate must be 0.5-4.0%"}
		}
		if a.EmployerRate < 1.5 || a.EmployerRate > 4.0 {
			return nil, &validationError{Field: "employer_rate", Msg: "Employer rate must be 1.5-4.0%"}
		}
		if a.StartingBalance < 0 {
			return nil, &validationError{Field: "starting_balance", Msg: "Starting balance cannot be negative"}
		}
		if a.MonthlyGrossSalary < 0 {
			return nil, &validationError{Field: "monthly_gross_salary", Msg: "Monthly gross salary cannot be negative"}
		}
		if a.SalaryBelowThreshold && a.MonthlyGrossSalary > ppkBelowThresholdSalary {
			return nil, &validationError{
				Field: "salary_below_threshold",
				Msg: fmt.Sprintf(
					"salary_below_threshold cannot be True when monthly_gross_salary exceeds %s PLN.",
					strconv.FormatFloat(ppkBelowThresholdSalary, 'f', -1, 64),
				),
			}
		}
		out = append(out, a)
	}
	return out, nil
}

func parseBrokerage(raw json.RawMessage) ([]brokerageInput, *validationError) {
	var entries []map[string]any
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, &validationError{Field: "brokerage_accounts", Msg: "must be an array"}
	}
	out := make([]brokerageInput, 0, len(entries))
	for _, e := range entries {
		var a brokerageInput
		a.Enabled = boolField(e, "enabled")
		owner, vErr := requiredIntOrNullField(e, "owner_user_id")
		if vErr != nil {
			return nil, vErr
		}
		a.OwnerUserID = owner
		a.Balance = floatField(e, "balance")
		a.MonthlyContribution = floatField(e, "monthly_contribution")
		if a.Balance < 0 || a.MonthlyContribution < 0 {
			return nil, &validationError{Field: "brokerage_accounts", Msg: "Value cannot be negative"}
		}
		out = append(out, a)
	}
	return out, nil
}

// --- raw decoders ---

func requireFloat(raw map[string]json.RawMessage, key string) (float64, *validationError) {
	v, ok := raw[key]
	if !ok || isNullRaw(v) {
		return 0, &validationError{Field: key, Msg: "Field required"}
	}
	f, err := floatFromRaw(v)
	if err != nil {
		return 0, &validationError{Field: key, Msg: "must be a number"}
	}
	return f, nil
}

func requireInt(raw map[string]json.RawMessage, key string) (int, *validationError) {
	v, ok := raw[key]
	if !ok || isNullRaw(v) {
		return 0, &validationError{Field: key, Msg: "Field required"}
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &validationError{Field: key, Msg: "must be an integer"}
	}
	return n, nil
}

func requireAge(raw map[string]json.RawMessage, key string) (int, *validationError) {
	n, vErr := requireInt(raw, key)
	if vErr != nil {
		return 0, vErr
	}
	if n < 18 || n > 120 {
		return 0, &validationError{Field: key, Msg: "Age must be between 18 and 120"}
	}
	return n, nil
}

func floatFromRaw(v json.RawMessage) (float64, error) {
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, err
	}
	return f, nil
}

func isNullRaw(v json.RawMessage) bool {
	return string(v) == "null"
}

// requiredStringField enforces a non-empty string for fields the Pydantic
// schema marks required (wrapper) — missing/wrong-type/empty -> 422.
func requiredStringField(m map[string]any, key string) (string, *validationError) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", &validationError{Field: key, Msg: "Field required"}
	}
	s, ok := v.(string)
	if !ok {
		return "", &validationError{Field: key, Msg: "must be a string"}
	}
	if s == "" {
		return "", &validationError{Field: key, Msg: key + " cannot be empty"}
	}
	return s, nil
}

// requiredIntOrNullField reads an integer key that must be present; an
// explicit null is allowed and yields nil (jointly owned). JSON numbers
// decode into map[string]any as float64.
func requiredIntOrNullField(m map[string]any, key string) (*int, *validationError) {
	v, ok := m[key]
	if !ok {
		return nil, &validationError{Field: key, Msg: "Field required"}
	}
	if v == nil {
		return nil, nil
	}
	f, ok := v.(float64)
	if !ok || f != float64(int(f)) {
		return nil, &validationError{Field: key, Msg: "must be an integer"}
	}
	n := int(f)
	return &n, nil
}

func floatField(m map[string]any, key string) float64 {
	if f, ok := m[key].(float64); ok {
		return f
	}
	return 0
}

func boolField(m map[string]any, key string) bool {
	if b, ok := m[key].(bool); ok {
		return b
	}
	return false
}

func boolFieldDefault(m map[string]any, key string, fallback bool) bool {
	if b, ok := m[key].(bool); ok {
		return b
	}
	return fallback
}

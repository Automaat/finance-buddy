package retirement

import (
	"testing"

	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestComputeYearlyStat_AttributesUntypedRemainderToEmployee(t *testing.T) {
	totals := ContributionTotals{
		Total:    dec("10000"),
		Employee: dec("3000"),
		Employer: dec("2000"),
	}
	got := computeYearlyStat(2025, "IKE", nil, totals, dec("20000"))
	if got.EmployeeContributed != 8000 {
		t.Errorf("EmployeeContributed: want 8000, got %v", got.EmployeeContributed)
	}
	if got.EmployerContributed != 2000 {
		t.Errorf("EmployerContributed: want 2000, got %v", got.EmployerContributed)
	}
}

func TestComputeYearlyStat_CapsRemainingAtZero(t *testing.T) {
	totals := ContributionTotals{Total: dec("25000"), Employee: dec("25000")}
	got := computeYearlyStat(2025, "IKE", nil, totals, dec("20000"))
	if *got.Remaining != 0 {
		t.Errorf("Remaining: want 0 (over-limit), got %v", *got.Remaining)
	}
}

func TestComputeYearlyStat_IsWarningUsesUnroundedPercentage(t *testing.T) {
	// 8995 / 10000 = 89.95% — rounds to 90.0 but must NOT trigger warning.
	totals := ContributionTotals{Total: dec("8995"), Employee: dec("8995")}
	got := computeYearlyStat(2025, "IKE", nil, totals, dec("10000"))
	if got.IsWarning {
		t.Errorf("IsWarning: want false at 89.95%%, got true (rounded pct=%v)", *got.PercentageUsed)
	}
	if *got.PercentageUsed != 90.0 {
		t.Errorf("PercentageUsed: want 90.0 (rounded), got %v", *got.PercentageUsed)
	}
}

func TestComputeYearlyStat_WarningAtNinety(t *testing.T) {
	totals := ContributionTotals{Total: dec("9000"), Employee: dec("9000")}
	got := computeYearlyStat(2025, "IKE", nil, totals, dec("10000"))
	if !got.IsWarning {
		t.Errorf("IsWarning: want true at 90%%, got false")
	}
}

func TestComputeYearlyStat_ZeroLimitGivesZeroPercentage(t *testing.T) {
	got := computeYearlyStat(2025, "IKE", nil, ContributionTotals{}, decimal.Zero)
	if *got.PercentageUsed != 0 {
		t.Errorf("PercentageUsed: want 0 with zero limit, got %v", *got.PercentageUsed)
	}
	if got.IsWarning {
		t.Errorf("IsWarning: want false with zero limit, got true")
	}
}

func TestComputePPKStat_AttributesUntypedRemainderToEmployee(t *testing.T) {
	totals := ContributionTotals{
		Total: dec("1000"), Employee: dec("400"), Employer: dec("300"), Government: dec("100"),
	}
	got := computePPKStat(nil, totals, dec("1100"))
	if got.EmployeeContributed != 600 {
		t.Errorf("EmployeeContributed: want 600 (400 + 200 untyped), got %v", got.EmployeeContributed)
	}
}

func TestComputePPKStat_ROIPositive(t *testing.T) {
	totals := ContributionTotals{Total: dec("1000"), Employee: dec("1000")}
	got := computePPKStat(nil, totals, dec("1100"))
	if got.Returns != 100 {
		t.Errorf("Returns: want 100, got %v", got.Returns)
	}
	if got.ROIPercentage != 10.0 {
		t.Errorf("ROIPercentage: want 10.0, got %v", got.ROIPercentage)
	}
}

func TestComputePPKStat_ZeroContributedAvoidsDivByZero(t *testing.T) {
	got := computePPKStat(nil, ContributionTotals{}, dec("500"))
	if got.ROIPercentage != 0 {
		t.Errorf("ROIPercentage: want 0 with zero contributed, got %v", got.ROIPercentage)
	}
	if got.Returns != 500 {
		t.Errorf("Returns: want 500 (latest - 0), got %v", got.Returns)
	}
}

func TestComputePPKContributionAmounts(t *testing.T) {
	emp, empr := computePPKContributionAmounts(dec("10000"), dec("2"), dec("1.5"))
	if !emp.Equal(dec("200")) {
		t.Errorf("employeeAmt: want 200, got %s", emp)
	}
	if !empr.Equal(dec("150")) {
		t.Errorf("employerAmt: want 150, got %s", empr)
	}
}

func TestComputePPKGenerateResponse_WelcomeAndAnnual(t *testing.T) {
	welcomeID, annualID := 11, 12
	req := generateRequest{OwnerUserID: new(1), Month: 6, Year: 2025}
	subsidy := SubsidyConfig{WelcomeAmount: dec("250"), AnnualAmount: dec("240")}
	result := PPKContributionResult{
		EmployeeID: 1, EmployerID: 2, WelcomeID: &welcomeID, AnnualID: &annualID,
	}
	got := computePPKGenerateResponse(req, dec("5000"), dec("100"), dec("75"), subsidy, result)
	if !got.WelcomeApplied || !got.AnnualApplied {
		t.Errorf("flags: want both true, got welcome=%v annual=%v", got.WelcomeApplied, got.AnnualApplied)
	}
	if got.GovernmentAmount != 490 {
		t.Errorf("GovernmentAmount: want 490 (250+240), got %v", got.GovernmentAmount)
	}
	if got.TotalAmount != 100+75+490 {
		t.Errorf("TotalAmount: want 665, got %v", got.TotalAmount)
	}
	if len(got.TransactionsCreated) != 4 {
		t.Errorf("TransactionsCreated: want 4 ids (emp/empr/welcome/annual), got %v", got.TransactionsCreated)
	}
}

func TestComputePPKGenerateResponse_NoSubsidies(t *testing.T) {
	req := generateRequest{OwnerUserID: new(1), Month: 6, Year: 2025}
	subsidy := SubsidyConfig{WelcomeAmount: dec("250"), AnnualAmount: dec("240")}
	result := PPKContributionResult{EmployeeID: 1, EmployerID: 2}
	got := computePPKGenerateResponse(req, dec("5000"), dec("100"), dec("75"), subsidy, result)
	if got.WelcomeApplied || got.AnnualApplied {
		t.Errorf("flags: want both false, got welcome=%v annual=%v", got.WelcomeApplied, got.AnnualApplied)
	}
	if got.GovernmentAmount != 0 {
		t.Errorf("GovernmentAmount: want 0 (no subsidies), got %v", got.GovernmentAmount)
	}
	if len(got.TransactionsCreated) != 2 {
		t.Errorf("TransactionsCreated: want 2 ids, got %v", got.TransactionsCreated)
	}
}

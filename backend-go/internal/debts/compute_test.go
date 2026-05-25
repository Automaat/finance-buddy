package debts

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func decPtr(s string) *decimal.Decimal {
	d := dec(s)
	return &d
}

func TestComputeMetrics_NoBalanceReturnsZeroPair(t *testing.T) {
	principal, interest := computeMetrics(dec("100000"), nil, dec("5000"))
	if !principal.IsZero() || !interest.IsZero() {
		t.Errorf("want (0,0) when balance nil, got (%s,%s)", principal, interest)
	}
}

func TestComputeMetrics_PrincipalIsInitialMinusBalance(t *testing.T) {
	principal, _ := computeMetrics(dec("100000"), decPtr("80000"), dec("25000"))
	if !principal.Equal(dec("20000")) {
		t.Errorf("principal: want 20000 (100000 - 80000), got %s", principal)
	}
}

func TestComputeMetrics_InterestIsTotalMinusPrincipal(t *testing.T) {
	_, interest := computeMetrics(dec("100000"), decPtr("80000"), dec("25000"))
	if !interest.Equal(dec("5000")) {
		t.Errorf("interest: want 5000 (25000 - 20000), got %s", interest)
	}
}

func TestComputeMetrics_HandlesOverpayment(t *testing.T) {
	// Balance > initial would yield negative principal (refund scenario); the
	// function passes the arithmetic through unchanged so callers see the
	// sign and can surface it.
	principal, interest := computeMetrics(dec("100"), decPtr("120"), dec("10"))
	if !principal.Equal(dec("-20")) {
		t.Errorf("principal: want -20, got %s", principal)
	}
	if !interest.Equal(dec("30")) {
		t.Errorf("interest: want 30 (10 - (-20)), got %s", interest)
	}
}

func TestMetricsFor_MissingAccountUsesZero(t *testing.T) {
	m := metricsFor(99, map[int]LatestBalance{}, map[int]decimal.Decimal{})
	if m.latestBalance != nil || m.latestBalanceDate != nil {
		t.Errorf("want nil balance/date for missing acct, got %+v", m)
	}
	if !m.totalPaid.IsZero() {
		t.Errorf("totalPaid: want zero, got %s", m.totalPaid)
	}
}

func TestMetricsFor_PicksUpBalanceAndDate(t *testing.T) {
	d := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	balances := map[int]LatestBalance{1: {Value: dec("80000"), Date: d}}
	totals := map[int]decimal.Decimal{1: dec("15000")}
	m := metricsFor(1, balances, totals)
	if m.latestBalance == nil || !m.latestBalance.Equal(dec("80000")) {
		t.Errorf("latestBalance: want 80000, got %v", m.latestBalance)
	}
	if m.latestBalanceDate == nil || !m.latestBalanceDate.Equal(d) {
		t.Errorf("latestBalanceDate: want %v, got %v", d, m.latestBalanceDate)
	}
	if !m.totalPaid.Equal(dec("15000")) {
		t.Errorf("totalPaid: want 15000, got %s", m.totalPaid)
	}
}

func TestToResponse_WithoutBalanceLeavesPointersNil(t *testing.T) {
	d := Debt{
		ID: 1, AccountID: 10, Name: "M1", DebtType: "mortgage",
		StartDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		InitialAmount: dec("250000"), InterestRate: dec("6.5"),
		Currency: "PLN", IsActive: true,
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	a := AccountInfo{ID: 10, Name: "Mortgage A"}
	r := toResponse(d, a, debtMetrics{})
	if r.LatestBalance != nil {
		t.Errorf("LatestBalance: want nil, got %v", *r.LatestBalance)
	}
	if r.LatestBalanceDate != nil {
		t.Errorf("LatestBalanceDate: want nil, got %v", *r.LatestBalanceDate)
	}
	if r.TotalPaid != 0 || r.InterestPaid != 0 {
		t.Errorf("totals: want 0,0 got %v,%v", r.TotalPaid, r.InterestPaid)
	}
	if r.Name != "M1" || r.DebtType != "mortgage" {
		t.Errorf("identity copy wrong: %+v", r)
	}
}

func TestToResponse_PopulatesBalancePointersAndInterest(t *testing.T) {
	d := Debt{
		ID: 1, AccountID: 10, InitialAmount: dec("100000"),
		InterestRate: dec("5"), Currency: "PLN",
	}
	a := AccountInfo{ID: 10, Name: "Mortgage A"}
	bal := dec("80000")
	balDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	m := debtMetrics{
		latestBalance:     &bal,
		latestBalanceDate: &balDate,
		totalPaid:         dec("25000"),
	}
	r := toResponse(d, a, m)
	if r.LatestBalance == nil || *r.LatestBalance != 80000 {
		t.Errorf("LatestBalance: want 80000, got %v", r.LatestBalance)
	}
	if r.LatestBalanceDate == nil {
		t.Errorf("LatestBalanceDate: want set")
	}
	// principal = 100000 - 80000 = 20000; interest = 25000 - 20000 = 5000.
	if r.TotalPaid != 25000 {
		t.Errorf("TotalPaid: want 25000, got %v", r.TotalPaid)
	}
	if r.InterestPaid != 5000 {
		t.Errorf("InterestPaid: want 5000, got %v", r.InterestPaid)
	}
}

func TestPyFloatMarshalsIntegerWithTrailingZero(t *testing.T) {
	got, _ := json.Marshal(wire.PyFloat(100))
	if string(got) != "100.0" {
		t.Errorf("want 100.0, got %s", got)
	}
}

func TestIsoDateMarshal(t *testing.T) {
	got, _ := json.Marshal(wire.IsoDate(time.Date(2025, 6, 1, 14, 0, 0, 0, time.UTC)))
	if string(got) != `"2025-06-01"` {
		t.Errorf("want \"2025-06-01\", got %s", got)
	}
}

func TestIsoNaiveMarshalStripsTimezone(t *testing.T) {
	loc := time.FixedZone("CET", 3600)
	got, _ := json.Marshal(wire.IsoNaive(time.Date(2025, 1, 5, 14, 30, 0, 0, loc)))
	body := strings.TrimSuffix(strings.TrimPrefix(string(got), `"`), `"`)
	timePart := body[strings.Index(body, "T")+1:]
	if strings.ContainsAny(timePart, "Z+-") {
		t.Errorf("naive time must drop tz marker, got %s", got)
	}
}

package debts

import (
	"time"

	"github.com/shopspring/decimal"
)

// debtTypeToCategory maps a debt_type to the liability account category
// used when /api/debts creates the account in the same transaction.
var debtTypeToCategory = map[string]string{
	"mortgage":             "mortgage",
	"installment_0percent": "installment",
}

// debtMetrics is the per-debt aggregate that the response carries: the
// latest snapshot balance, its date, and the running total of payments.
type debtMetrics struct {
	latestBalance     *decimal.Decimal
	latestBalanceDate *time.Time
	totalPaid         decimal.Decimal
}

// computeMetrics returns (principalPaid, interestPaid) given the original
// loan amount, the latest balance (nil when no snapshot exists), and the
// running total of payments. With no balance the function returns zero
// pairs — there is no anchor to deduce principal from.
func computeMetrics(initial decimal.Decimal, balance *decimal.Decimal, totalPaid decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if balance == nil {
		return decimal.Zero, decimal.Zero
	}
	principalPaid := initial.Sub(*balance)
	return principalPaid, totalPaid.Sub(principalPaid)
}

// metricsFor builds a debtMetrics for one accountID from the bulk maps the
// list endpoint loads in two queries. Missing entries are treated as
// "no snapshot, zero paid".
func metricsFor(accountID int, balances map[int]LatestBalance, totals map[int]decimal.Decimal) debtMetrics {
	m := debtMetrics{totalPaid: totals[accountID]}
	if lb, ok := balances[accountID]; ok {
		balance := lb.Value
		date := lb.Date
		m.latestBalance = &balance
		m.latestBalanceDate = &date
	}
	return m
}

// toResponse renders the wire-shape response from the domain Debt, parent
// account info, and the computed metrics.
func toResponse(d Debt, a AccountInfo, m debtMetrics) response {
	initial, _ := d.InitialAmount.Float64()
	rate, _ := d.InterestRate.Float64()
	totalPaid, _ := m.totalPaid.Float64()
	_, interestPaid := computeMetrics(d.InitialAmount, m.latestBalance, m.totalPaid)
	ip, _ := interestPaid.Float64()
	out := response{
		ID: d.ID, AccountID: d.AccountID, AccountName: a.Name, AccountOwnerUserID: a.OwnerUserID,
		Name: d.Name, DebtType: d.DebtType, StartDate: isoDate(d.StartDate),
		InitialAmount: pyFloat(initial), InterestRate: pyFloat(rate),
		Currency: d.Currency, Notes: d.Notes, IsActive: d.IsActive,
		CreatedAt: isoNaive(d.CreatedAt.UTC()),
		TotalPaid: pyFloat(totalPaid), InterestPaid: pyFloat(ip),
	}
	if m.latestBalance != nil {
		f, _ := m.latestBalance.Float64()
		pf := pyFloat(f)
		out.LatestBalance = &pf
	}
	if m.latestBalanceDate != nil {
		d := isoDate(*m.latestBalanceDate)
		out.LatestBalanceDate = &d
	}
	return out
}

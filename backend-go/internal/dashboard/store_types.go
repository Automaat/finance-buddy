package dashboard

import (
	"time"

	"github.com/shopspring/decimal"
)

// AggregateRow mirrors a snapshot_aggregates row. OwnerUserID is nil for
// the jointly-owned ("Shared") bucket.
type AggregateRow struct {
	SnapshotID       int
	Month            time.Time
	OwnerUserID      *int
	TotalAssets      decimal.Decimal
	TotalLiabilities decimal.Decimal
	NetWorth         decimal.Decimal
	Allocation       []AllocationJSONItem
}

// AllocationJSONItem is one entry of allocation_json.
type AllocationJSONItem struct {
	Category string  `json:"category"`
	Value    float64 `json:"value"`
}

// Account mirrors the columns the dashboard reads from accounts.
// OwnerUserID is nil for jointly-owned accounts. IsActive lets the
// dashboard preserve historical snapshot values from soft-deleted
// accounts while keeping "current state" reads filtered.
type Account struct {
	ID               int
	Name             string
	Type             string
	Category         string
	OwnerUserID      *int
	AccountWrapper   *string
	Purpose          string
	SquareMeters     *decimal.Decimal
	IsActive         bool
	ExcludedFromFire bool
}

// SnapshotValue is one snapshot_values row.
type SnapshotValue struct {
	SnapshotID int
	AccountID  *int
	AssetID    *int
	Value      decimal.Decimal
}

// Transaction is the subset of transactions the dashboard reads.
type Transaction struct {
	AccountID       int
	Amount          decimal.Decimal
	Date            time.Time
	TransactionType *string
}

// AppConfig mirrors the app_config row.
type AppConfig struct {
	MonthlyExpenses        decimal.Decimal
	MonthlyMortgagePayment decimal.Decimal
	AllocationStocks       int
	AllocationBonds        int
	AllocationGold         int
	WithdrawalRate         decimal.Decimal
	BirthDate              time.Time
	CoastFIRETargetAge     *int
	ExpectedReturnRate     decimal.Decimal
	BaristaMonthlyIncome   *decimal.Decimal
	LeanMonthlyExpenses    *decimal.Decimal
	FatMonthlyExpenses     *decimal.Decimal
	MonthlySavings         *decimal.Decimal
}

// SnapshotMeta is id+date.
type SnapshotMeta struct {
	ID   int
	Date time.Time
}

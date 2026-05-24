// Package recurring implements scheduled / recurring transactions —
// templates that generate concrete `transactions` rows on a cadence
// (monthly salary, mortgage payment, subscriptions). Subissue of #355.
//
// The scheduler in internal/scheduler runs ProcessDue daily; the API in
// handler.go exposes CRUD, manual "run now", and per-occurrence skip.
package recurring

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// Frequency is the recurrence cadence.
type Frequency string

const (
	FrequencyDaily     Frequency = "daily"
	FrequencyWeekly    Frequency = "weekly"
	FrequencyMonthly   Frequency = "monthly"
	FrequencyQuarterly Frequency = "quarterly"
	FrequencyYearly    Frequency = "yearly"
)

var validFrequencies = []Frequency{
	FrequencyDaily, FrequencyWeekly, FrequencyMonthly, FrequencyQuarterly, FrequencyYearly,
}

// IsValidFrequency reports whether s is a known cadence.
func IsValidFrequency(s string) bool {
	for _, f := range validFrequencies {
		if string(f) == s {
			return true
		}
	}
	return false
}

// ValidFrequencies returns a fresh copy of the canonical cadence list.
func ValidFrequencies() []Frequency {
	out := make([]Frequency, len(validFrequencies))
	copy(out, validFrequencies)
	return out
}

// Recurring is a recurring-transaction template row.
type Recurring struct {
	ID              int
	AccountID       int
	Amount          decimal.Decimal
	OwnerUserID     *int
	TransactionType *string
	Category        *string
	Description     string
	Frequency       Frequency
	DayOfMonth      *int
	StartDate       time.Time
	EndDate         *time.Time
	Active          bool
	SkippedDates    []time.Time
	LastRunDate     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Sentinel errors mapped to HTTP statuses by the handler.
var (
	ErrNotFound        = errors.New("recurring transaction not found")
	ErrAccountNotFound = errors.New("account not found")
)

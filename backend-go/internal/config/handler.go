package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// response mirrors backend/app/schemas/config.ConfigResponse byte-for-byte.
//
// Date is serialized as "YYYY-MM-DD". Decimal money fields are quoted JSON
// strings with two decimals (Python Pydantic v2's default for Decimal).
type response struct {
	ID                      int       `json:"id"`
	BirthDate               isoDate   `json:"birth_date"`
	RetirementAge           int       `json:"retirement_age"`
	RetirementMonthlySalary moneyJSON `json:"retirement_monthly_salary"`
	AllocationRealEstate    int       `json:"allocation_real_estate"`
	AllocationStocks        int       `json:"allocation_stocks"`
	AllocationBonds         int       `json:"allocation_bonds"`
	AllocationGold          int       `json:"allocation_gold"`
	AllocationCommodities   int       `json:"allocation_commodities"`
	MonthlyExpenses         moneyJSON `json:"monthly_expenses"`
	MonthlyMortgagePayment  moneyJSON `json:"monthly_mortgage_payment"`
	WithdrawalRate          rateJSON  `json:"withdrawal_rate"`
}

// request is the PUT body. Date arrives as "YYYY-MM-DD" and money as either
// a JSON number or string — pydantic on the Python side accepts both.
type request struct {
	BirthDate               isoDate          `json:"birth_date"`
	RetirementAge           int              `json:"retirement_age"`
	RetirementMonthlySalary decimal.Decimal  `json:"retirement_monthly_salary"`
	AllocationRealEstate    int              `json:"allocation_real_estate"`
	AllocationStocks        int              `json:"allocation_stocks"`
	AllocationBonds         int              `json:"allocation_bonds"`
	AllocationGold          int              `json:"allocation_gold"`
	AllocationCommodities   int              `json:"allocation_commodities"`
	MonthlyExpenses         decimal.Decimal  `json:"monthly_expenses"`
	MonthlyMortgagePayment  decimal.Decimal  `json:"monthly_mortgage_payment"`
	WithdrawalRate          *decimal.Decimal `json:"withdrawal_rate"`
}

func toResponse(c *Config) response {
	return response{
		ID:                      c.ID,
		BirthDate:               isoDate(c.BirthDate),
		RetirementAge:           c.RetirementAge,
		RetirementMonthlySalary: moneyJSON(c.RetirementMonthlySalary),
		AllocationRealEstate:    c.AllocationRealEstate,
		AllocationStocks:        c.AllocationStocks,
		AllocationBonds:         c.AllocationBonds,
		AllocationGold:          c.AllocationGold,
		AllocationCommodities:   c.AllocationCommodities,
		MonthlyExpenses:         moneyJSON(c.MonthlyExpenses),
		MonthlyMortgagePayment:  moneyJSON(c.MonthlyMortgagePayment),
		WithdrawalRate:          rateJSON(c.WithdrawalRate),
	}
}

// Handler returns the chi-compatible HTTP handler for GET + PUT /api/config.
//
// Wire format and validation rules match
// backend/app/schemas/config.{ConfigCreate,ConfigResponse} so the parity
// suite (backend-bb-tests/tests/test_config.py) is the contract.
type Handler struct {
	store  *Store
	logger *slog.Logger
}

// NewHandler wires the store and logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger}
}

// Get serves GET /api/config.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	c, err := h.store.Get(r.Context())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeDetailError(w, http.StatusNotFound, "Configuration not initialized")
			return
		}
		h.logger.Error("config get", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toResponse(c))
}

// Put serves PUT /api/config.
func (h *Handler) Put(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	if vErr := req.validate(); vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	c, err := h.store.Upsert(r.Context(), req.toConfig())
	if err != nil {
		h.logger.Error("config upsert", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toResponse(c))
}

func (r *request) toConfig() *Config {
	rate := defaultWithdrawalRate
	if r.WithdrawalRate != nil {
		rate = *r.WithdrawalRate
	}
	return &Config{
		BirthDate:               time.Time(r.BirthDate),
		RetirementAge:           r.RetirementAge,
		RetirementMonthlySalary: r.RetirementMonthlySalary,
		AllocationRealEstate:    r.AllocationRealEstate,
		AllocationStocks:        r.AllocationStocks,
		AllocationBonds:         r.AllocationBonds,
		AllocationGold:          r.AllocationGold,
		AllocationCommodities:   r.AllocationCommodities,
		MonthlyExpenses:         r.MonthlyExpenses,
		MonthlyMortgagePayment:  r.MonthlyMortgagePayment,
		WithdrawalRate:          rate,
	}
}

// defaultWithdrawalRate is the 4% Trinity-study safe-withdrawal default
// — used when a PUT body omits withdrawal_rate (older clients).
// RequireFromString keeps the constant exact at the numeric(5,4) precision
// the DB column expects.
var defaultWithdrawalRate = decimal.RequireFromString("0.04")

// isoDate is a time.Time alias that JSON-marshals as "YYYY-MM-DD" and
// unmarshals from the same. Matches Python's `date` field on the wire.
type isoDate time.Time

const isoDateLayout = "2006-01-02"

func (d isoDate) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("%q", time.Time(d).Format(isoDateLayout))
	return []byte(stamp), nil
}

func (d *isoDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("birth_date must be a YYYY-MM-DD string: %w", err)
	}
	t, err := time.Parse(isoDateLayout, s)
	if err != nil {
		return fmt.Errorf("birth_date must be YYYY-MM-DD: %w", err)
	}
	*d = isoDate(t)
	return nil
}

// moneyJSON is a decimal.Decimal that marshals as a JSON string with two
// decimals — matches Pydantic v2's default Decimal serialization for
// Numeric(15,2) columns.
type moneyJSON decimal.Decimal

func (m moneyJSON) MarshalJSON() ([]byte, error) {
	d := decimal.Decimal(m)
	return []byte(`"` + d.StringFixed(2) + `"`), nil
}

// rateJSON serializes withdrawal_rate as a 4-decimal JSON string — matches
// the numeric(5,4) DB column (e.g. "0.0400" for the 4% default).
type rateJSON decimal.Decimal

func (r rateJSON) MarshalJSON() ([]byte, error) {
	d := decimal.Decimal(r)
	return []byte(`"` + d.StringFixed(4) + `"`), nil
}

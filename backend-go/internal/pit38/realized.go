// Package pit38 computes the PIT-38 helper: per-sale realized gain/loss for
// brokerage lots, with NBP FX conversion when the security trades in a
// foreign currency. Output is intended as a worksheet for the annual
// Polish capital-gains return — not a filed document.
package pit38

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
	"github.com/Automaat/finance-buddy/backend-go/internal/holdings"
)

// isoDate marshals as a date-only YYYY-MM-DD string, matching the wire
// format every other endpoint uses for transaction dates.
type isoDate time.Time

// MarshalJSON renders the underlying time as YYYY-MM-DD.
func (d isoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format("2006-01-02") + `"`), nil
}

// SaleRow is one realized-sale row in the report. All `_PLN` fields are
// after NBP conversion at the per-row tax date (sale date for proceeds,
// running PLN-weighted cost basis for cost). Currency-side fields are
// kept too so users can reconcile against broker statements.
type SaleRow struct {
	SecurityID   int             `json:"security_id"`
	Symbol       string          `json:"symbol"`
	Currency     string          `json:"currency"`
	Date         isoDate         `json:"date"`
	Quantity     decimal.Decimal `json:"quantity"`
	Proceeds     decimal.Decimal `json:"proceeds"`
	CostBasis    decimal.Decimal `json:"cost_basis"`
	Fees         decimal.Decimal `json:"fees"`
	RealizedGain decimal.Decimal `json:"realized_gain"`
	FXRate       decimal.Decimal `json:"fx_rate"`
	HasFX        bool            `json:"has_fx"`
	ProceedsPLN  decimal.Decimal `json:"proceeds_pln"`
	CostBasisPLN decimal.Decimal `json:"cost_basis_pln"`
	FeesPLN      decimal.Decimal `json:"fees_pln"`
	RealizedPLN  decimal.Decimal `json:"realized_pln"`
}

// Totals sums the per-row PLN figures for the report footer.
type Totals struct {
	ProceedsPLN  decimal.Decimal `json:"proceeds_pln"`
	CostBasisPLN decimal.Decimal `json:"cost_basis_pln"`
	FeesPLN      decimal.Decimal `json:"fees_pln"`
	RealizedPLN  decimal.Decimal `json:"realized_pln"`
}

// Report is the full PIT-38 helper payload.
type Report struct {
	Year   int       `json:"year"`
	Rows   []SaleRow `json:"rows"`
	Totals Totals    `json:"totals"`
}

// RateProvider abstracts NBP rate lookups so the algorithm stays a pure
// function and is testable without a Postgres / network roundtrip.
type RateProvider interface {
	GetRateToPLN(ctx context.Context, currency string, onDate time.Time) (fx.Result, error)
}

// ErrUnknownCurrency surfaces when NBP couldn't supply a rate for a
// foreign-currency lot. The caller decides whether to drop the row or
// surface a 5xx — the helper itself doesn't apply policy.
var ErrUnknownCurrency = errors.New("pit38: no FX rate available for currency on date")

// SecurityLots groups the chronologically-sorted lot history for one
// security so ComputeReport can walk it linearly.
type SecurityLots struct {
	SecurityID int
	Symbol     string
	Currency   string
	Lots       []holdings.Lot
}

// ComputeReport produces the PIT-38 helper for the supplied year. It
// replays each security's full lot history (not just the year's), then
// keeps only sales whose date falls inside the year — earlier lots are
// needed to derive the running weighted-average cost basis at the time
// of sale, even when the sale itself isn't in scope.
func ComputeReport(ctx context.Context, year int, rates RateProvider, byses []SecurityLots) (Report, error) {
	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	rep := Report{Year: year, Rows: []SaleRow{}}
	for i := range byses {
		rows, err := realizedSalesFor(ctx, &byses[i], yearStart, yearEnd, rates)
		if err != nil {
			return Report{}, err
		}
		rep.Rows = append(rep.Rows, rows...)
	}
	sort.SliceStable(rep.Rows, func(i, j int) bool {
		di := time.Time(rep.Rows[i].Date)
		dj := time.Time(rep.Rows[j].Date)
		if !di.Equal(dj) {
			return di.Before(dj)
		}
		return rep.Rows[i].Symbol < rep.Rows[j].Symbol
	})
	for i := range rep.Rows {
		row := &rep.Rows[i]
		rep.Totals.ProceedsPLN = rep.Totals.ProceedsPLN.Add(row.ProceedsPLN)
		rep.Totals.CostBasisPLN = rep.Totals.CostBasisPLN.Add(row.CostBasisPLN)
		rep.Totals.FeesPLN = rep.Totals.FeesPLN.Add(row.FeesPLN)
		rep.Totals.RealizedPLN = rep.Totals.RealizedPLN.Add(row.RealizedPLN)
	}
	return rep, nil
}

// realizedSalesFor walks the lot history for one security, maintaining
// both the security-currency weighted-average cost and a parallel PLN
// weighted-average cost (each buy converted at its trade-date NBP rate).
// Each sell inside the report window emits one SaleRow.
func realizedSalesFor(
	ctx context.Context,
	sec *SecurityLots,
	from, to time.Time,
	rates RateProvider,
) ([]SaleRow, error) {
	lots := make([]holdings.Lot, len(sec.Lots))
	copy(lots, sec.Lots)
	sort.SliceStable(lots, func(i, j int) bool {
		if !lots[i].Date.Equal(lots[j].Date) {
			return lots[i].Date.Before(lots[j].Date)
		}
		return lots[i].ID < lots[j].ID
	})
	var qty, avgCcy, avgPLN decimal.Decimal
	out := []SaleRow{}
	for i := range lots {
		l := &lots[i]
		rate, err := rates.GetRateToPLN(ctx, sec.Currency, l.Date)
		if err != nil {
			return nil, err
		}
		switch l.Side {
		case holdings.SideBuy:
			lotCost := l.Quantity.Mul(l.Price).Add(l.Fee)
			lotCostPLN, ok := toPLN(lotCost, sec.Currency, rate)
			if !ok {
				return nil, ErrUnknownCurrency
			}
			newQty := qty.Add(l.Quantity)
			if newQty.IsZero() {
				avgCcy = decimal.Zero
				avgPLN = decimal.Zero
			} else {
				prevCcy := qty.Mul(avgCcy)
				prevPLN := qty.Mul(avgPLN)
				avgCcy = prevCcy.Add(lotCost).Div(newQty)
				avgPLN = prevPLN.Add(lotCostPLN).Div(newQty)
			}
			qty = newQty
		case holdings.SideSell:
			if l.Quantity.GreaterThan(qty) {
				return nil, holdings.ErrOversell
			}
			proceedsCcy := l.Quantity.Mul(l.Price).Sub(l.Fee)
			costBasisCcy := l.Quantity.Mul(avgCcy)
			feesCcy := l.Fee
			gainCcy := proceedsCcy.Sub(costBasisCcy)
			proceedsPLN, ok := toPLN(proceedsCcy, sec.Currency, rate)
			if !ok {
				return nil, ErrUnknownCurrency
			}
			feesPLN, _ := toPLN(feesCcy, sec.Currency, rate)
			costPLN := l.Quantity.Mul(avgPLN)
			gainPLN := proceedsPLN.Sub(costPLN)
			qty = qty.Sub(l.Quantity)
			if qty.IsZero() {
				avgCcy = decimal.Zero
				avgPLN = decimal.Zero
			}
			if !l.Date.Before(from) && l.Date.Before(to) {
				out = append(out, SaleRow{
					SecurityID:   sec.SecurityID,
					Symbol:       sec.Symbol,
					Currency:     sec.Currency,
					Date:         isoDate(l.Date),
					Quantity:     l.Quantity,
					Proceeds:     proceedsCcy,
					CostBasis:    costBasisCcy,
					Fees:         feesCcy,
					RealizedGain: gainCcy,
					FXRate:       rate.Rate,
					HasFX:        sec.Currency != "PLN",
					ProceedsPLN:  proceedsPLN,
					CostBasisPLN: costPLN,
					FeesPLN:      feesPLN,
					RealizedPLN:  gainPLN,
				})
			}
		}
	}
	return out, nil
}

// toPLN wraps fx.ToPLN so the algorithm doesn't need a pointer dance for
// every conversion.
func toPLN(amount decimal.Decimal, currency string, rate fx.Result) (decimal.Decimal, bool) {
	return fx.ToPLN(&amount, currency, rate)
}

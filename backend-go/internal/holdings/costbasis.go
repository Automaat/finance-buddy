// Package holdings implements per-security holdings tracking — buy/sell lots,
// weighted-average cost basis, realized/unrealized gain, and manual price
// quotes. Issue #400.
//
// Cost-basis convention: weighted average. For every buy, the average cost
// rolls forward including the fee; for every sell, realized P&L is taken
// against the running average and the fee subtracts from realized gain. This
// matches the most common Polish broker reporting model and the IRS
// "average cost" basis for funds. FIFO/LIFO can be added later behind a
// per-account flag without changing the storage model.
package holdings

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// Side discriminates a Lot row.
type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

// IsValidSide reports whether s is "buy" or "sell".
func IsValidSide(s string) bool {
	return s == string(SideBuy) || s == string(SideSell)
}

// Lot is one buy or sell of a security inside an account. The store reads
// rows in chronological order and walks them through ComputeRunning to
// produce per-security holdings.
type Lot struct {
	ID         int
	AccountID  int
	SecurityID int
	Side       Side
	Quantity   decimal.Decimal // always positive on the lot row; Side carries the sign
	Price      decimal.Decimal // per-unit
	Fee        decimal.Decimal // total fee for the lot, in account currency
	Date       time.Time
	CreatedAt  time.Time
}

// Running is the running cost-basis state for one security after a sequence
// of lots has been replayed.
type Running struct {
	Quantity     decimal.Decimal
	AverageCost  decimal.Decimal // per unit, weighted average
	CostBasis    decimal.Decimal // Quantity * AverageCost (cached so callers don't need decimal math)
	RealizedGain decimal.Decimal
	TotalBought  decimal.Decimal // gross deposits — for return-attribution
	TotalSold    decimal.Decimal // gross proceeds — same
	FeesPaid     decimal.Decimal
}

// ErrOversell is returned by ComputeRunning when a sell lot exceeds the
// current quantity held.
var ErrOversell = errors.New("holdings: sell exceeds quantity held")

// ComputeRunning replays a chronologically sorted lot history and returns the
// final cost-basis state. Pure function — same input gives same output.
//
// Buy:
//
//	new_qty       = qty + lot.qty
//	new_avg_cost  = (qty*avg_cost + lot.qty*lot.price + lot.fee) / new_qty
//	cost_basis    = new_qty * new_avg_cost
//	total_bought += lot.qty*lot.price + lot.fee
//
// Sell:
//
//	realized_gain += (lot.price - avg_cost) * lot.qty - lot.fee
//	qty           -= lot.qty                 ; avg_cost unchanged
//	cost_basis     = qty * avg_cost
//	total_sold    += lot.qty*lot.price - lot.fee
func ComputeRunning(lots []Lot) (Running, error) {
	var r Running
	for i := range lots {
		l := &lots[i]
		switch l.Side {
		case SideBuy:
			lotCost := l.Quantity.Mul(l.Price).Add(l.Fee)
			newQty := r.Quantity.Add(l.Quantity)
			if newQty.IsZero() {
				r.AverageCost = decimal.Zero
			} else {
				existing := r.Quantity.Mul(r.AverageCost)
				r.AverageCost = existing.Add(lotCost).Div(newQty)
			}
			r.Quantity = newQty
			r.TotalBought = r.TotalBought.Add(lotCost)
			r.FeesPaid = r.FeesPaid.Add(l.Fee)
		case SideSell:
			if l.Quantity.GreaterThan(r.Quantity) {
				return Running{}, ErrOversell
			}
			gain := l.Price.Sub(r.AverageCost).Mul(l.Quantity).Sub(l.Fee)
			r.RealizedGain = r.RealizedGain.Add(gain)
			proceeds := l.Quantity.Mul(l.Price).Sub(l.Fee)
			r.TotalSold = r.TotalSold.Add(proceeds)
			r.Quantity = r.Quantity.Sub(l.Quantity)
			r.FeesPaid = r.FeesPaid.Add(l.Fee)
			if r.Quantity.IsZero() {
				r.AverageCost = decimal.Zero
			}
		}
	}
	r.CostBasis = r.Quantity.Mul(r.AverageCost)
	return r, nil
}

// UnrealizedGain reports the open-position gain at price `quote`. Returns
// zero when no quote is supplied (Decimal.IsZero()).
func UnrealizedGain(r Running, quote decimal.Decimal) decimal.Decimal {
	if quote.IsZero() {
		return decimal.Zero
	}
	return quote.Sub(r.AverageCost).Mul(r.Quantity)
}

// MarketValue is Quantity * latest quote, or CostBasis when no quote.
func MarketValue(r Running, quote decimal.Decimal) decimal.Decimal {
	if quote.IsZero() {
		return r.CostBasis
	}
	return r.Quantity.Mul(quote)
}

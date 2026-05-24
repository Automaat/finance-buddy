package investment

import (
	"errors"
	"math"
	"time"
)

// CashFlow is one dated movement in the XIRR series.
//
// Convention used throughout this package:
//   - positive amount = deposit (money in to the account)
//   - negative amount = withdrawal / distribution (money out of the account)
//   - the final, dated `current_value` is appended as a *negative* terminal
//     flow, mirroring closing the position.
//
// This matches the Excel XIRR convention and keeps the rate sign intuitive:
// a positive XIRR means the contributions grew, a negative XIRR means they
// shrank.
type CashFlow struct {
	Date   time.Time
	Amount float64
}

// ErrXIRRNoConverge is returned when Newton's method fails to converge
// within xirrMaxIterations. The handler swallows it and reports return=null
// rather than guessing.
var ErrXIRRNoConverge = errors.New("xirr: did not converge")

const (
	xirrMaxIterations = 100
	xirrTolerance     = 1e-7
)

// XIRR returns the annualized money-weighted rate of return for the given
// dated cash flows. Inputs must contain at least one positive and one
// negative flow (otherwise no rate exists that can balance NPV at zero).
//
// Algorithm: Newton-Raphson on r in NPV(r) = 0, starting at r=0.1 and falling
// back to a 0.5x dampened step when an iteration would push r below -100%
// (which would make the discount factor undefined for any date after t0).
func XIRR(flows []CashFlow) (float64, error) {
	if len(flows) < 2 {
		return 0, ErrXIRRNoConverge
	}
	hasPos, hasNeg := false, false
	for _, f := range flows {
		if f.Amount > 0 {
			hasPos = true
		} else if f.Amount < 0 {
			hasNeg = true
		}
	}
	if !hasPos || !hasNeg {
		return 0, ErrXIRRNoConverge
	}
	t0 := flows[0].Date
	for _, f := range flows {
		if f.Date.Before(t0) {
			t0 = f.Date
		}
	}
	rate := 0.1
	for range xirrMaxIterations {
		npv, dnpv := 0.0, 0.0
		for _, f := range flows {
			years := f.Date.Sub(t0).Hours() / 24 / 365.0
			base := 1 + rate
			if base <= 0 {
				return 0, ErrXIRRNoConverge
			}
			disc := math.Pow(base, years)
			npv += f.Amount / disc
			dnpv -= years * f.Amount / (disc * base)
		}
		if math.Abs(dnpv) < 1e-12 {
			return 0, ErrXIRRNoConverge
		}
		newRate := rate - npv/dnpv
		if newRate <= -1 {
			newRate = (rate - 1) / 2 // dampen toward the wall
		}
		if math.Abs(newRate-rate) < xirrTolerance {
			return newRate, nil
		}
		rate = newRate
	}
	return 0, ErrXIRRNoConverge
}

// SimpleROI is the contribution-naive return shown by the existing investment
// dashboard — (value - contributed) / contributed. Surfaced alongside XIRR so
// the UI can show the difference between "I added X and now have Y" and "Y/X
// annualized over the actual time it took".
func SimpleROI(contributed, value float64) float64 {
	if contributed <= 0 {
		return 0
	}
	return (value - contributed) / contributed
}

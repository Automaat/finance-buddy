package equitygrants

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	companyvaluations "github.com/Automaat/finance-buddy/backend-go/internal/company_valuations"
	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
)

// paperValueResult bundles the per-share-vested values (base / low / high)
// plus the valuation metadata that drove them. Mirrors the tuple Python
// returns from _paper_values.
type paperValueResult struct {
	Base            *decimal.Decimal
	Low             *decimal.Decimal
	High            *decimal.Decimal
	Currency        string
	ValuationDate   *time.Time
	ValuationSource string
}

// computePaperValues replicates backend/app/services/equity_grants._paper_values.
//
// Only computes when grant and valuation currencies match (Python keeps FX-
// across-share-value out of scope). All-nil result when no valuation exists
// or zero shares are vested.
func computePaperValues(
	ctx context.Context,
	valuations *companyvaluations.Store,
	g *EquityGrant,
	vested int,
) (paperValueResult, error) {
	result := paperValueResult{}
	valuation, err := valuations.GetLatestForCompany(ctx, g.Company, nil)
	if err != nil {
		if errors.Is(err, companyvaluations.ErrNoValuation) {
			return result, nil
		}
		return result, err
	}
	if vested <= 0 {
		return result, nil
	}
	// Capture metadata that's also surfaced when currencies mismatch.
	valDate := valuation.Date
	result.ValuationDate = &valDate
	result.ValuationSource = valuation.Source
	if valuation.Currency != g.Currency {
		return result, nil
	}

	fmvBase := valuation.FMVPerShare
	fmvLow := fmvBase
	if valuation.FMVLow != nil {
		fmvLow = *valuation.FMVLow
	}
	fmvHigh := fmvBase
	if valuation.FMVHigh != nil {
		fmvHigh = *valuation.FMVHigh
	}
	perShareBase := intrinsicShareValue(g.Type, fmvBase, g.StrikePrice)
	perShareLow := intrinsicShareValue(g.Type, fmvLow, g.StrikePrice)
	perShareHigh := intrinsicShareValue(g.Type, fmvHigh, g.StrikePrice)
	vestedDec := decimal.NewFromInt(int64(vested))
	base := perShareBase.Mul(vestedDec)
	low := perShareLow.Mul(vestedDec)
	high := perShareHigh.Mul(vestedDec)
	result.Base = &base
	result.Low = &low
	result.High = &high
	result.Currency = valuation.Currency
	return result, nil
}

// intrinsicShareValue: FMV for RSU, max(FMV - strike, 0) for options.
func intrinsicShareValue(grantType string, fmv decimal.Decimal, strike *decimal.Decimal) decimal.Decimal {
	if grantType != "option" {
		return fmv
	}
	if strike == nil {
		return decimal.Zero
	}
	diff := fmv.Sub(*strike)
	if diff.IsNegative() {
		return decimal.Zero
	}
	return diff
}

// fxRateFor returns the rate for the valuation's currency on its date. Used
// to convert paper_value_*_pln so the PLN figure matches when the valuation
// was set, not today's FX (parity with Python's get_fx_rate_to_pln(db,
// pv_currency, pv_date)).
func fxRateFor(
	ctx context.Context,
	fxSvc *fx.Service,
	currency string,
	on *time.Time,
) (fx.Result, error) {
	if currency == "" {
		return fx.Result{}, nil
	}
	target := time.Time{}
	if on != nil {
		target = *on
	}
	return fxSvc.GetRateToPLN(ctx, currency, target)
}

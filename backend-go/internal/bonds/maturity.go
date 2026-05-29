package bonds

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
)

// LadderEventKind is the cashflow's role on the calendar.
//
// "redemption" is the principal returned at MaturityDate. For capitalized
// bonds it carries the compounded interest too; for non-capitalized bonds
// the interest has already been paid out year-by-year as separate
// "coupon" events.
type LadderEventKind string

const (
	EventRedemption LadderEventKind = "redemption"
	EventCoupon     LadderEventKind = "coupon"
)

// LadderEvent is one cashflow row in the maturity calendar. Rows are
// bucketed by (month, type, kind) so several bonds of the same series
// maturing in the same month collapse into a single line — that's the
// shape the issue's "grouped by maturity month and bond type" calls for.
type LadderEvent struct {
	Month         time.Time // first day of the bucket month, UTC
	Type          BondType
	Kind          LadderEventKind
	BondIDs       []int
	Count         int
	Principal     decimal.Decimal // redemption only
	InterestGross decimal.Decimal
	Tax           decimal.Decimal
	NetCashflow   decimal.Decimal
}

// NextMaturity is the nearest upcoming redemption, surfaced to the
// dashboard as the warning trigger.
type NextMaturity struct {
	Date          time.Time
	Type          BondType
	BondIDs       []int
	Count         int
	Principal     decimal.Decimal
	InterestGross decimal.Decimal
	Tax           decimal.Decimal
	NetCashflow   decimal.Decimal
	DaysUntil     int
}

// LadderResult bundles the calendar events with the dashboard's next-up
// warning so callers can compute both in one pass.
type LadderResult struct {
	Events       []LadderEvent
	NextMaturity *NextMaturity
}

// MaturityLadder projects future redemption + coupon cashflows for every
// bond in `bonds`, bucketed into month groups by (month, type, kind).
//
// Capitalize=true bonds (EDO, DOS) emit one redemption event at maturity
// carrying the full compounded value; the interest portion is taxed at
// `belkaRate` (encoded as a [0,1] fraction, 0.19 for Belka).
// Capitalize=false bonds (COI, ROR, TOZ) emit yearly coupon events plus
// a redemption-of-face at maturity — that matches how the Treasury
// actually pays them.
//
// Past events (cashflow date <= now) are dropped: the calendar is
// forward-looking, and a bond's matured principal has already hit the
// owner's account.
func MaturityLadder(bonds []TreasuryBond, yoyByYear map[int]decimal.Decimal, monthly map[cpi.YearMonth]decimal.Decimal, now time.Time, belkaRate decimal.Decimal) LadderResult {
	if len(bonds) == 0 {
		return LadderResult{Events: []LadderEvent{}, NextMaturity: nil}
	}
	bucketKey := func(month time.Time, t BondType, kind LadderEventKind) string {
		return month.Format("2006-01") + "|" + string(t) + "|" + string(kind)
	}
	buckets := map[string]*LadderEvent{}
	// Cache the per-bond redemption cashflow built during the main pass
	// so computeNextMaturity doesn't recompute YieldToMaturity below.
	redemptions := make(map[int]redemptionCashflow, len(bonds))

	for i := range bonds {
		b := &bonds[i]
		cf := appendBondEvents(b, yoyByYear, monthly, now, belkaRate, buckets, bucketKey)
		redemptions[b.ID] = cf
	}

	events := make([]LadderEvent, 0, len(buckets))
	for _, ev := range buckets {
		sort.Ints(ev.BondIDs)
		events = append(events, *ev)
	}
	sort.Slice(events, func(i, j int) bool {
		if !events[i].Month.Equal(events[j].Month) {
			return events[i].Month.Before(events[j].Month)
		}
		if events[i].Type != events[j].Type {
			return events[i].Type < events[j].Type
		}
		return events[i].Kind < events[j].Kind
	})

	next := computeNextMaturity(bonds, yoyByYear, monthly, now, belkaRate, redemptions)
	return LadderResult{Events: events, NextMaturity: next}
}

// appendBondEvents emits all calendar events for one bond and returns
// the redemption cashflow so the dashboard-warning pass can reuse it
// without recomputing YieldToMaturity.
func appendBondEvents(
	b *TreasuryBond,
	yoy map[int]decimal.Decimal,
	monthly map[cpi.YearMonth]decimal.Decimal,
	now time.Time,
	belka decimal.Decimal,
	buckets map[string]*LadderEvent,
	keyFn func(time.Time, BondType, LadderEventKind) string,
) redemptionCashflow {
	tenor := bondTenorYears(b.Type)
	if tenor <= 0 {
		return redemptionCashflow{}
	}

	if b.Capitalize {
		final := YieldToMaturity(b, yoy, monthly)
		if len(final) == 0 {
			return redemptionCashflow{}
		}
		finalValue := final[len(final)-1].Value
		interest := finalValue.Sub(b.FaceValue)
		if interest.IsNegative() {
			interest = decimal.Zero
		}
		tax := interest.Mul(belka).Round(2)
		net := finalValue.Sub(tax)
		cf := redemptionCashflow{final: finalValue, interest: interest, tax: tax, net: net}
		maturityDate := MaturityDate(b)
		if maturityDate.After(now) {
			addEvent(buckets, keyFn, maturityDate, b.Type, EventRedemption, b.ID, b.FaceValue, interest, tax, net)
		}
		return cf
	}

	for year := 1; year <= tenor; year++ {
		couponDate := b.PurchaseDate.AddDate(0, 12*year, 0)
		if !couponDate.After(now) {
			continue
		}
		rate := YearRate(b, yoy, monthly, year)
		gross := b.FaceValue.Mul(rate).Div(decimal.NewFromInt(100)).Round(2)
		tax := gross.Mul(belka).Round(2)
		net := gross.Sub(tax)
		addEvent(buckets, keyFn, couponDate, b.Type, EventCoupon, b.ID, decimal.Zero, gross, tax, net)
	}

	maturityDate := MaturityDate(b)
	if maturityDate.After(now) {
		addEvent(buckets, keyFn, maturityDate, b.Type, EventRedemption, b.ID, b.FaceValue, decimal.Zero, decimal.Zero, b.FaceValue)
	}
	return redemptionCashflow{final: b.FaceValue, net: b.FaceValue}
}

func addEvent(
	buckets map[string]*LadderEvent,
	keyFn func(time.Time, BondType, LadderEventKind) string,
	when time.Time,
	t BondType,
	kind LadderEventKind,
	bondID int,
	principal, gross, tax, net decimal.Decimal,
) {
	month := time.Date(when.Year(), when.Month(), 1, 0, 0, 0, 0, time.UTC)
	key := keyFn(month, t, kind)
	ev, ok := buckets[key]
	if !ok {
		ev = &LadderEvent{
			Month:         month,
			Type:          t,
			Kind:          kind,
			BondIDs:       []int{},
			Principal:     decimal.Zero,
			InterestGross: decimal.Zero,
			Tax:           decimal.Zero,
			NetCashflow:   decimal.Zero,
		}
		buckets[key] = ev
	}
	ev.BondIDs = append(ev.BondIDs, bondID)
	ev.Count++
	ev.Principal = ev.Principal.Add(principal)
	ev.InterestGross = ev.InterestGross.Add(gross)
	ev.Tax = ev.Tax.Add(tax)
	ev.NetCashflow = ev.NetCashflow.Add(net)
}

// redemptionCashflow is the (final value, interest, tax, net) tuple for
// a bond's redemption row. Capitalized bonds carry compounded interest;
// non-capitalized bonds return only principal at maturity (the interest
// has already been paid as yearly coupons).
type redemptionCashflow struct{ final, interest, tax, net decimal.Decimal }

// computeNextMaturity collapses bonds redeeming in the soonest future
// month into one warning record. Aggregation is restricted to bonds of
// the *same type* as the nearest match so the surfaced `Type` and the
// summed cashflow stay coherent — if e.g. an EDO and a DOS happen to
// mature in the same month, the warning still describes a single
// homogeneous batch.
func computeNextMaturity(bonds []TreasuryBond, yoy map[int]decimal.Decimal, monthly map[cpi.YearMonth]decimal.Decimal, now time.Time, belka decimal.Decimal, redemptions map[int]redemptionCashflow) *NextMaturity {
	nearestIdx, nearestDate := pickNearestMaturity(bonds, now)
	if nearestIdx < 0 {
		return nil
	}
	nearestType := bonds[nearestIdx].Type
	out := &NextMaturity{Date: nearestDate, DaysUntil: daysBetween(now, nearestDate), Type: nearestType}
	for i := range bonds {
		b := &bonds[i]
		if b.Type != nearestType {
			continue
		}
		md := MaturityDate(b)
		if !md.After(now) || !sameMonth(md, nearestDate) {
			continue
		}
		cf, ok := redemptions[b.ID]
		if !ok {
			continue
		}
		out.BondIDs = append(out.BondIDs, b.ID)
		out.Count++
		out.Principal = out.Principal.Add(b.FaceValue)
		out.InterestGross = out.InterestGross.Add(cf.interest)
		out.Tax = out.Tax.Add(cf.tax)
		out.NetCashflow = out.NetCashflow.Add(cf.net)
		// For non-capitalized bonds the final-year coupon lands in the
		// same month as the redemption — fold it in so "netto" matches
		// what the owner actually receives that month, not just the
		// principal.
		if !b.Capitalize {
			coupon := finalYearCoupon(b, yoy, monthly, belka)
			out.InterestGross = out.InterestGross.Add(coupon.gross)
			out.Tax = out.Tax.Add(coupon.tax)
			out.NetCashflow = out.NetCashflow.Add(coupon.net)
		}
		if md.Before(out.Date) {
			out.Date = md
			out.DaysUntil = daysBetween(now, md)
		}
	}
	sort.Ints(out.BondIDs)
	return out
}

func pickNearestMaturity(bonds []TreasuryBond, now time.Time) (int, time.Time) {
	idx := -1
	var earliest time.Time
	for i := range bonds {
		md := MaturityDate(&bonds[i])
		if !md.After(now) {
			continue
		}
		if idx < 0 || md.Before(earliest) {
			idx = i
			earliest = md
		}
	}
	return idx, earliest
}

// finalYearCoupon returns the gross/tax/net for the last coupon of a
// non-capitalized bond. Used by computeNextMaturity to fold the
// final-year payout into the redemption-month summary.
type couponAmounts struct{ gross, tax, net decimal.Decimal }

func finalYearCoupon(b *TreasuryBond, yoy map[int]decimal.Decimal, monthly map[cpi.YearMonth]decimal.Decimal, belka decimal.Decimal) couponAmounts {
	tenor := bondTenorYears(b.Type)
	if tenor <= 0 {
		return couponAmounts{}
	}
	rate := YearRate(b, yoy, monthly, tenor)
	gross := b.FaceValue.Mul(rate).Div(decimal.NewFromInt(100)).Round(2)
	tax := gross.Mul(belka).Round(2)
	return couponAmounts{gross: gross, tax: tax, net: gross.Sub(tax)}
}

func sameMonth(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month()
}

// daysBetween counts whole UTC calendar days between two instants. We
// snap both ends to UTC midnight before subtracting so a dashboard
// countdown rendered in Europe/Warsaw doesn't flicker by ±1 day vs. a
// UTC-stored maturity date.
func daysBetween(from, to time.Time) int {
	f := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	t := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	days := int(t.Sub(f).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

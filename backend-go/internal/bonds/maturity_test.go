package bonds

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

var testBelka = decimal.NewFromFloat(0.19)

func TestMaturityLadderCapitalizedEDOEmitsSingleRedemption(t *testing.T) {
	// 1-year-ago purchase; maturity 9y in the future.
	purchase := time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	b := bondEDO(purchase)
	b.ID = 1
	yoy := map[int]decimal.Decimal{2025: decimal.NewFromFloat(104)}

	res := MaturityLadder([]TreasuryBond{*b}, yoy, nil, now, testBelka)

	if len(res.Events) != 1 {
		t.Fatalf("EDO should produce 1 event, got %d: %+v", len(res.Events), res.Events)
	}
	ev := res.Events[0]
	if ev.Kind != EventRedemption {
		t.Fatalf("EDO event kind = %s, want redemption", ev.Kind)
	}
	if ev.Type != BondEDO {
		t.Fatalf("event type = %s, want EDO", ev.Type)
	}
	wantMonth := time.Date(2035, 5, 1, 0, 0, 0, 0, time.UTC)
	if !ev.Month.Equal(wantMonth) {
		t.Fatalf("event month = %s, want %s", ev.Month, wantMonth)
	}
	if ev.Count != 1 || len(ev.BondIDs) != 1 || ev.BondIDs[0] != 1 {
		t.Fatalf("bond grouping wrong: %+v", ev)
	}
	if !ev.Principal.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("principal = %s, want 1000", ev.Principal)
	}
	if !ev.InterestGross.IsPositive() {
		t.Fatalf("interest should be positive, got %s", ev.InterestGross)
	}
	expectTax := ev.InterestGross.Mul(testBelka).Round(2)
	if !ev.Tax.Equal(expectTax) {
		t.Fatalf("tax = %s, want %s (19%% of interest)", ev.Tax, expectTax)
	}
	wantNet := ev.Principal.Add(ev.InterestGross).Sub(ev.Tax)
	if !ev.NetCashflow.Equal(wantNet) {
		t.Fatalf("net = %s, want %s", ev.NetCashflow, wantNet)
	}
}

func TestMaturityLadderCOIEmitsYearlyCouponsAndRedemption(t *testing.T) {
	// 4y COI purchased 2026-01-15 → 4 coupons + 1 redemption.
	purchase := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	b := bondCOI(purchase)
	b.ID = 7
	yoy := map[int]decimal.Decimal{
		2026: decimal.NewFromFloat(104),
		2027: decimal.NewFromFloat(103),
		2028: decimal.NewFromFloat(102),
	}

	res := MaturityLadder([]TreasuryBond{*b}, yoy, nil, now, testBelka)

	if len(res.Events) != 5 {
		t.Fatalf("COI should produce 5 events (4 coupons + redemption), got %d", len(res.Events))
	}

	coupons := 0
	redemptions := 0
	for _, ev := range res.Events {
		switch ev.Kind {
		case EventCoupon:
			coupons++
			if !ev.Principal.Equal(decimal.Zero) {
				t.Fatalf("coupon principal must be 0, got %s", ev.Principal)
			}
			if !ev.InterestGross.IsPositive() {
				t.Fatalf("coupon gross must be positive, got %s", ev.InterestGross)
			}
			expectTax := ev.InterestGross.Mul(testBelka).Round(2)
			if !ev.Tax.Equal(expectTax) {
				t.Fatalf("coupon tax = %s, want %s", ev.Tax, expectTax)
			}
		case EventRedemption:
			redemptions++
			if !ev.Principal.Equal(decimal.NewFromInt(1000)) {
				t.Fatalf("COI redemption principal = %s, want 1000", ev.Principal)
			}
			if !ev.InterestGross.IsZero() {
				t.Fatalf("COI redemption interest should be 0 (paid as coupons), got %s", ev.InterestGross)
			}
			if !ev.NetCashflow.Equal(decimal.NewFromInt(1000)) {
				t.Fatalf("COI net redemption = %s, want 1000", ev.NetCashflow)
			}
		}
	}
	if coupons != 4 || redemptions != 1 {
		t.Fatalf("event mix: coupons=%d, redemptions=%d", coupons, redemptions)
	}
}

func TestMaturityLadderSkipsPastEvents(t *testing.T) {
	purchase := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC) // ~2y in
	b := bondCOI(purchase)
	b.ID = 1
	yoy := map[int]decimal.Decimal{2025: decimal.NewFromFloat(104), 2026: decimal.NewFromFloat(103)}

	res := MaturityLadder([]TreasuryBond{*b}, yoy, nil, now, testBelka)

	// year-1 (2025-01) + year-2 (2026-01) coupons are in the past; only
	// year-3 + year-4 coupons and the redemption remain.
	if len(res.Events) != 3 {
		t.Fatalf("expected 3 future events (2 coupons + 1 redemption), got %d", len(res.Events))
	}
	for _, ev := range res.Events {
		if !ev.Month.After(now.AddDate(0, 0, -30)) {
			t.Fatalf("emitted past event at %s", ev.Month)
		}
	}
}

func TestMaturityLadderGroupsSameMonthSameType(t *testing.T) {
	// Two EDO bonds maturing in the same month → one row in the ladder.
	purchase := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	b1 := bondEDO(purchase)
	b1.ID = 1
	b2 := bondEDO(purchase.AddDate(0, 0, 10)) // same month
	b2.ID = 2

	res := MaturityLadder([]TreasuryBond{*b1, *b2}, map[int]decimal.Decimal{}, nil, now, testBelka)

	if len(res.Events) != 1 {
		t.Fatalf("two EDO maturing in same month should merge, got %d events", len(res.Events))
	}
	ev := res.Events[0]
	if ev.Count != 2 {
		t.Fatalf("merged event count = %d, want 2", ev.Count)
	}
	if ev.BondIDs[0] != 1 || ev.BondIDs[1] != 2 {
		t.Fatalf("merged bond IDs = %v, want [1, 2]", ev.BondIDs)
	}
	wantPrincipal := decimal.NewFromInt(2000)
	if !ev.Principal.Equal(wantPrincipal) {
		t.Fatalf("merged principal = %s, want %s", ev.Principal, wantPrincipal)
	}
}

func TestMaturityLadderSortsByMonthThenTypeThenKind(t *testing.T) {
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	earlyEDO := bondEDO(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) // 2035-01
	earlyEDO.ID = 1
	lateEDO := bondEDO(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)) // 2036-06
	lateEDO.ID = 2

	res := MaturityLadder([]TreasuryBond{*lateEDO, *earlyEDO}, map[int]decimal.Decimal{}, nil, now, testBelka)
	if len(res.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(res.Events))
	}
	if !res.Events[0].Month.Before(res.Events[1].Month) {
		t.Fatalf("events not sorted by month: %v", res.Events)
	}
}

func TestNextMaturityIsNearestFuture(t *testing.T) {
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	// EDO matures 2035-03; DOS (2y) purchased 2026-01 matures 2028-01 → nearest.
	soon := &TreasuryBond{
		ID:            10,
		Type:          BondDOS,
		FaceValue:     decimal.NewFromInt(5000),
		PurchaseDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FirstYearRate: decimal.NewFromFloat(6.30),
		Margin:        decimal.Zero,
		Capitalize:    true,
	}
	far := bondEDO(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC))
	far.ID = 11

	res := MaturityLadder([]TreasuryBond{*far, *soon}, map[int]decimal.Decimal{}, nil, now, testBelka)
	if res.NextMaturity == nil {
		t.Fatal("NextMaturity should be set")
	}
	if res.NextMaturity.Type != BondDOS {
		t.Fatalf("nearest type = %s, want DOS", res.NextMaturity.Type)
	}
	if res.NextMaturity.BondIDs[0] != 10 {
		t.Fatalf("nearest bond ID = %d, want 10", res.NextMaturity.BondIDs[0])
	}
	if res.NextMaturity.DaysUntil <= 0 {
		t.Fatalf("DaysUntil should be positive, got %d", res.NextMaturity.DaysUntil)
	}
}

func TestNextMaturityAggregatesSameMonth(t *testing.T) {
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	dos1 := &TreasuryBond{
		ID:            1,
		Type:          BondDOS,
		FaceValue:     decimal.NewFromInt(1000),
		PurchaseDate:  time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		FirstYearRate: decimal.NewFromFloat(6.30),
		Capitalize:    true,
	}
	dos2 := &TreasuryBond{
		ID:            2,
		Type:          BondDOS,
		FaceValue:     decimal.NewFromInt(2000),
		PurchaseDate:  time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC), // same maturity month
		FirstYearRate: decimal.NewFromFloat(6.30),
		Capitalize:    true,
	}

	res := MaturityLadder([]TreasuryBond{*dos1, *dos2}, map[int]decimal.Decimal{}, nil, now, testBelka)
	if res.NextMaturity == nil {
		t.Fatal("NextMaturity should be set")
	}
	if res.NextMaturity.Count != 2 {
		t.Fatalf("nearest count = %d, want 2", res.NextMaturity.Count)
	}
	wantPrincipal := decimal.NewFromInt(3000)
	if !res.NextMaturity.Principal.Equal(wantPrincipal) {
		t.Fatalf("nearest principal = %s, want %s", res.NextMaturity.Principal, wantPrincipal)
	}
	// Date is the earliest individual maturity in the group.
	if res.NextMaturity.Date.Day() != 5 {
		t.Fatalf("nearest date day = %d, want 5 (earliest)", res.NextMaturity.Date.Day())
	}
}

func TestMaturityLadderEmptyForNoBonds(t *testing.T) {
	res := MaturityLadder(nil, map[int]decimal.Decimal{}, nil, time.Now(), testBelka)
	if len(res.Events) != 0 || res.NextMaturity != nil {
		t.Fatalf("empty input should return empty result, got %+v", res)
	}
}

func TestMaturityLadderNonCapFinalMonthEmitsCouponAndRedemption(t *testing.T) {
	// COI year-4 coupon date == maturity date → same bucket month
	// produces two distinct rows (kind=coupon, kind=redemption).
	purchase := time.Date(2022, 5, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	b := bondCOI(purchase)
	b.ID = 1
	yoy := map[int]decimal.Decimal{2025: decimal.NewFromFloat(104)}

	res := MaturityLadder([]TreasuryBond{*b}, yoy, nil, now, testBelka)

	if len(res.Events) != 2 {
		t.Fatalf("final month should have 2 rows (coupon + redemption), got %d", len(res.Events))
	}
	kinds := map[LadderEventKind]bool{}
	for _, ev := range res.Events {
		if !sameMonth(ev.Month, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("event month %s not in final month", ev.Month)
		}
		kinds[ev.Kind] = true
	}
	if !kinds[EventCoupon] || !kinds[EventRedemption] {
		t.Fatalf("expected both coupon + redemption, got %+v", kinds)
	}
}

func TestNextMaturityFoldsFinalCouponForNonCapBond(t *testing.T) {
	// COI bond maturing soon: NextMaturity.NetCashflow should include
	// principal + final coupon (after tax), not just principal.
	purchase := time.Date(2022, 5, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	b := bondCOI(purchase)
	b.ID = 1
	yoy := map[int]decimal.Decimal{2025: decimal.NewFromFloat(104)}

	res := MaturityLadder([]TreasuryBond{*b}, yoy, nil, now, testBelka)

	if res.NextMaturity == nil {
		t.Fatal("NextMaturity should be set")
	}
	if !res.NextMaturity.NetCashflow.GreaterThan(decimal.NewFromInt(1000)) {
		t.Fatalf("net should include final coupon on top of 1000 principal, got %s",
			res.NextMaturity.NetCashflow)
	}
	if !res.NextMaturity.InterestGross.IsPositive() {
		t.Fatalf("InterestGross should reflect final coupon, got %s",
			res.NextMaturity.InterestGross)
	}
}

func TestMaturityLadderHandlesNilYoYMap(t *testing.T) {
	// With no CPI data, year-N rates fall back to FirstYearRate so the
	// ladder still computes a sensible non-zero cashflow.
	purchase := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	b := bondCOI(purchase)
	b.ID = 1

	res := MaturityLadder([]TreasuryBond{*b}, nil, nil, now, testBelka)

	if len(res.Events) == 0 {
		t.Fatal("nil yoy should still emit future events via FirstYearRate fallback")
	}
	for _, ev := range res.Events {
		if ev.Kind == EventCoupon && !ev.InterestGross.IsPositive() {
			t.Fatalf("nil-yoy coupon should still be positive, got %s", ev.InterestGross)
		}
	}
}

func TestDaysBetweenIsTimezoneStable(t *testing.T) {
	// Maturity stored as UTC midnight; now in Europe/Warsaw same day → 0
	// days, not -1 or 1. Snapping to UTC date prevents the dashboard
	// countdown flickering at midnight.
	warsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		t.Fatalf("load Europe/Warsaw: %v", err)
	}
	maturity := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	nowWarsaw := time.Date(2026, 5, 31, 23, 30, 0, 0, warsaw) // 21:30 UTC same day before
	if got := daysBetween(nowWarsaw, maturity); got != 1 {
		t.Fatalf("daysBetween should be 1 (next UTC day), got %d", got)
	}
	sameDayWarsaw := time.Date(2026, 6, 1, 14, 0, 0, 0, warsaw) // noon Warsaw on maturity day
	if got := daysBetween(sameDayWarsaw, maturity); got != 0 {
		t.Fatalf("daysBetween on maturity day should be 0, got %d", got)
	}
}

func TestMaturityLadderSkipsAlreadyMaturedBond(t *testing.T) {
	// EDO purchased 2010 → matured 2020, before `now` 2026.
	b := bondEDO(time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC))
	b.ID = 99
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)

	res := MaturityLadder([]TreasuryBond{*b}, map[int]decimal.Decimal{}, nil, now, testBelka)
	if len(res.Events) != 0 {
		t.Fatalf("matured bond should emit no future events, got %d", len(res.Events))
	}
	if res.NextMaturity != nil {
		t.Fatal("NextMaturity should be nil when all bonds have matured")
	}
}

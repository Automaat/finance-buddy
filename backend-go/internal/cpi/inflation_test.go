package cpi

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestCumulativeIndexEmpty(t *testing.T) {
	out := CumulativeIndex(map[int]decimal.Decimal{})
	if len(out) != 0 {
		t.Fatalf("expected empty, got %+v", out)
	}
}

func TestCumulativeIndexSingleYearIsAnchor(t *testing.T) {
	out := CumulativeIndex(map[int]decimal.Decimal{2024: d("110")})
	if got := out[2024]; !got.Equal(d("100")) {
		t.Fatalf("expected anchor 100, got %s", got)
	}
}

func TestCumulativeIndexCompoundsForward(t *testing.T) {
	in := map[int]decimal.Decimal{
		2022: d("100"),
		2023: d("110"),
		2024: d("105"),
	}
	out := CumulativeIndex(in)
	if !out[2022].Equal(d("100")) {
		t.Fatalf("2022 should anchor at 100, got %s", out[2022])
	}
	if !out[2023].Equal(d("110")) {
		t.Fatalf("2023 expected 110 (100 * 110 / 100), got %s", out[2023])
	}
	if !out[2024].Equal(d("115.5")) {
		t.Fatalf("2024 expected 115.5 (110 * 105 / 100), got %s", out[2024])
	}
}

func TestIndexAtDateEmptyIsError(t *testing.T) {
	_, err := IndexAtDate(map[int]decimal.Decimal{}, time.Now())
	if !errors.Is(err, ErrInflationDataMissing) {
		t.Fatalf("expected ErrInflationDataMissing, got %v", err)
	}
}

func TestIndexAtDateBelowRangeClampsToEarliest(t *testing.T) {
	in := map[int]decimal.Decimal{2024: d("100"), 2025: d("110")}
	got, err := IndexAtDate(in, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !got.Equal(d("100")) {
		t.Fatalf("expected 100, got %s", got)
	}
}

func TestIndexAtDateAboveRangeClampsToLatest(t *testing.T) {
	in := map[int]decimal.Decimal{2024: d("100"), 2025: d("110")}
	got, err := IndexAtDate(in, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !got.Equal(d("110")) {
		t.Fatalf("expected 110, got %s", got)
	}
}

func TestIndexAtDateInterpolatesMidYear(t *testing.T) {
	in := map[int]decimal.Decimal{2024: d("100"), 2025: d("110")}
	// Mid 2025 — 100 + (110-100)*0.5 ≈ 105 (give or take a fraction)
	mid := time.Date(2025, 7, 2, 0, 0, 0, 0, time.UTC)
	got, err := IndexAtDate(in, mid)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	f, _ := got.Float64()
	if f < 104.5 || f > 105.5 {
		t.Fatalf("expected ~105, got %s", got)
	}
}

func TestIndexAtDateMissingYearIsHardError(t *testing.T) {
	in := map[int]decimal.Decimal{2024: d("100"), 2026: d("110")}
	_, err := IndexAtDate(in, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	if err == nil || errors.Is(err, ErrInflationDataMissing) == false {
		t.Fatalf("expected wrapped missing-data error, got %v", err)
	}
}

func TestAdjustWithIndexEmpty(t *testing.T) {
	_, err := AdjustWithIndex(map[int]decimal.Decimal{}, 100, time.Now(), time.Now())
	if !errors.Is(err, ErrInflationDataMissing) {
		t.Fatalf("expected ErrInflationDataMissing, got %v", err)
	}
}

func TestAdjustWithIndexInflate(t *testing.T) {
	// Index goes 100 -> 110 -> 120 across three contiguous years. Adjusting
	// 100 PLN from end-2022 to end-2024 should compound to ~120.
	in := map[int]decimal.Decimal{2022: d("100"), 2023: d("110"), 2024: d("120")}
	got, err := AdjustWithIndex(in, 100,
		time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got < 119 || got > 121 {
		t.Fatalf("expected ~120, got %v", got)
	}
}

func TestAdjustWithIndexZeroSourceIsError(t *testing.T) {
	in := map[int]decimal.Decimal{2020: d("0"), 2025: d("110")}
	_, err := AdjustWithIndex(in, 100,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("expected error on zero source")
	}
}

func TestInflationErrorMessage(t *testing.T) {
	e := newInflationErr("custom")
	if e.Error() != "custom" {
		t.Fatalf("got %q want custom", e.Error())
	}
	if !errors.Is(e, ErrInflationDataMissing) {
		t.Fatal("expected errors.Is to match family")
	}
}

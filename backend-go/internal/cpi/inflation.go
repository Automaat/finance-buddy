// Package cpi implements /api/cpi — series, inflation adjust, refresh.
//
// Math (inflation.go) is a pure-functions port of
// backend/app/services/inflation.py. yoy_rate is stored as published by GUS
// (114.4 = +14.4% YoY). A fixed-base cumulative index is derived per call;
// arbitrary dates linearly interpolate within a year.
package cpi

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// ErrInflationDataMissing mirrors Python's InflationDataMissingError. Use
// errors.Is to detect the family from the handler — both the
// empty-table case and the zero-source-index case wrap it.
var ErrInflationDataMissing = errors.New("CPI table is empty")

// inflationDataError carries a specific message but answers errors.Is for
// ErrInflationDataMissing so the handler routes both to 503 with the
// specific message preserved (matches Python's raise InflationDataMissingError(msg)).
type inflationDataError struct{ msg string }

func (e *inflationDataError) Error() string { return e.msg }
func (e *inflationDataError) Is(target error) bool {
	return target == ErrInflationDataMissing
}

func newInflationErr(msg string) error {
	return &inflationDataError{msg: msg}
}

// YearRate is a (year, yoy_rate) pair as published by GUS.
type YearRate struct {
	Year int
	YoY  decimal.Decimal
}

// CumulativeIndex returns a fixed-base index map: index[Y] is the end-of-year-Y
// price level, anchored at index[earliest] = 100 and compounded forward by
// yoy_rate.
func CumulativeIndex(yoyByYear map[int]decimal.Decimal) map[int]decimal.Decimal {
	if len(yoyByYear) == 0 {
		return map[int]decimal.Decimal{}
	}
	years := sortedYears(yoyByYear)
	index := map[int]decimal.Decimal{years[0]: decimal.NewFromInt(100)}
	hundred := decimal.NewFromInt(100)
	for i := 1; i < len(years); i++ {
		prev := years[i-1]
		year := years[i]
		index[year] = index[prev].Mul(yoyByYear[year]).Div(hundred)
	}
	return index
}

// IndexAtDate interpolates the fixed-base index at an arbitrary calendar date.
// Outside the known range we clamp to the earliest or latest known year.
// Inside the range, a missing year (gap in the series) is a hard error —
// silently zeroing it would corrupt downstream math.
func IndexAtDate(indexByYear map[int]decimal.Decimal, when time.Time) (decimal.Decimal, error) {
	if len(indexByYear) == 0 {
		return decimal.Zero, ErrInflationDataMissing
	}
	years := sortedYears(indexByYear)
	if when.Year() < years[0] {
		return indexByYear[years[0]], nil
	}
	if when.Year() > years[len(years)-1] {
		return indexByYear[years[len(years)-1]], nil
	}
	endIdx, ok := indexByYear[when.Year()]
	if !ok {
		return decimal.Zero, newInflationErr(fmt.Sprintf("CPI series missing year %d", when.Year()))
	}
	startIdx, ok := indexByYear[when.Year()-1]
	if !ok {
		startIdx = indexByYear[years[0]]
	}

	yearStart := time.Date(when.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	nextYearStart := time.Date(when.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)
	spanDays := decimal.NewFromInt(int64(nextYearStart.Sub(yearStart).Hours() / 24))
	when = time.Date(when.Year(), when.Month(), when.Day(), 0, 0, 0, 0, time.UTC)
	elapsedDays := decimal.NewFromInt(int64(when.Sub(yearStart).Hours() / 24))
	fraction := elapsedDays.Div(spanDays)
	return startIdx.Add(endIdx.Sub(startIdx).Mul(fraction)), nil
}

// AdjustWithIndex inflates/deflates amount between two dates using a
// pre-loaded fixed-base index.
func AdjustWithIndex(
	indexByYear map[int]decimal.Decimal,
	amount float64,
	from, to time.Time,
) (float64, error) {
	if len(indexByYear) == 0 {
		return 0, ErrInflationDataMissing
	}
	fromIdx, err := IndexAtDate(indexByYear, from)
	if err != nil {
		return 0, err
	}
	toIdx, err := IndexAtDate(indexByYear, to)
	if err != nil {
		return 0, err
	}
	if fromIdx.IsZero() {
		return 0, newInflationErr("Source index is zero")
	}
	factor := toIdx.Div(fromIdx)
	amountDec := decimal.NewFromFloat(amount)
	result, _ := amountDec.Mul(factor).Float64()
	return result, nil
}

func sortedYears(m map[int]decimal.Decimal) []int {
	years := make([]int, 0, len(m))
	for y := range m {
		years = append(years, y)
	}
	sort.Ints(years)
	return years
}

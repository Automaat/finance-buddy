package dashboard

import (
	"sort"
	"time"
)

const (
	monthlyWorkHours = 160.0
	monthlyLifeHours = 730.0
)

// deltaValue mirrors the schema's DeltaValue. Percentage is nil when the
// baseline is 0.
type deltaValue struct {
	Absolute   float64
	Percentage *float64
}

// tileDelta mirrors TileDelta.
type tileDelta struct {
	MoM *deltaValue
	YoY *deltaValue
}

// tileDeltas mirrors TileDeltas.
type tileDeltas struct {
	NetWorth    tileDelta
	Assets      tileDelta
	Liabilities tileDelta
}

// computeTileDeltas ports metrics.compute_tile_deltas.
//
// MoM window = [latest-45d, latest-15d]; YoY = [latest-395d, latest-335d].
// The latest snapshot is excluded from baseline candidates.
func computeTileDeltas(totals []snapshotTotals) tileDeltas {
	if len(totals) == 0 {
		return tileDeltas{}
	}
	current := totals[len(totals)-1]
	prior := totals[:len(totals)-1]
	latest := current.Date

	momBase := pickBaseline(prior, latest.AddDate(0, 0, -45), latest.AddDate(0, 0, -15))
	yoyBase := pickBaseline(prior, latest.AddDate(0, 0, -395), latest.AddDate(0, 0, -335))

	tile := func(get func(snapshotTotals) float64) tileDelta {
		cur := get(current)
		return tileDelta{
			MoM: delta(cur, momBase, get),
			YoY: delta(cur, yoyBase, get),
		}
	}
	return tileDeltas{
		NetWorth:    tile(func(t snapshotTotals) float64 { return t.NetWorth }),
		Assets:      tile(func(t snapshotTotals) float64 { return t.Assets }),
		Liabilities: tile(func(t snapshotTotals) float64 { return t.Liabilities }),
	}
}

// pickBaseline returns the latest row with low <= date <= high; ties broken
// by snapshot_id desc.
func pickBaseline(totals []snapshotTotals, low, high time.Time) *snapshotTotals {
	window := []snapshotTotals{}
	for _, t := range totals {
		if !t.Date.Before(low) && !t.Date.After(high) {
			window = append(window, t)
		}
	}
	if len(window) == 0 {
		return nil
	}
	maxDate := window[0].Date
	for _, t := range window {
		if t.Date.After(maxDate) {
			maxDate = t.Date
		}
	}
	sameDay := []snapshotTotals{}
	for _, t := range window {
		if t.Date.Equal(maxDate) {
			sameDay = append(sameDay, t)
		}
	}
	sort.Slice(sameDay, func(i, j int) bool {
		return sameDay[i].SnapshotID > sameDay[j].SnapshotID
	})
	out := sameDay[0]
	return &out
}

func delta(current float64, baseline *snapshotTotals, get func(snapshotTotals) float64) *deltaValue {
	if baseline == nil {
		return nil
	}
	base := get(*baseline)
	absChange := current - base
	dv := deltaValue{Absolute: absChange}
	if base != 0 {
		pct := absChange / abs(base) * 100
		dv.Percentage = &pct
	}
	return &dv
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// savingsRate ports _calculate_savings_rate / the aggregate variant: average
// of the last 3 monthly net-worth deltas divided by the average of the last
// 3 salaries, ×100. nil when there's insufficient data. netWorths must be in
// the same chronological order Python uses (last 4 are the recent window).
func savingsRate(netWorths, salaries []float64) *float64 {
	if len(netWorths) < 4 {
		return nil
	}
	last4 := netWorths[len(netWorths)-4:]
	avgDelta := ((last4[1] - last4[0]) + (last4[2] - last4[1]) + (last4[3] - last4[2])) / 3
	if len(salaries) < 3 {
		return nil
	}
	avgSalary := (salaries[0] + salaries[1] + salaries[2]) / 3
	if avgSalary == 0 {
		return nil
	}
	rate := avgDelta / avgSalary * 100
	return &rate
}

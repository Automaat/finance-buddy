package dashboard

import (
	"fmt"
	"sort"
	"time"
)

// aggregatePointsBySnapshot collapses snapshot_aggregates rows (which are
// per-(snapshot, owner)) into per-snapshot net-worth/assets/liabilities
// points sorted by date. Missing snapshot dates cause a fail-fast error.
func aggregatePointsBySnapshot(aggRows []AggregateRow, snapshotDate map[int]time.Time) ([]netWorthPoint, error) {
	type snapAgg struct {
		nw, assets, liabs float64
	}
	bySnap := map[int]snapAgg{}
	for _, r := range aggRows {
		nw, _ := r.NetWorth.Float64()
		ta, _ := r.TotalAssets.Float64()
		tl, _ := r.TotalLiabilities.Float64()
		agg := bySnap[r.SnapshotID]
		agg.nw += nw
		agg.assets += ta
		agg.liabs += tl
		bySnap[r.SnapshotID] = agg
	}
	points := make([]netWorthPoint, 0, len(bySnap))
	for sid, agg := range bySnap {
		date, ok := snapshotDate[sid]
		if !ok {
			// A snapshot_aggregates row references a snapshot that doesn't
			// exist — partial restore / corruption. Fail fast, as Python's
			// snap_date[sid] KeyError does, rather than skewing the history
			// with a zero date.
			return nil, fmt.Errorf("aggregate references missing snapshot %d", sid)
		}
		points = append(points, netWorthPoint{
			Date:        date,
			Value:       agg.nw,
			Assets:      agg.assets,
			Liabilities: agg.liabs,
			SnapshotID:  sid,
		})
	}
	// Date primary, SnapshotID tiebreaker — keeps order stable when two
	// snapshots share a calendar day.
	sort.Slice(points, func(i, j int) bool {
		if !points[i].Date.Equal(points[j].Date) {
			return points[i].Date.Before(points[j].Date)
		}
		return points[i].SnapshotID < points[j].SnapshotID
	})
	return points, nil
}

// retirementValueOf sums latest-snapshot account-asset rows with purpose=retirement.
func retirementValueOf(latestRows []mergedRow) float64 {
	total := 0.0
	for _, r := range latestRows {
		if r.AccountID != nil && r.AccType != nil && *r.AccType == "asset" &&
			r.Purpose != nil && *r.Purpose == "retirement" {
			total += r.Value
		}
	}
	return total
}

func rawTotals(latestRows []mergedRow) (float64, float64) {
	assets := 0.0
	liabilities := 0.0
	for _, r := range latestRows {
		switch {
		case r.AssetID != nil:
			assets += r.Value
		case r.AccountID != nil && r.AccType != nil:
			switch *r.AccType {
			case "asset":
				assets += r.Value
			case "liability":
				liabilities += r.Value
			}
		}
	}
	return assets, liabilities
}

func rawAllocation(latestRows []mergedRow) []allocationItem {
	// owner is *int — pack it into a comparable key. shared==true marks a
	// nil owner_user_id (jointly owned).
	type key struct {
		cat    string
		owner  int
		shared bool
	}
	sums := map[key]float64{}
	for _, r := range latestRows {
		if r.AccountID == nil || r.AccType == nil || *r.AccType != "asset" {
			continue
		}
		if r.Category == nil {
			continue
		}
		k := key{cat: *r.Category}
		if r.OwnerUserID == nil {
			k.shared = true
		} else {
			k.owner = *r.OwnerUserID
		}
		sums[k] += r.Value
	}
	out := make([]allocationItem, 0, len(sums))
	for k, v := range sums {
		item := allocationItem{Category: k.cat, Value: v}
		if !k.shared {
			id := k.owner
			item.OwnerUserID = &id
		}
		out = append(out, item)
	}
	sortAllocation(out)
	return out
}

// aggregateMonthlyNetWorths ports _compute_savings_rate_from_aggregates'
// month-bucketing: latest snapshot per month, sorted chronologically.
func aggregateMonthlyNetWorths(aggRows []AggregateRow) []float64 {
	snapNW := map[int]float64{}
	snapMonth := map[int]time.Time{}
	for _, r := range aggRows {
		nw, _ := r.NetWorth.Float64()
		snapNW[r.SnapshotID] += nw
		snapMonth[r.SnapshotID] = r.Month
	}
	monthLatestSID := map[time.Time]int{}
	for sid, month := range snapMonth {
		if cur, ok := monthLatestSID[month]; !ok || sid > cur {
			monthLatestSID[month] = sid
		}
	}
	months := make([]time.Time, 0, len(monthLatestSID))
	for m := range monthLatestSID {
		months = append(months, m)
	}
	sort.Slice(months, func(i, j int) bool { return months[i].Before(months[j]) })
	out := make([]float64, 0, len(months))
	for _, m := range months {
		out = append(out, snapNW[monthLatestSID[m]])
	}
	return out
}

// rawMonthlyNetWorths ports the raw savings-rate input: net worth of the
// last-4 snapshots (by the snapshots-ordered-by-date tail).
func rawMonthlyNetWorths(rows []mergedRow, snapshots []SnapshotMeta) []float64 {
	nwBySnap := map[int]float64{}
	for _, r := range rows {
		nwBySnap[r.SnapshotID] += r.signedValue()
	}
	out := make([]float64, 0, len(snapshots))
	for _, s := range snapshots {
		out = append(out, nwBySnap[s.ID])
	}
	return out
}

// synthTotalsFromAggregates ports the no-investment tile-delta branch — one
// synthetic totals row per snapshot from the aggregate sums.
func synthTotalsFromAggregates(aggRows []AggregateRow, snapshotDate map[int]time.Time) []snapshotTotals {
	type acc struct{ assets, liabilities float64 }
	bySnap := map[int]*acc{}
	for _, r := range aggRows {
		a, ok := bySnap[r.SnapshotID]
		if !ok {
			a = &acc{}
			bySnap[r.SnapshotID] = a
		}
		ta, _ := r.TotalAssets.Float64()
		tl, _ := r.TotalLiabilities.Float64()
		a.assets += ta
		a.liabilities += tl
	}
	out := make([]snapshotTotals, 0, len(bySnap))
	for sid, a := range bySnap {
		out = append(out, snapshotTotals{
			SnapshotID:  sid,
			Date:        snapshotDate[sid],
			Assets:      a.assets,
			Liabilities: a.liabilities,
			NetWorth:    a.assets - a.liabilities,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.Before(out[j].Date)
		}
		return out[i].SnapshotID < out[j].SnapshotID
	})
	return out
}

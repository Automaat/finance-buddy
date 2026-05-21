// Package aggregates ports backend/app/services/aggregate_spec.py +
// snapshot_aggregates.py — pure aggregation math and the DB writeback that
// recomputes snapshot_aggregates from snapshot_values + accounts + assets.
package aggregates

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// AccountInput is the subset of Account columns the spec consumes.
type AccountInput struct {
	ID       int
	Owner    string
	Type     string
	Category string
}

// SnapshotValueInput is one row from snapshot_values.
type SnapshotValueInput struct {
	AccountID *int
	AssetID   *int
	Value     decimal.Decimal
}

// SnapshotInput identifies the snapshot whose aggregates are being computed.
type SnapshotInput struct {
	ID   int
	Date time.Time
}

// AllocationEntry is one row of the allocation_json array.
type AllocationEntry struct {
	Category string
	Value    decimal.Decimal
}

// AggregateRow is one precomputed row per (snapshot_id, owner).
type AggregateRow struct {
	SnapshotID       int
	Month            time.Time
	Owner            string
	TotalAssets      decimal.Decimal
	TotalLiabilities decimal.Decimal
	NetWorth         decimal.Decimal
	Allocation       []AllocationEntry
}

// ComputeAggregates is the pure-math port of aggregate_spec.compute_aggregates.
// Caller must filter accounts/assets to is_active=true rows before calling.
func ComputeAggregates(
	snap SnapshotInput,
	values []SnapshotValueInput,
	accounts []AccountInput,
	activeAssetIDs map[int]struct{},
) []AggregateRow {
	accountMap := map[int]AccountInput{}
	for _, a := range accounts {
		accountMap[a.ID] = a
	}

	type bucket struct {
		assets      decimal.Decimal
		liabilities decimal.Decimal
		alloc       map[string]decimal.Decimal
	}
	totals := map[string]*bucket{}
	getBucket := func(owner string) *bucket {
		if b, ok := totals[owner]; ok {
			return b
		}
		b := &bucket{alloc: map[string]decimal.Decimal{}}
		totals[owner] = b
		return b
	}

	for _, sv := range values {
		v := sv.Value
		switch {
		case sv.AssetID != nil:
			if _, active := activeAssetIDs[*sv.AssetID]; active {
				b := getBucket("Shared")
				b.assets = b.assets.Add(v)
			}
		case sv.AccountID != nil:
			acc, ok := accountMap[*sv.AccountID]
			if !ok {
				continue
			}
			b := getBucket(acc.Owner)
			if acc.Type == "asset" {
				b.assets = b.assets.Add(v)
				b.alloc[acc.Category] = b.alloc[acc.Category].Add(v)
			} else {
				b.liabilities = b.liabilities.Add(v)
			}
		}
	}

	month := firstOfMonth(snap.Date)
	if len(totals) == 0 {
		return []AggregateRow{{
			SnapshotID: snap.ID,
			Month:      month,
			Owner:      "Shared",
			Allocation: []AllocationEntry{},
		}}
	}

	owners := make([]string, 0, len(totals))
	for o := range totals {
		owners = append(owners, o)
	}
	sort.Strings(owners)
	out := make([]AggregateRow, 0, len(owners))
	for _, owner := range owners {
		b := totals[owner]
		cats := make([]string, 0, len(b.alloc))
		for c := range b.alloc {
			cats = append(cats, c)
		}
		sort.Strings(cats)
		alloc := make([]AllocationEntry, 0, len(cats))
		for _, c := range cats {
			alloc = append(alloc, AllocationEntry{Category: c, Value: b.alloc[c]})
		}
		out = append(out, AggregateRow{
			SnapshotID:       snap.ID,
			Month:            month,
			Owner:            owner,
			TotalAssets:      b.assets,
			TotalLiabilities: b.liabilities,
			NetWorth:         b.assets.Sub(b.liabilities),
			Allocation:       alloc,
		})
	}
	return out
}

func firstOfMonth(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
}

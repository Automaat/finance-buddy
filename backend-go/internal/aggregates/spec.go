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
// OwnerUserID is the owning household member; nil means jointly owned.
type AccountInput struct {
	ID          int
	OwnerUserID *int
	Type        string
	Category    string
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

// AggregateRow is one precomputed row per (snapshot_id, owner_user_id).
// OwnerUserID is nil for the jointly-owned ("Shared") bucket.
type AggregateRow struct {
	SnapshotID       int
	Month            time.Time
	OwnerUserID      *int
	TotalAssets      decimal.Decimal
	TotalLiabilities decimal.Decimal
	NetWorth         decimal.Decimal
	Allocation       []AllocationEntry
}

// ownerKey is a comparable map key for owner buckets. A nil owner_user_id
// (jointly owned / asset-table rows) maps to {shared: true}.
type ownerKey struct {
	id     int
	shared bool
}

func keyFor(ownerUserID *int) ownerKey {
	if ownerUserID == nil {
		return ownerKey{shared: true}
	}
	return ownerKey{id: *ownerUserID}
}

func (k ownerKey) ownerUserID() *int {
	if k.shared {
		return nil
	}
	id := k.id
	return &id
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
	totals := map[ownerKey]*bucket{}
	getBucket := func(k ownerKey) *bucket {
		if b, ok := totals[k]; ok {
			return b
		}
		b := &bucket{alloc: map[string]decimal.Decimal{}}
		totals[k] = b
		return b
	}

	for _, sv := range values {
		v := sv.Value
		switch {
		case sv.AssetID != nil:
			if _, active := activeAssetIDs[*sv.AssetID]; active {
				b := getBucket(ownerKey{shared: true})
				b.assets = b.assets.Add(v)
			}
		case sv.AccountID != nil:
			acc, ok := accountMap[*sv.AccountID]
			if !ok {
				continue
			}
			b := getBucket(keyFor(acc.OwnerUserID))
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
			SnapshotID:  snap.ID,
			Month:       month,
			OwnerUserID: nil,
			Allocation:  []AllocationEntry{},
		}}
	}

	keys := make([]ownerKey, 0, len(totals))
	for k := range totals {
		keys = append(keys, k)
	}
	// Deterministic order: the jointly-owned (Shared) bucket first, then by id.
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].shared != keys[j].shared {
			return keys[i].shared
		}
		return keys[i].id < keys[j].id
	})
	out := make([]AggregateRow, 0, len(keys))
	for _, k := range keys {
		b, ok := totals[k]
		if !ok || b == nil {
			continue
		}
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
			OwnerUserID:      k.ownerUserID(),
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

package dashboard

import (
	"sort"
	"time"
)

// investmentCategories — used by time series + the has-investment check.
var investmentCategories = map[string]struct{}{
	"stock": {}, "bond": {}, "fund": {}, "etf": {}, "gold": {}, "ppk": {},
}

// mergedRow is a snapshot_value joined to its asset (name) and account
// (type/category/owner/wrapper/purpose) plus the snapshot date — the Go
// equivalent of the pandas merged DataFrame row.
type mergedRow struct {
	SnapshotID int
	Date       time.Time
	AccountID  *int
	AssetID    *int
	AssetName  *string // non-nil only when the asset join hit an active asset
	Value      float64

	// account columns (nil when account_id is nil or account inactive)
	AccType     *string
	Category    *string
	OwnerUserID *int
	Wrapper     *string
	Purpose     *string
}

// signedValue mirrors the pandas vectorized sign rule:
//   - account row (account_id + type present) -> +value if asset, -value if liability
//   - else asset-table row (asset_id + joined name present) -> +value
//   - else 0
func (r mergedRow) signedValue() float64 {
	if r.AccountID != nil && r.AccType != nil {
		if *r.AccType == "asset" {
			return r.Value
		}
		return -r.Value
	}
	if r.AssetID != nil && r.AssetName != nil {
		return r.Value
	}
	return 0
}

// buildMergedRows joins snapshot_values to assets + accounts + snapshots,
// mirroring _build_merged_df / the raw-path merge. Accounts/assets include
// soft-deleted rows so historical snapshots keep their metadata after a
// delete (issue #394); current-state checks filter on Account.IsActive at
// the call site instead.
func buildMergedRows(
	values []SnapshotValue,
	snapshotDate map[int]time.Time,
	accounts map[int]Account,
	assetIDs map[int]struct{},
) []mergedRow {
	out := make([]mergedRow, 0, len(values))
	for _, v := range values {
		d, ok := snapshotDate[v.SnapshotID]
		if !ok {
			continue // snapshot missing — pandas inner-merges snapshots, dropping it
		}
		val, _ := v.Value.Float64()
		row := mergedRow{
			SnapshotID: v.SnapshotID,
			Date:       d,
			AccountID:  v.AccountID,
			AssetID:    v.AssetID,
			Value:      val,
		}
		if v.AssetID != nil {
			if _, known := assetIDs[*v.AssetID]; known {
				name := "asset" // any non-nil name marks "asset join hit"
				row.AssetName = &name
			}
		}
		if v.AccountID != nil {
			if acc, ok := accounts[*v.AccountID]; ok {
				t := acc.Type
				c := acc.Category
				p := acc.Purpose
				row.AccType = &t
				row.Category = &c
				row.OwnerUserID = acc.OwnerUserID
				row.Purpose = &p
				row.Wrapper = acc.AccountWrapper
			}
		}
		out = append(out, row)
	}
	return out
}

// snapshotTotals is the per-snapshot (assets, liabilities, net_worth) row.
type snapshotTotals struct {
	SnapshotID  int
	Date        time.Time
	Assets      float64
	Liabilities float64
	NetWorth    float64
}

// perSnapshotTotals ports metrics._per_snapshot_totals — aggregate assets +
// liabilities per snapshot, sorted by (date, snapshot_id).
func perSnapshotTotals(rows []mergedRow) []snapshotTotals {
	type acc struct {
		date        time.Time
		assets      float64
		liabilities float64
	}
	bySnap := map[int]*acc{}
	for _, r := range rows {
		a, ok := bySnap[r.SnapshotID]
		if !ok {
			a = &acc{date: r.Date}
			bySnap[r.SnapshotID] = a
		}
		assetTable := r.AssetID != nil && r.AssetName != nil
		accountAsset := r.AccountID != nil && r.AccType != nil && *r.AccType == "asset"
		accountLiab := r.AccountID != nil && r.AccType != nil && *r.AccType == "liability"
		if assetTable || accountAsset {
			a.assets += r.Value
		}
		if accountLiab {
			a.liabilities += r.Value
		}
	}
	out := make([]snapshotTotals, 0, len(bySnap))
	for sid, a := range bySnap {
		out = append(out, snapshotTotals{
			SnapshotID:  sid,
			Date:        a.date,
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

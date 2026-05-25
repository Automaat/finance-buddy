package aggregates

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func snap(t *testing.T, ymd string) SnapshotInput {
	t.Helper()
	d, err := time.Parse("2006-01-02", ymd)
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	return SnapshotInput{ID: 1, Date: d}
}

func TestComputeAggregates_EmptyValuesProducesSharedZeroRow(t *testing.T) {
	got := ComputeAggregates(snap(t, "2025-06-15"), nil, nil, nil)
	if len(got) != 1 {
		t.Fatalf("rows: want 1, got %d", len(got))
	}
	row := got[0]
	if row.OwnerUserID != nil {
		t.Errorf("OwnerUserID: want nil (Shared), got %v", *row.OwnerUserID)
	}
	if !row.TotalAssets.IsZero() || !row.TotalLiabilities.IsZero() || !row.NetWorth.IsZero() {
		t.Errorf("want all zero, got assets=%s liab=%s nw=%s",
			row.TotalAssets, row.TotalLiabilities, row.NetWorth)
	}
	if len(row.Allocation) != 0 {
		t.Errorf("Allocation: want empty, got %v", row.Allocation)
	}
	want := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	if !row.Month.Equal(want) {
		t.Errorf("Month: want %v (first of June), got %v", want, row.Month)
	}
}

func TestComputeAggregates_SingleOwnerAssetAndLiability(t *testing.T) {
	owner := 7
	accounts := []AccountInput{
		{ID: 1, OwnerUserID: &owner, Type: "asset", Category: "Bank"},
		{ID: 2, OwnerUserID: &owner, Type: "liability", Category: "Mortgage"},
	}
	values := []SnapshotValueInput{
		{AccountID: new(1), Value: dec("1000")},
		{AccountID: new(2), Value: dec("400")},
	}
	got := ComputeAggregates(snap(t, "2025-06-15"), values, accounts, nil)
	if len(got) != 1 {
		t.Fatalf("rows: want 1, got %d", len(got))
	}
	row := got[0]
	if row.OwnerUserID == nil || *row.OwnerUserID != owner {
		t.Errorf("OwnerUserID: want %d, got %v", owner, row.OwnerUserID)
	}
	if !row.TotalAssets.Equal(dec("1000")) {
		t.Errorf("TotalAssets: want 1000, got %s", row.TotalAssets)
	}
	if !row.TotalLiabilities.Equal(dec("400")) {
		t.Errorf("TotalLiabilities: want 400, got %s", row.TotalLiabilities)
	}
	if !row.NetWorth.Equal(dec("600")) {
		t.Errorf("NetWorth: want 600, got %s", row.NetWorth)
	}
	if len(row.Allocation) != 1 || row.Allocation[0].Category != "Bank" ||
		!row.Allocation[0].Value.Equal(dec("1000")) {
		t.Errorf("Allocation: want [{Bank, 1000}], got %v", row.Allocation)
	}
}

func TestComputeAggregates_MultiOwnerSharedFirstThenByID(t *testing.T) {
	o2, o5 := 2, 5
	accounts := []AccountInput{
		{ID: 1, OwnerUserID: &o5, Type: "asset", Category: "Bank"},
		{ID: 2, OwnerUserID: &o2, Type: "asset", Category: "Bank"},
		{ID: 3, OwnerUserID: nil, Type: "asset", Category: "Bank"}, // jointly owned
	}
	values := []SnapshotValueInput{
		{AccountID: new(1), Value: dec("100")},
		{AccountID: new(2), Value: dec("200")},
		{AccountID: new(3), Value: dec("300")},
	}
	got := ComputeAggregates(snap(t, "2025-06-15"), values, accounts, nil)
	if len(got) != 3 {
		t.Fatalf("rows: want 3, got %d", len(got))
	}
	if got[0].OwnerUserID != nil {
		t.Errorf("row[0] should be Shared, got owner=%v", *got[0].OwnerUserID)
	}
	if got[1].OwnerUserID == nil || *got[1].OwnerUserID != 2 {
		t.Errorf("row[1] should be owner 2, got %v", got[1].OwnerUserID)
	}
	if got[2].OwnerUserID == nil || *got[2].OwnerUserID != 5 {
		t.Errorf("row[2] should be owner 5, got %v", got[2].OwnerUserID)
	}
}

func TestComputeAggregates_AllocationSortedByCategory(t *testing.T) {
	owner := 1
	accounts := []AccountInput{
		{ID: 1, OwnerUserID: &owner, Type: "asset", Category: "Zeta"},
		{ID: 2, OwnerUserID: &owner, Type: "asset", Category: "Alpha"},
		{ID: 3, OwnerUserID: &owner, Type: "asset", Category: "Mu"},
	}
	values := []SnapshotValueInput{
		{AccountID: new(1), Value: dec("1")},
		{AccountID: new(2), Value: dec("2")},
		{AccountID: new(3), Value: dec("3")},
	}
	got := ComputeAggregates(snap(t, "2025-06-15"), values, accounts, nil)
	if len(got) != 1 || len(got[0].Allocation) != 3 {
		t.Fatalf("unexpected shape: %+v", got)
	}
	wantOrder := []string{"Alpha", "Mu", "Zeta"}
	for i, want := range wantOrder {
		if got[0].Allocation[i].Category != want {
			t.Errorf("Allocation[%d]: want %q, got %q", i, want, got[0].Allocation[i].Category)
		}
	}
}

func TestComputeAggregates_AllocationSumsBuckets(t *testing.T) {
	owner := 1
	accounts := []AccountInput{
		{ID: 1, OwnerUserID: &owner, Type: "asset", Category: "Bank"},
		{ID: 2, OwnerUserID: &owner, Type: "asset", Category: "Bank"},
	}
	values := []SnapshotValueInput{
		{AccountID: new(1), Value: dec("100")},
		{AccountID: new(2), Value: dec("250")},
	}
	got := ComputeAggregates(snap(t, "2025-06-15"), values, accounts, nil)
	if !got[0].Allocation[0].Value.Equal(dec("350")) {
		t.Errorf("Bank total: want 350, got %s", got[0].Allocation[0].Value)
	}
}

func TestComputeAggregates_LiabilityExcludedFromAllocation(t *testing.T) {
	owner := 1
	accounts := []AccountInput{
		{ID: 1, OwnerUserID: &owner, Type: "asset", Category: "Bank"},
		{ID: 2, OwnerUserID: &owner, Type: "liability", Category: "Mortgage"},
	}
	values := []SnapshotValueInput{
		{AccountID: new(1), Value: dec("100")},
		{AccountID: new(2), Value: dec("50")},
	}
	got := ComputeAggregates(snap(t, "2025-06-15"), values, accounts, nil)
	for _, e := range got[0].Allocation {
		if e.Category == "Mortgage" {
			t.Errorf("Allocation must exclude liability categories, got %v", got[0].Allocation)
		}
	}
}

func TestComputeAggregates_UnknownAccountSkipped(t *testing.T) {
	// Account 99 not in accounts list — simulates a soft-deleted row missing
	// from the snapshot's account map. Must be skipped, not panic.
	values := []SnapshotValueInput{{AccountID: new(99), Value: dec("999")}}
	got := ComputeAggregates(snap(t, "2025-06-15"), values, nil, nil)
	if len(got) != 1 || !got[0].TotalAssets.IsZero() {
		t.Errorf("want single zero-row, got %+v", got)
	}
}

func TestComputeAggregates_KnownAssetRoutesToSharedBucket(t *testing.T) {
	known := map[int]struct{}{42: {}}
	values := []SnapshotValueInput{{AssetID: new(42), Value: dec("500")}}
	got := ComputeAggregates(snap(t, "2025-06-15"), values, nil, known)
	if len(got) != 1 || got[0].OwnerUserID != nil {
		t.Fatalf("want single Shared row, got %+v", got)
	}
	if !got[0].TotalAssets.Equal(dec("500")) {
		t.Errorf("TotalAssets: want 500, got %s", got[0].TotalAssets)
	}
}

func TestComputeAggregates_UnknownAssetSkipped(t *testing.T) {
	// Unknown asset id (not in knownAssetIDs) must not flow into the bucket.
	values := []SnapshotValueInput{{AssetID: new(7), Value: dec("999")}}
	got := ComputeAggregates(snap(t, "2025-06-15"), values, nil, map[int]struct{}{})
	if len(got) != 1 || !got[0].TotalAssets.IsZero() {
		t.Errorf("unknown asset should be skipped, got %+v", got)
	}
}

func TestComputeAggregates_MonthNormalization(t *testing.T) {
	cases := []struct {
		in  string
		out time.Time
	}{
		{"2025-01-01", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"2025-01-31", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"2025-12-31", time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got := ComputeAggregates(snap(t, c.in), nil, nil, nil)
		if !got[0].Month.Equal(c.out) {
			t.Errorf("date %s: Month want %v, got %v", c.in, c.out, got[0].Month)
		}
	}
}

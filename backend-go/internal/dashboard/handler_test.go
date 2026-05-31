package dashboard

import (
	"net/url"
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseDateRange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		q        url.Values
		wantErr  bool
		wantFrom string
		wantTo   string
	}{
		{name: "empty"},
		{name: "from only", q: url.Values{"date_from": {"2024-01-01"}}, wantFrom: "2024-01-01"},
		{name: "to only", q: url.Values{"date_to": {"2024-12-31"}}, wantTo: "2024-12-31"},
		{
			name:     "both",
			q:        url.Values{"date_from": {"2024-01-01"}, "date_to": {"2024-12-31"}},
			wantFrom: "2024-01-01", wantTo: "2024-12-31",
		},
		{name: "bad from", q: url.Values{"date_from": {"not-a-date"}}, wantErr: true},
		{name: "bad to", q: url.Values{"date_to": {"2024-13-40"}}, wantErr: true},
		{
			name:    "reversed",
			q:       url.Values{"date_from": {"2024-06-01"}, "date_to": {"2024-01-01"}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDateRange(tc.q)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (range=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantFrom == "" && !got.from.IsZero() {
				t.Errorf("expected zero from, got %v", got.from)
			}
			if tc.wantFrom != "" && !got.from.Equal(date(tc.wantFrom)) {
				t.Errorf("from=%v want=%s", got.from, tc.wantFrom)
			}
			if tc.wantTo == "" && !got.to.IsZero() {
				t.Errorf("expected zero to, got %v", got.to)
			}
			if tc.wantTo != "" && !got.to.Equal(date(tc.wantTo)) {
				t.Errorf("to=%v want=%s", got.to, tc.wantTo)
			}
		})
	}
}

func TestApplyDateRangeFiltersTimeSeries(t *testing.T) {
	t.Parallel()

	res := result{
		NetWorthHistory: []netWorthPoint{
			{Date: date("2024-01-01"), Value: 10},
			{Date: date("2024-06-15"), Value: 20},
			{Date: date("2025-01-01"), Value: 30},
		},
		InvestmentTimeSeries: []timeSeriesPoint{
			{Date: date("2024-01-01"), Value: 1},
			{Date: date("2024-06-15"), Value: 2},
			{Date: date("2025-01-01"), Value: 3},
		},
		WrapperTimeSeries: map[string][]timeSeriesPoint{
			"IKE": {
				{Date: date("2024-01-01"), Value: 1},
				{Date: date("2025-01-01"), Value: 3},
			},
		},
		CategoryTimeSeries: map[string][]timeSeriesPoint{
			"stock": {
				{Date: date("2024-06-15"), Value: 2},
				{Date: date("2025-01-01"), Value: 3},
			},
		},
		CurrentNetWorth: 30,
		TotalAssets:     100,
	}

	applyDateRange(&res, dateRange{from: date("2024-02-01"), to: date("2024-12-31")})

	if got := len(res.NetWorthHistory); got != 1 {
		t.Errorf("net worth history len=%d want=1", got)
	}
	if !res.NetWorthHistory[0].Date.Equal(date("2024-06-15")) {
		t.Errorf("unexpected kept point: %v", res.NetWorthHistory[0].Date)
	}
	if got := len(res.InvestmentTimeSeries); got != 1 {
		t.Errorf("investment series len=%d want=1", got)
	}
	if got := len(res.WrapperTimeSeries["IKE"]); got != 0 {
		t.Errorf("IKE series len=%d want=0", got)
	}
	if got := len(res.CategoryTimeSeries["stock"]); got != 1 {
		t.Errorf("stock series len=%d want=1", got)
	}
	if res.CurrentNetWorth != 30 || res.TotalAssets != 100 {
		t.Errorf("snapshot tiles were mutated: nw=%v assets=%v", res.CurrentNetWorth, res.TotalAssets)
	}
}

func TestApplyDateRangePreservesLatestSnapshotDate(t *testing.T) {
	t.Parallel()

	d := date("2025-01-01")
	res := result{
		LatestSnapshotDate: &d,
		NetWorthHistory: []netWorthPoint{
			{Date: date("2024-01-01"), Value: 10},
			{Date: date("2025-01-01"), Value: 30},
		},
	}
	applyDateRange(&res, dateRange{from: date("2030-01-01"), to: date("2030-12-31")})

	if res.LatestSnapshotDate == nil || !res.LatestSnapshotDate.Equal(d) {
		t.Errorf("latest snapshot date was mutated by date-range filter: %v", res.LatestSnapshotDate)
	}
}

func TestToWireSerializesLatestSnapshotDate(t *testing.T) {
	t.Parallel()

	d := date("2025-03-09")
	withDate := toWire(result{LatestSnapshotDate: &d})
	if withDate.LatestSnapshotDate == nil {
		t.Fatal("expected latest_snapshot_date to be serialized")
	}
	if got := time.Time(*withDate.LatestSnapshotDate); !got.Equal(d) {
		t.Errorf("latest_snapshot_date=%v want=%v", got, d)
	}

	if got := toWire(result{}); got.LatestSnapshotDate != nil {
		t.Errorf("expected nil latest_snapshot_date when unset, got %v", got.LatestSnapshotDate)
	}
}

func TestApplyDateRangeNoBoundsIsNoop(t *testing.T) {
	t.Parallel()

	pts := []netWorthPoint{
		{Date: date("2024-01-01"), Value: 10},
		{Date: date("2025-01-01"), Value: 20},
	}
	res := result{NetWorthHistory: append([]netWorthPoint(nil), pts...)}
	applyDateRange(&res, dateRange{})
	if len(res.NetWorthHistory) != 2 {
		t.Fatalf("expected no-op, got len=%d", len(res.NetWorthHistory))
	}
}

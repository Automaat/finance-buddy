package allocation

import (
	"testing"

	"github.com/shopspring/decimal"
)

func pct(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func TestComputeDriftBalancedPortfolio(t *testing.T) {
	holdings := []CategoryAllocation{
		{Category: "stock", Value: 60000},
		{Category: "bond", Value: 30000},
		{Category: "gold", Value: 10000},
	}
	targets := []Target{
		{Category: "stock", TargetPct: pct(60)},
		{Category: "bond", TargetPct: pct(30)},
		{Category: "gold", TargetPct: pct(10)},
	}
	got := ComputeDrift(holdings, targets)
	if len(got.Scopes) != 1 {
		t.Fatalf("expected 1 scope, got %d", len(got.Scopes))
	}
	scope := got.Scopes[0]
	if !scope.HasCompleteTarget {
		t.Fatalf("expected complete targets")
	}
	for _, it := range scope.Items {
		if it.Severity != "ok" {
			t.Fatalf("category %q expected ok severity, got %q", it.Category, it.Severity)
		}
		if absFloat(it.RebalanceAmount) > 0.01 {
			t.Fatalf("category %q expected near-zero rebalance, got %v", it.Category, it.RebalanceAmount)
		}
	}
}

func TestComputeDriftWarningAboveThreshold(t *testing.T) {
	holdings := []CategoryAllocation{
		{Category: "stock", Value: 80000}, // 80% actual vs 60% target = +20pp drift
		{Category: "bond", Value: 20000},
	}
	targets := []Target{
		{Category: "stock", TargetPct: pct(60)},
		{Category: "bond", TargetPct: pct(40)},
	}
	got := ComputeDrift(holdings, targets)
	stock := findItem(t, got.Scopes[0].Items, "stock")
	if stock.Severity != "warning" {
		t.Fatalf("expected warning, got %q", stock.Severity)
	}
	if stock.RebalanceAmount > 0 {
		t.Fatalf("over-allocated stock should suggest selling (negative), got %v", stock.RebalanceAmount)
	}
}

func TestComputeDriftMissingTargetTagsHolding(t *testing.T) {
	holdings := []CategoryAllocation{
		{Category: "stock", Value: 50000},
		{Category: "crypto", Value: 5000},
	}
	targets := []Target{
		{Category: "stock", TargetPct: pct(100)},
	}
	got := ComputeDrift(holdings, targets)
	if len(got.Scopes[0].Items) != 2 {
		t.Fatalf("expected 2 items (stock target + untargeted crypto), got %d", len(got.Scopes[0].Items))
	}
	crypto := findItem(t, got.Scopes[0].Items, "crypto")
	if crypto.Severity != "missing_target" {
		t.Fatalf("untargeted holding should be missing_target, got %q", crypto.Severity)
	}
}

func TestComputeDriftPerOwnerScopes(t *testing.T) {
	o1, o2 := 1, 2
	holdings := []CategoryAllocation{
		{Category: "stock", OwnerUserID: &o1, Value: 30000},
		{Category: "bond", OwnerUserID: &o1, Value: 20000},
		{Category: "stock", OwnerUserID: &o2, Value: 40000},
	}
	targets := []Target{
		{Category: "stock", OwnerUserID: &o1, TargetPct: pct(60)},
		{Category: "bond", OwnerUserID: &o1, TargetPct: pct(40)},
		{Category: "stock", OwnerUserID: &o2, TargetPct: pct(100)},
	}
	got := ComputeDrift(holdings, targets)
	if len(got.Scopes) != 2 {
		t.Fatalf("expected 2 owner scopes, got %d", len(got.Scopes))
	}
}

func TestComputeDriftIncompleteTargets(t *testing.T) {
	holdings := []CategoryAllocation{
		{Category: "stock", Value: 50000},
		{Category: "bond", Value: 50000},
	}
	targets := []Target{
		{Category: "stock", TargetPct: pct(60)},
		{Category: "bond", TargetPct: pct(30)},
	}
	got := ComputeDrift(holdings, targets)
	if got.Scopes[0].HasCompleteTarget {
		t.Fatalf("expected incomplete (sum=90)")
	}
}

func TestComputeDriftRebalanceAmount(t *testing.T) {
	holdings := []CategoryAllocation{
		{Category: "stock", Value: 50000}, // 50% actual
		{Category: "bond", Value: 50000},  // 50% actual
	}
	targets := []Target{
		{Category: "stock", TargetPct: pct(70)},
		{Category: "bond", TargetPct: pct(30)},
	}
	got := ComputeDrift(holdings, targets)
	stock := findItem(t, got.Scopes[0].Items, "stock")
	// stock: 20pp short on 100k = 20k buy
	if absFloat(stock.RebalanceAmount-20000) > 0.5 {
		t.Fatalf("stock rebalance: want ~+20000, got %v", stock.RebalanceAmount)
	}
	bond := findItem(t, got.Scopes[0].Items, "bond")
	if absFloat(bond.RebalanceAmount+20000) > 0.5 {
		t.Fatalf("bond rebalance: want ~-20000, got %v", bond.RebalanceAmount)
	}
}

func TestComputeDriftEmptyNoTargetsNoScopes(t *testing.T) {
	got := ComputeDrift([]CategoryAllocation{{Category: "stock", Value: 1000}}, []Target{})
	if len(got.Scopes) != 0 {
		t.Fatalf("no targets configured should produce no scopes; got %d", len(got.Scopes))
	}
}

func findItem(t *testing.T, items []DriftItem, category string) DriftItem {
	t.Helper()
	for _, it := range items {
		if it.Category == category {
			return it
		}
	}
	t.Fatalf("category %q not in items", category)
	return DriftItem{}
}

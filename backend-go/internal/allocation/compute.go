package allocation

import (
	"sort"
)

// driftThresholdPP is the drift magnitude (percentage points) above which
// a category triggers the dashboard warning badge. Mirrors the threshold
// called out in the issue acceptance criteria.
const driftThresholdPP = 5.0

// CategoryAllocation is one (category, owner) holding from the latest
// snapshot. Value is in PLN.
type CategoryAllocation struct {
	Category    string
	OwnerUserID *int
	Value       float64
}

// DriftItem is one row of the actual-vs-target widget.
type DriftItem struct {
	Category          string
	OwnerUserID       *int
	CurrentValue      float64
	CurrentPercentage float64
	TargetPercentage  float64
	DriftPP           float64
	Severity          string // "ok" | "warning" | "missing_target"
	RebalanceAmount   float64
}

// DriftAnalysis is the dashboard widget payload, grouped by scope.
type DriftAnalysis struct {
	Scopes []DriftScope
}

// DriftScope is one owner_user_id bucket (nil = household).
type DriftScope struct {
	OwnerUserID       *int
	TotalValue        float64
	Items             []DriftItem
	TargetSumPct      float64
	HasCompleteTarget bool
}

// ComputeDrift bucket-sums the latest-snapshot holdings against the
// declared targets and tags each row with the per-scope drift severity.
//
// Behavior:
//   - For each (owner_user_id) scope that has any target row, render the
//     full target set as DriftItems. Holdings whose category isn't in the
//     target set show under that scope with target_pct=0 (so the user can
//     see they're holding something they didn't plan for).
//   - For each scope, HasCompleteTarget is true iff the configured targets
//     sum to exactly 100 (within 0.01 tolerance).
//   - Rebalance amount = (target_pct - current_pct) / 100 * total_value.
//     Positive = buy, negative = sell.
func ComputeDrift(holdings []CategoryAllocation, targets []Target) DriftAnalysis {
	type scopeKey struct {
		owner    int
		isShared bool
	}
	holdingsByScope := map[scopeKey]map[string]float64{}
	for _, h := range holdings {
		k := scopeKey{isShared: h.OwnerUserID == nil}
		if h.OwnerUserID != nil {
			k.owner = *h.OwnerUserID
		}
		if holdingsByScope[k] == nil {
			holdingsByScope[k] = map[string]float64{}
		}
		holdingsByScope[k][h.Category] += h.Value
	}

	targetsByScope := map[scopeKey][]Target{}
	scopeOwner := map[scopeKey]*int{}
	for _, t := range targets {
		k := scopeKey{isShared: t.OwnerUserID == nil}
		if t.OwnerUserID != nil {
			k.owner = *t.OwnerUserID
		}
		targetsByScope[k] = append(targetsByScope[k], t)
		owner := t.OwnerUserID
		scopeOwner[k] = owner
	}

	allKeys := map[scopeKey]struct{}{}
	for k := range targetsByScope {
		allKeys[k] = struct{}{}
	}
	for k := range holdingsByScope {
		if _, hasTarget := targetsByScope[k]; !hasTarget {
			continue
		}
		allKeys[k] = struct{}{}
	}

	keys := make([]scopeKey, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].isShared != keys[j].isShared {
			return keys[i].isShared
		}
		return keys[i].owner < keys[j].owner
	})

	out := DriftAnalysis{Scopes: make([]DriftScope, 0, len(keys))}
	for _, k := range keys {
		scope := buildScope(holdingsByScope[k], targetsByScope[k], scopeOwner[k])
		out.Scopes = append(out.Scopes, scope)
	}
	return out
}

func buildScope(holdings map[string]float64, targets []Target, owner *int) DriftScope {
	total := 0.0
	for _, v := range holdings {
		total += v
	}
	scope := DriftScope{OwnerUserID: owner, TotalValue: total}

	covered := map[string]struct{}{}
	for _, t := range targets {
		tgt, _ := t.TargetPct.Float64()
		scope.TargetSumPct += tgt
		current := holdings[t.Category]
		currentPct := 0.0
		if total > 0 {
			currentPct = current / total * 100
		}
		drift := currentPct - tgt
		rebalance := 0.0
		if total > 0 {
			rebalance = (tgt - currentPct) / 100 * total
		}
		scope.Items = append(scope.Items, DriftItem{
			Category:          t.Category,
			OwnerUserID:       owner,
			CurrentValue:      current,
			CurrentPercentage: currentPct,
			TargetPercentage:  tgt,
			DriftPP:           drift,
			Severity:          driftSeverity(drift),
			RebalanceAmount:   rebalance,
		})
		covered[t.Category] = struct{}{}
	}
	categories := make([]string, 0, len(holdings))
	for cat := range holdings {
		if _, ok := covered[cat]; ok {
			continue
		}
		categories = append(categories, cat)
	}
	sort.Strings(categories)
	for _, cat := range categories {
		current := holdings[cat]
		currentPct := 0.0
		if total > 0 {
			currentPct = current / total * 100
		}
		scope.Items = append(scope.Items, DriftItem{
			Category:          cat,
			OwnerUserID:       owner,
			CurrentValue:      current,
			CurrentPercentage: currentPct,
			TargetPercentage:  0,
			DriftPP:           currentPct,
			Severity:          "missing_target",
			RebalanceAmount:   -current,
		})
	}
	sort.SliceStable(scope.Items, func(i, j int) bool {
		return scope.Items[i].Category < scope.Items[j].Category
	})
	scope.HasCompleteTarget = absFloat(scope.TargetSumPct-100) < 0.01
	return scope
}

func driftSeverity(driftPP float64) string {
	if absFloat(driftPP) > driftThresholdPP {
		return "warning"
	}
	return "ok"
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

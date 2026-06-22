package bonds

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

// Valuator binds a Store + CPI source so accounts.Handler can override
// `current_value` for bond-category accounts with the live sum from the
// bond ledger. Mirrors holdings.Valuator's shape so the wiring in
// server.go is symmetric. Implements accounts.HoldingsValuator interface
// (any source of per-account PLN values fits the contract).
type Valuator struct {
	store  *Store
	cpi    CPILoader
	logger *slog.Logger
	now    func() time.Time
}

// NewValuator wires the store + CPI loader. cpi may be nil — value math
// then falls back to FirstYearRate per the bonds.calc fallback chain.
func NewValuator(store *Store, cpiStore CPILoader, logger *slog.Logger) *Valuator {
	if logger == nil {
		logger = slog.Default()
	}
	return &Valuator{store: store, cpi: cpiStore, logger: logger, now: time.Now}
}

// AccountValuesPLN walks every active bond, computes its current value
// via the same engine the handler uses, and sums per account_id. Bonds
// without an account_id are dropped (they have no tile to roll into) so
// the result is a sparse map of accounts that actually hold tracked
// bonds. Same contract as holdings.Valuator.AccountValuesPLN — both
// satisfy accounts.HoldingsValuator.
func (v *Valuator) AccountValuesPLN(ctx context.Context) (map[int]decimal.Decimal, error) {
	bonds, err := v.store.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(bonds) == 0 {
		return map[int]decimal.Decimal{}, nil
	}
	yoy, err := v.cpi.LoadYoYMap(ctx)
	if err != nil {
		v.logger.Warn("bonds.valuator: load annual cpi failed", "err", err)
	}
	monthly, err := v.cpi.LoadMonthlyYoYMap(ctx)
	if err != nil {
		v.logger.Warn("bonds.valuator: load monthly cpi failed", "err", err)
	}
	out := map[int]decimal.Decimal{}
	now := v.now()
	for i := range bonds {
		b := &bonds[i]
		if b.AccountID == nil {
			continue
		}
		cur := CurrentValue(b, yoy, monthly, now)
		out[*b.AccountID] = out[*b.AccountID].Add(cur)
	}
	return out, nil
}

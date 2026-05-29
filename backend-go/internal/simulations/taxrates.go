package simulations

import "github.com/Automaat/finance-buddy/backend-go/internal/rules"

// capitalGainsTaxRate is the Polish capital-gains tax ("podatek Belki"),
// a flat 19% on investment gains. Sourced from the centralized rules
// table (#545) — change there to update all callers.
var capitalGainsTaxRate = rules.Float64Or("capital_gains_tax_2026", 0.19)

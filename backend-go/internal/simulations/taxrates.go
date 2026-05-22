package simulations

// capitalGainsTaxRate is the Polish capital-gains tax ("podatek Belki"),
// a flat 19% on investment gains, unchanged since 2004. Defined once here
// so the brokerage and mortgage simulations share a single source instead
// of repeating the literal.
const capitalGainsTaxRate = 0.19

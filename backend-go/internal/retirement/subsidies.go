package retirement

import "github.com/shopspring/decimal"

// SubsidyConfig is the PPK government subsidy amount table for one year.
// Polish PPK pays a one-time welcome payment and an annual subsidy when the
// participant meets contribution thresholds. The amounts are set by
// regulation; thresholds shift with the minimum wage. Configurable per
// year by editing this file.
type SubsidyConfig struct {
	WelcomeAmount decimal.Decimal
	AnnualAmount  decimal.Decimal
}

// defaultSubsidy is the stable PPK regulation: 250 PLN welcome (one-time)
// and 240 PLN annual subsidy. Used for any year not explicitly listed in
// subsidiesByYear.
var defaultSubsidy = SubsidyConfig{
	WelcomeAmount: decimal.NewFromInt(250),
	AnnualAmount:  decimal.NewFromInt(240),
}

// subsidiesByYear holds year-specific overrides. Add an entry when
// regulation changes — defaultSubsidy applies for missing years.
var subsidiesByYear = map[int]SubsidyConfig{}

// SubsidyFor returns the configured subsidy amounts for a given year.
func SubsidyFor(year int) SubsidyConfig {
	if cfg, ok := subsidiesByYear[year]; ok {
		return cfg
	}
	return defaultSubsidy
}

package retirement

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestSubsidyForReturnsDefault(t *testing.T) {
	cfg := SubsidyFor(2030)
	if !cfg.WelcomeAmount.Equal(decimal.NewFromInt(250)) {
		t.Fatalf("welcome want 250, got %s", cfg.WelcomeAmount)
	}
	if !cfg.AnnualAmount.Equal(decimal.NewFromInt(240)) {
		t.Fatalf("annual want 240, got %s", cfg.AnnualAmount)
	}
}

func TestSubsidyForRespectsYearOverride(t *testing.T) {
	subsidiesByYear[1999] = SubsidyConfig{
		WelcomeAmount: decimal.NewFromInt(123),
		AnnualAmount:  decimal.NewFromInt(456),
	}
	t.Cleanup(func() { delete(subsidiesByYear, 1999) })

	cfg := SubsidyFor(1999)
	if !cfg.WelcomeAmount.Equal(decimal.NewFromInt(123)) {
		t.Fatalf("override welcome got %s", cfg.WelcomeAmount)
	}
	if !cfg.AnnualAmount.Equal(decimal.NewFromInt(456)) {
		t.Fatalf("override annual got %s", cfg.AnnualAmount)
	}
}

// Package bonds implements /api/bonds — Polish Treasury Bond tracking
// (EDO/COI/ROR/TOZ/DOS) with auto-computed current value from CPI + bond
// rules (year-1 fixed rate, year-2+ = CPI YoY + margin).
package bonds

import (
	"errors"
	"slices"
	"time"

	"github.com/shopspring/decimal"
)

// BondType is the issuer code printed on the bond certificate.
type BondType string

const (
	BondEDO BondType = "EDO" // 10y CPI-linked, capitalized
	BondCOI BondType = "COI" // 4y CPI-linked, interest paid out
	BondROR BondType = "ROR" // 1y floating
	BondTOZ BondType = "TOZ" // 3y WIBOR-linked
	BondDOS BondType = "DOS" // 2y fixed
)

// AllBondTypes enumerates the supported issuer codes.
var AllBondTypes = []BondType{BondEDO, BondCOI, BondROR, BondTOZ, BondDOS}

// IsValid reports whether t is a supported bond type.
func (t BondType) IsValid() bool {
	return slices.Contains(AllBondTypes, t)
}

// MaturityMonths returns the canonical bond tenor in months.
func (t BondType) MaturityMonths() int {
	switch t {
	case BondEDO:
		return 120
	case BondCOI:
		return 48
	case BondROR:
		return 12
	case BondTOZ:
		return 36
	case BondDOS:
		return 24
	}
	return 0
}

// TreasuryBond mirrors the treasury_bonds row.
//
// Money fields use shopspring/decimal so percent margins and PLN face values
// keep numeric precision across the JSON boundary. OwnerUserID is nullable
// for jointly-owned ("Shared") bonds, matching the household-shared account
// model elsewhere in the schema.
type TreasuryBond struct {
	ID            int
	Type          BondType
	Series        string
	FaceValue     decimal.Decimal
	PurchaseDate  time.Time
	OwnerUserID   *int
	FirstYearRate decimal.Decimal // annual %, e.g. 6.80
	Margin        decimal.Decimal // annual %, applied on top of CPI YoY from year 2+
	Capitalize    bool            // true: interest compounds; false: paid out yearly
	IsActive      bool
	CreatedAt     time.Time
}

// Sentinel errors mapped to HTTP status by the handler.
var (
	ErrNotFound = errors.New("treasury bond not found")
)

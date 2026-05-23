package transactions

// TransactionType is the discriminator on the transactions table. Stored
// as a varchar today; the canonical set lives here so handlers, retirement
// stats, and the API surface all share one source of truth.
//
// The names match the Polish PPK regulator's terminology rather than the
// generic accounting words in #390's original wording — existing rows in
// production use these values and a strict DB enum would otherwise reject
// them on insert.
type TransactionType string

const (
	TransactionTypeEmployee   TransactionType = "employee"
	TransactionTypeEmployer   TransactionType = "employer"
	TransactionTypeGovernment TransactionType = "government"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
)

// ValidTypes is the authoritative list of accepted transaction_type values.
// Order is stable so the frontend dropdown renders deterministically.
var ValidTypes = []TransactionType{
	TransactionTypeEmployee,
	TransactionTypeEmployer,
	TransactionTypeGovernment,
	TransactionTypeWithdrawal,
}

// LabelsPL maps each type to its Polish display label — used by /api/transactions/types
// so the frontend dropdown doesn't have to hardcode translations.
var LabelsPL = map[TransactionType]string{
	TransactionTypeEmployee:   "Wpłata pracownika",
	TransactionTypeEmployer:   "Wpłata pracodawcy",
	TransactionTypeGovernment: "Dopłata państwa",
	TransactionTypeWithdrawal: "Wypłata",
}

// IsValid reports whether s is one of the recognized values.
func IsValid(s string) bool {
	for _, t := range ValidTypes {
		if string(t) == s {
			return true
		}
	}
	return false
}

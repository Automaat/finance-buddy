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

// validTypes is the authoritative ordered list of accepted transaction_type
// values. Kept unexported + only reached through ValidTypes() to prevent
// other packages from reordering or appending at runtime.
var validTypes = []TransactionType{
	TransactionTypeEmployee,
	TransactionTypeEmployer,
	TransactionTypeGovernment,
	TransactionTypeWithdrawal,
}

// labelsPL maps each type to its Polish display label. Same encapsulation
// reason as validTypes — read via LabelPL().
var labelsPL = map[TransactionType]string{
	TransactionTypeEmployee:   "Wpłata pracownika",
	TransactionTypeEmployer:   "Wpłata pracodawcy",
	TransactionTypeGovernment: "Dopłata państwa",
	TransactionTypeWithdrawal: "Wypłata",
}

// ValidTypes returns a fresh copy of the canonical, ordered enum slice.
// Callers can mutate the returned slice without affecting the package's
// source of truth.
func ValidTypes() []TransactionType {
	out := make([]TransactionType, len(validTypes))
	copy(out, validTypes)
	return out
}

// LabelPL returns the Polish display label for t, or an empty string if
// t is not a recognized value.
func LabelPL(t TransactionType) string {
	return labelsPL[t]
}

// IsValid reports whether s is one of the recognized values.
func IsValid(s string) bool {
	for _, t := range validTypes {
		if string(t) == s {
			return true
		}
	}
	return false
}

package transactions

import "testing"

func TestIsValid(t *testing.T) {
	cases := map[string]bool{
		"employee":     true,
		"employer":     true,
		"government":   true,
		"withdrawal":   true,
		"":             false,
		"contribution": false, // listed in #390 but not in production use
		"random":       false,
	}
	for input, want := range cases {
		if got := IsValid(input); got != want {
			t.Errorf("IsValid(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestValidTypesAndLabelsCoverEachOther(t *testing.T) {
	types := ValidTypes()
	for _, tt := range types {
		if label := LabelPL(tt); label == "" {
			t.Errorf("LabelPL missing entry for %q", tt)
		}
	}
	// Reverse: every labelsPL entry must be in validTypes too.
	seen := make(map[TransactionType]struct{}, len(types))
	for _, tt := range types {
		seen[tt] = struct{}{}
	}
	for tt := range labelsPL {
		if _, ok := seen[tt]; !ok {
			t.Errorf("labelsPL has %q but ValidTypes() doesn't", tt)
		}
	}
}

func TestValidTypesReturnsCopy(t *testing.T) {
	a := ValidTypes()
	a[0] = "tampered"
	b := ValidTypes()
	if b[0] == "tampered" {
		t.Fatal("ValidTypes() returned a shared slice; mutation leaked")
	}
}

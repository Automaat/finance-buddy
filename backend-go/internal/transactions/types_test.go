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
	if len(ValidTypes) != len(LabelsPL) {
		t.Fatalf("ValidTypes (%d) and LabelsPL (%d) lengths diverged", len(ValidTypes), len(LabelsPL))
	}
	for _, tt := range ValidTypes {
		if _, ok := LabelsPL[tt]; !ok {
			t.Errorf("LabelsPL missing entry for %q", tt)
		}
	}
}

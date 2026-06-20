package validation

import "testing"

func TestParseOwnerCompanyDateFilter(t *testing.T) {
	got, vErr := ParseOwnerCompanyDateFilter(map[string][]string{
		"owner_user_id": {" 7 "},
		"company":       {" ACME "},
		"date_from":     {"2026-01-01"},
		"date_to":       {"2026-01-31"},
	})
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got.OwnerUserID == nil || *got.OwnerUserID != 7 {
		t.Fatalf("OwnerUserID = %v", got.OwnerUserID)
	}
	if got.Company == nil || *got.Company != "ACME" {
		t.Fatalf("Company = %v", got.Company)
	}
	if got.DateFrom == nil || got.DateFrom.Format("2006-01-02") != "2026-01-01" {
		t.Fatalf("DateFrom = %v", got.DateFrom)
	}
	if got.DateTo == nil || got.DateTo.Format("2006-01-02") != "2026-01-31" {
		t.Fatalf("DateTo = %v", got.DateTo)
	}
}

func TestParseOwnerCompanyDateFilterIgnoresMissingAndBlank(t *testing.T) {
	got, vErr := ParseOwnerCompanyDateFilter(map[string][]string{
		"owner_user_id": {" "},
		"company":       nil,
	})
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got.OwnerUserID != nil || got.Company != nil || got.DateFrom != nil || got.DateTo != nil {
		t.Fatalf("got = %#v", got)
	}
}

func TestParseOwnerCompanyDateFilterErrors(t *testing.T) {
	cases := []struct {
		name  string
		query map[string][]string
		field string
		msg   string
	}{
		{
			name:  "bad owner",
			query: map[string][]string{"owner_user_id": {"x"}},
			field: "owner_user_id",
			msg:   "must be an integer",
		},
		{
			name:  "bad date from",
			query: map[string][]string{"date_from": {"01-2026"}},
			field: "date_from",
			msg:   "must be YYYY-MM-DD",
		},
		{
			name:  "bad date to",
			query: map[string][]string{"date_to": {"2026-13-01"}},
			field: "date_to",
			msg:   "must be YYYY-MM-DD",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, vErr := ParseOwnerCompanyDateFilter(tc.query)
			if vErr == nil {
				t.Fatal("vErr is nil")
			}
			if vErr.Field != tc.field || vErr.Msg != tc.msg {
				t.Fatalf("vErr = %#v, want field=%q msg=%q", vErr, tc.field, tc.msg)
			}
		})
	}
}

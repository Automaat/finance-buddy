package validation

import (
	"strconv"
	"strings"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

// OwnerCompanyDateFilter is the shared shape used by list endpoints that
// accept owner_user_id, company, date_from, and date_to query parameters.
type OwnerCompanyDateFilter struct {
	OwnerUserID *int
	Company     *string
	DateFrom    *time.Time
	DateTo      *time.Time
}

// ParseOwnerCompanyDateFilter parses the common owner/company/date query
// filter set used by salary and bonus list endpoints.
func ParseOwnerCompanyDateFilter(q map[string][]string) (OwnerCompanyDateFilter, *httputil.ValidationError) {
	f := OwnerCompanyDateFilter{}
	if v := firstQueryValue(q["owner_user_id"]); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, &httputil.ValidationError{Field: "owner_user_id", Msg: "must be an integer"}
		}
		f.OwnerUserID = &n
	}
	if v := firstQueryValue(q["company"]); v != "" {
		s := v
		f.Company = &s
	}
	if v := firstQueryValue(q["date_from"]); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, &httputil.ValidationError{Field: "date_from", Msg: "must be YYYY-MM-DD"}
		}
		f.DateFrom = &t
	}
	if v := firstQueryValue(q["date_to"]); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, &httputil.ValidationError{Field: "date_to", Msg: "must be YYYY-MM-DD"}
		}
		f.DateTo = &t
	}
	return f, nil
}

func firstQueryValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

package validation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

func TestRequiredTrimmedString(t *testing.T) {
	raw := map[string]json.RawMessage{"name": json.RawMessage(`"  Cash  "`)}
	got, vErr := RequiredTrimmedString(raw, "name", "Field required", "Name cannot be empty")
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != "Cash" {
		t.Fatalf("got = %q", got)
	}

	_, vErr = RequiredTrimmedString(map[string]json.RawMessage{}, "name", "Field required", "Name cannot be empty")
	requireValidation(t, vErr, "name", "Field required")

	_, vErr = RequiredTrimmedString(
		map[string]json.RawMessage{"name": json.RawMessage(`"  "`)},
		"name",
		"Field required",
		"Name cannot be empty",
	)
	requireValidation(t, vErr, "name", "Name cannot be empty")
}

func TestOptionalTrimmedString(t *testing.T) {
	got, vErr := OptionalTrimmedString(
		map[string]json.RawMessage{"name": json.RawMessage(`"  Cash  "`)},
		"name",
		"Name cannot be empty",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != "Cash" {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalTrimmedString(map[string]json.RawMessage{}, "name", "Name cannot be empty")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalTrimmedString(
		map[string]json.RawMessage{"name": json.RawMessage(`null`)},
		"name",
		"Name cannot be empty",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalTrimmedString(
		map[string]json.RawMessage{"name": json.RawMessage(`"  "`)},
		"name",
		"Name cannot be empty",
	)
	if got != nil {
		t.Fatalf("got = %v", got)
	}
	requireValidation(t, vErr, "name", "Name cannot be empty")
}

func TestOptionalString(t *testing.T) {
	got, vErr := OptionalString(
		map[string]json.RawMessage{"notes": json.RawMessage(`"  keep spaces  "`)},
		"notes",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != "  keep spaces  " {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalString(map[string]json.RawMessage{}, "notes")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalString(
		map[string]json.RawMessage{"notes": json.RawMessage(`null`)},
		"notes",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalString(
		map[string]json.RawMessage{"notes": json.RawMessage(`""`)},
		"notes",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != "" {
		t.Fatalf("got = %v", got)
	}

	_, vErr = OptionalString(
		map[string]json.RawMessage{"notes": json.RawMessage(`7`)},
		"notes",
	)
	requireValidation(t, vErr, "notes", "must be a string")
}

func TestRequiredString(t *testing.T) {
	got, vErr := RequiredString(
		map[string]json.RawMessage{"frequency": json.RawMessage(`" monthly "`)},
		"frequency",
		"required",
		"cannot be empty",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != " monthly " {
		t.Fatalf("got = %q", got)
	}

	_, vErr = RequiredString(map[string]json.RawMessage{}, "frequency", "required", "cannot be empty")
	requireValidation(t, vErr, "frequency", "required")

	_, vErr = RequiredString(
		map[string]json.RawMessage{"frequency": json.RawMessage(`null`)},
		"frequency",
		"required",
		"cannot be empty",
	)
	requireValidation(t, vErr, "frequency", "required")

	_, vErr = RequiredString(
		map[string]json.RawMessage{"frequency": json.RawMessage(`7`)},
		"frequency",
		"required",
		"cannot be empty",
	)
	requireValidation(t, vErr, "frequency", "must be a string")

	_, vErr = RequiredString(
		map[string]json.RawMessage{"frequency": json.RawMessage(`""`)},
		"frequency",
		"required",
		"cannot be empty",
	)
	requireValidation(t, vErr, "frequency", "cannot be empty")
}

func TestRequiredDate(t *testing.T) {
	got, vErr := RequiredDate(map[string]json.RawMessage{"date": json.RawMessage(`"2026-05-26"`)}, "date")
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got.Format("2006-01-02") != "2026-05-26" {
		t.Fatalf("got = %s", got.Format("2006-01-02"))
	}

	for _, tc := range []struct {
		name string
		raw  map[string]json.RawMessage
		msg  string
	}{
		{name: "missing", raw: map[string]json.RawMessage{}, msg: "Field required"},
		{name: "null", raw: map[string]json.RawMessage{"date": json.RawMessage(`null`)}, msg: "Field required"},
		{
			name: "invalid format",
			raw:  map[string]json.RawMessage{"date": json.RawMessage(`"26/05/2026"`)},
			msg:  "must be YYYY-MM-DD",
		},
		{name: "non-string", raw: map[string]json.RawMessage{"date": json.RawMessage(`20260526`)}, msg: "must be a string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, vErr = RequiredDate(tc.raw, "date")
			requireValidation(t, vErr, "date", tc.msg)
		})
	}
}

func TestRequiredDateWithMissingMessage(t *testing.T) {
	got, vErr := RequiredDateWithMissingMessage(map[string]json.RawMessage{
		"date": json.RawMessage(`"2026-05-26"`),
	}, "date", "required")
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got.Format("2006-01-02") != "2026-05-26" {
		t.Fatalf("got = %s", got.Format("2006-01-02"))
	}

	_, vErr = RequiredDateWithMissingMessage(map[string]json.RawMessage{}, "date", "required")
	requireValidation(t, vErr, "date", "required")

	_, vErr = RequiredDateWithMissingMessage(
		map[string]json.RawMessage{"date": json.RawMessage(`null`)},
		"date",
		"required",
	)
	requireValidation(t, vErr, "date", "required")

	_, vErr = RequiredDateWithMissingMessage(
		map[string]json.RawMessage{"date": json.RawMessage(`"26/05/2026"`)},
		"date",
		"required",
	)
	requireValidation(t, vErr, "date", "must be YYYY-MM-DD")

	_, vErr = RequiredDateWithMissingMessage(
		map[string]json.RawMessage{"date": json.RawMessage(`20260526`)},
		"date",
		"required",
	)
	requireValidation(t, vErr, "date", "must be a string")
}

func TestRequiredNonEmptyDate(t *testing.T) {
	got, vErr := RequiredNonEmptyDate(
		map[string]json.RawMessage{"date": json.RawMessage(`"2026-05-26"`)},
		"date",
		"required",
		"cannot be empty",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got.Format("2006-01-02") != "2026-05-26" {
		t.Fatalf("got = %s", got.Format("2006-01-02"))
	}

	_, vErr = RequiredNonEmptyDate(
		map[string]json.RawMessage{"date": json.RawMessage(`""`)},
		"date",
		"required",
		"cannot be empty",
	)
	requireValidation(t, vErr, "date", "cannot be empty")

	_, vErr = RequiredNonEmptyDate(
		map[string]json.RawMessage{"date": json.RawMessage(`"26/05/2026"`)},
		"date",
		"required",
		"cannot be empty",
	)
	requireValidation(t, vErr, "date", "must be YYYY-MM-DD")
}

func TestOptionalDate(t *testing.T) {
	got, vErr := OptionalDate(map[string]json.RawMessage{"date": json.RawMessage(`"2026-05-26"`)}, "date")
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || got.Format("2006-01-02") != "2026-05-26" {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalDate(map[string]json.RawMessage{}, "date")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalDate(map[string]json.RawMessage{"date": json.RawMessage(`null`)}, "date")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalDate(map[string]json.RawMessage{"date": json.RawMessage(`"26/05/2026"`)}, "date")
	requireValidation(t, vErr, "date", "must be YYYY-MM-DD")

	_, vErr = OptionalDate(map[string]json.RawMessage{"date": json.RawMessage(`20260526`)}, "date")
	requireValidation(t, vErr, "date", "must be a string")
}

func TestRequiredDateNotFuture(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) }

	got, vErr := RequiredDateNotFuture(
		map[string]json.RawMessage{"date": json.RawMessage(`"2026-06-20"`)},
		"date",
		now,
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got.Format("2006-01-02") != "2026-06-20" {
		t.Fatalf("got = %s", got.Format("2006-01-02"))
	}

	_, vErr = RequiredDateNotFuture(
		map[string]json.RawMessage{"date": json.RawMessage(`"2026-06-21"`)},
		"date",
		now,
	)
	requireValidation(t, vErr, "date", "Date cannot be in the future")

	_, vErr = RequiredDateNotFuture(
		map[string]json.RawMessage{"date": json.RawMessage(`"06/20/2026"`)},
		"date",
		now,
	)
	requireValidation(t, vErr, "date", "must be YYYY-MM-DD")
}

func TestRequiredDateNotFutureWithMessage(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) }

	_, vErr := RequiredDateNotFutureWithMessage(
		map[string]json.RawMessage{"start_date": json.RawMessage(`"2026-06-21"`)},
		"start_date",
		now,
		"Start date cannot be in the future",
	)
	requireValidation(t, vErr, "start_date", "Start date cannot be in the future")
}

func TestOptionalDateNotFuture(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) }

	got, vErr := OptionalDateNotFuture(
		map[string]json.RawMessage{"date": json.RawMessage(`"2026-06-20"`)},
		"date",
		now,
		"Date cannot be in the future",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || got.Format("2006-01-02") != "2026-06-20" {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalDateNotFuture(map[string]json.RawMessage{}, "date", now, "Date cannot be in the future")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalDateNotFuture(
		map[string]json.RawMessage{"date": json.RawMessage(`null`)},
		"date",
		now,
		"Date cannot be in the future",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalDateNotFuture(
		map[string]json.RawMessage{"date": json.RawMessage(`"2026-06-21"`)},
		"date",
		now,
		"Date cannot be in the future",
	)
	requireValidation(t, vErr, "date", "Date cannot be in the future")

	_, vErr = OptionalDateNotFuture(
		map[string]json.RawMessage{"date": json.RawMessage(`"06/20/2026"`)},
		"date",
		now,
		"Date cannot be in the future",
	)
	requireValidation(t, vErr, "date", "must be YYYY-MM-DD")

	_, vErr = OptionalDateNotFuture(
		map[string]json.RawMessage{"date": json.RawMessage(`20260620`)},
		"date",
		now,
		"Date cannot be in the future",
	)
	requireValidation(t, vErr, "date", "must be a string")
}

func TestOptionalBool(t *testing.T) {
	got, vErr := OptionalBool(
		map[string]json.RawMessage{"active": json.RawMessage(`true`)},
		"active",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != true {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalBool(map[string]json.RawMessage{}, "active")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalBool(map[string]json.RawMessage{"active": json.RawMessage(`null`)}, "active")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalBool(map[string]json.RawMessage{"active": json.RawMessage(`"true"`)}, "active")
	requireValidation(t, vErr, "active", "must be a boolean")
}

func TestOptionalBoolDefault(t *testing.T) {
	got, vErr := OptionalBoolDefault(map[string]json.RawMessage{}, "active", true)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != true {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalBoolDefault(map[string]json.RawMessage{"active": json.RawMessage(`null`)}, "active", true)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != true {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalBoolDefault(map[string]json.RawMessage{"active": json.RawMessage(`false`)}, "active", true)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != false {
		t.Fatalf("got = %v", got)
	}

	_, vErr = OptionalBoolDefault(map[string]json.RawMessage{"active": json.RawMessage(`"false"`)}, "active", true)
	requireValidation(t, vErr, "active", "must be a boolean")
}

func TestOptionalInt(t *testing.T) {
	got, vErr := OptionalInt(
		map[string]json.RawMessage{"owner_user_id": json.RawMessage(`7`)},
		"owner_user_id",
		"must be an integer",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != 7 {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalInt(map[string]json.RawMessage{}, "owner_user_id", "must be an integer")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalInt(
		map[string]json.RawMessage{"owner_user_id": json.RawMessage(`null`)},
		"owner_user_id",
		"must be an integer",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalInt(
		map[string]json.RawMessage{"owner_user_id": json.RawMessage(`"7"`)},
		"owner_user_id",
		"must be an integer",
	)
	requireValidation(t, vErr, "owner_user_id", "must be an integer")

	_, vErr = OptionalInt(
		map[string]json.RawMessage{"owner_user_id": json.RawMessage(`"7"`)},
		"owner_user_id",
		"must be integer or null",
	)
	requireValidation(t, vErr, "owner_user_id", "must be integer or null")
}

func TestRequiredIntOrNull(t *testing.T) {
	got, vErr := RequiredIntOrNull(map[string]json.RawMessage{"owner_user_id": json.RawMessage(`7`)}, "owner_user_id")
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != 7 {
		t.Fatalf("got = %v", got)
	}

	got, vErr = RequiredIntOrNull(map[string]json.RawMessage{"owner_user_id": json.RawMessage(`null`)}, "owner_user_id")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = RequiredIntOrNull(map[string]json.RawMessage{"owner_user_id": json.RawMessage(`"7"`)}, "owner_user_id")
	requireValidation(t, vErr, "owner_user_id", "must be an integer")
}

func TestRequiredInt(t *testing.T) {
	got, vErr := RequiredInt(
		map[string]json.RawMessage{"account_id": json.RawMessage(`7`)},
		"account_id",
		"required",
		"must be integer",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != 7 {
		t.Fatalf("got = %d", got)
	}

	_, vErr = RequiredInt(map[string]json.RawMessage{}, "account_id", "required", "must be integer")
	requireValidation(t, vErr, "account_id", "required")

	_, vErr = RequiredInt(
		map[string]json.RawMessage{"account_id": json.RawMessage(`null`)},
		"account_id",
		"required",
		"must be integer",
	)
	requireValidation(t, vErr, "account_id", "required")

	_, vErr = RequiredInt(
		map[string]json.RawMessage{"account_id": json.RawMessage(`"7"`)},
		"account_id",
		"required",
		"must be integer",
	)
	requireValidation(t, vErr, "account_id", "must be integer")
}

func TestRequiredIntRange(t *testing.T) {
	got, vErr := RequiredIntRange(
		map[string]json.RawMessage{"year": json.RawMessage(`2026`)},
		"year",
		2000,
		2030,
		"Year must be between 2000 and 2030",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != 2026 {
		t.Fatalf("got = %d", got)
	}

	_, vErr = RequiredIntRange(map[string]json.RawMessage{}, "year", 2000, 2030, "Year must be between 2000 and 2030")
	requireValidation(t, vErr, "year", "Field required")

	_, vErr = RequiredIntRange(
		map[string]json.RawMessage{"year": json.RawMessage(`null`)},
		"year",
		2000,
		2030,
		"Year must be between 2000 and 2030",
	)
	requireValidation(t, vErr, "year", "Field required")

	_, vErr = RequiredIntRange(
		map[string]json.RawMessage{"year": json.RawMessage(`"2026"`)},
		"year",
		2000,
		2030,
		"Year must be between 2000 and 2030",
	)
	requireValidation(t, vErr, "year", "must be an integer")

	_, vErr = RequiredIntRange(
		map[string]json.RawMessage{"year": json.RawMessage(`1999`)},
		"year",
		2000,
		2030,
		"Year must be between 2000 and 2030",
	)
	requireValidation(t, vErr, "year", "Year must be between 2000 and 2030")
}

func TestRequiredPositiveInt(t *testing.T) {
	got, vErr := RequiredPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`7`)},
		"shares",
		"Shares must be greater than 0",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != 7 {
		t.Fatalf("got = %d", got)
	}

	_, vErr = RequiredPositiveInt(map[string]json.RawMessage{}, "shares", "Shares must be greater than 0")
	requireValidation(t, vErr, "shares", "Field required")

	_, vErr = RequiredPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`null`)},
		"shares",
		"Shares must be greater than 0",
	)
	requireValidation(t, vErr, "shares", "Field required")

	_, vErr = RequiredPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`"7"`)},
		"shares",
		"Shares must be greater than 0",
	)
	requireValidation(t, vErr, "shares", "must be an integer")

	_, vErr = RequiredPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`0`)},
		"shares",
		"Shares must be greater than 0",
	)
	requireValidation(t, vErr, "shares", "Shares must be greater than 0")

	_, vErr = RequiredPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`-1`)},
		"shares",
		"Shares must be greater than 0",
	)
	requireValidation(t, vErr, "shares", "Shares must be greater than 0")
}

func TestOptionalPositiveInt(t *testing.T) {
	got, vErr := OptionalPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`7`)},
		"shares",
		"Shares must be greater than 0",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != 7 {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalPositiveInt(map[string]json.RawMessage{}, "shares", "Shares must be greater than 0")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`null`)},
		"shares",
		"Shares must be greater than 0",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`"7"`)},
		"shares",
		"Shares must be greater than 0",
	)
	requireValidation(t, vErr, "shares", "must be an integer")

	_, vErr = OptionalPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`0`)},
		"shares",
		"Shares must be greater than 0",
	)
	requireValidation(t, vErr, "shares", "Shares must be greater than 0")

	_, vErr = OptionalPositiveInt(
		map[string]json.RawMessage{"shares": json.RawMessage(`-1`)},
		"shares",
		"Shares must be greater than 0",
	)
	requireValidation(t, vErr, "shares", "Shares must be greater than 0")
}

func TestOptionalNonNegativeInt(t *testing.T) {
	got, vErr := OptionalNonNegativeInt(
		map[string]json.RawMessage{"months": json.RawMessage(`0`)},
		"months",
		"Months must be non-negative",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != 0 {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalNonNegativeInt(map[string]json.RawMessage{}, "months", "Months must be non-negative")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalNonNegativeInt(
		map[string]json.RawMessage{"months": json.RawMessage(`null`)},
		"months",
		"Months must be non-negative",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalNonNegativeInt(
		map[string]json.RawMessage{"months": json.RawMessage(`"0"`)},
		"months",
		"Months must be non-negative",
	)
	requireValidation(t, vErr, "months", "must be an integer")

	_, vErr = OptionalNonNegativeInt(
		map[string]json.RawMessage{"months": json.RawMessage(`-1`)},
		"months",
		"Months must be non-negative",
	)
	requireValidation(t, vErr, "months", "Months must be non-negative")
}

func TestOptionalNonNegativeIntDefault(t *testing.T) {
	got, vErr := OptionalNonNegativeIntDefault(
		map[string]json.RawMessage{},
		"months",
		"Months must be non-negative",
		12,
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != 12 {
		t.Fatalf("got = %d", got)
	}

	got, vErr = OptionalNonNegativeIntDefault(
		map[string]json.RawMessage{"months": json.RawMessage(`null`)},
		"months",
		"Months must be non-negative",
		12,
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != 12 {
		t.Fatalf("got = %d", got)
	}

	got, vErr = OptionalNonNegativeIntDefault(
		map[string]json.RawMessage{"months": json.RawMessage(`3`)},
		"months",
		"Months must be non-negative",
		12,
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != 3 {
		t.Fatalf("got = %d", got)
	}

	_, vErr = OptionalNonNegativeIntDefault(
		map[string]json.RawMessage{"months": json.RawMessage(`-1`)},
		"months",
		"Months must be non-negative",
		12,
	)
	requireValidation(t, vErr, "months", "Months must be non-negative")
}

func TestRequiredEnumString(t *testing.T) {
	allowed := map[string]struct{}{"employment": {}, "b2b": {}}

	got, vErr := RequiredEnumString(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`"employment"`)},
		"contract_type",
		allowed,
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != "employment" {
		t.Fatalf("got = %q", got)
	}

	_, vErr = RequiredEnumString(map[string]json.RawMessage{}, "contract_type", allowed)
	requireValidation(t, vErr, "contract_type", "Field required")

	_, vErr = RequiredEnumString(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`null`)},
		"contract_type",
		allowed,
	)
	requireValidation(t, vErr, "contract_type", "Field required")

	_, vErr = RequiredEnumString(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`7`)},
		"contract_type",
		allowed,
	)
	requireValidation(t, vErr, "contract_type", "must be a string")

	_, vErr = RequiredEnumString(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`"mandate"`)},
		"contract_type",
		allowed,
	)
	requireValidation(t, vErr, "contract_type", `invalid value "mandate"`)
}

func TestOptionalEnumString(t *testing.T) {
	allowed := map[string]struct{}{"employment": {}, "b2b": {}}
	got, vErr := OptionalEnumString(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`"b2b"`)},
		"contract_type",
		allowed,
		"employment",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != "b2b" {
		t.Fatalf("got = %q", got)
	}

	got, vErr = OptionalEnumString(map[string]json.RawMessage{}, "contract_type", allowed, "employment")
	if vErr != nil || got != "employment" {
		t.Fatalf("got = %q vErr = %#v", got, vErr)
	}

	got, vErr = OptionalEnumString(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`null`)},
		"contract_type",
		allowed,
		"employment",
	)
	if vErr != nil || got != "employment" {
		t.Fatalf("got = %q vErr = %#v", got, vErr)
	}

	_, vErr = OptionalEnumString(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`7`)},
		"contract_type",
		allowed,
		"employment",
	)
	requireValidation(t, vErr, "contract_type", "must be a string")

	_, vErr = OptionalEnumString(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`"mandate"`)},
		"contract_type",
		allowed,
		"employment",
	)
	requireValidation(t, vErr, "contract_type", `invalid value "mandate"`)
}

func TestOptionalEnumStringPtr(t *testing.T) {
	allowed := map[string]struct{}{"employment": {}, "b2b": {}}
	got, vErr := OptionalEnumStringPtr(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`"b2b"`)},
		"contract_type",
		allowed,
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != "b2b" {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalEnumStringPtr(map[string]json.RawMessage{}, "contract_type", allowed)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalEnumStringPtr(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`null`)},
		"contract_type",
		allowed,
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalEnumStringPtr(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`7`)},
		"contract_type",
		allowed,
	)
	requireValidation(t, vErr, "contract_type", "must be a string")

	_, vErr = OptionalEnumStringPtr(
		map[string]json.RawMessage{"contract_type": json.RawMessage(`"mandate"`)},
		"contract_type",
		allowed,
	)
	requireValidation(t, vErr, "contract_type", `invalid value "mandate"`)
}

func TestOptionalUpperTrimmedEnumString(t *testing.T) {
	allowed := map[string]struct{}{"PLN": {}, "USD": {}}
	got, vErr := OptionalUpperTrimmedEnumString(
		map[string]json.RawMessage{"currency": json.RawMessage(`" usd "`)},
		"currency",
		allowed,
		"PLN",
		"invalid currency",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != "USD" {
		t.Fatalf("got = %q", got)
	}

	got, vErr = OptionalUpperTrimmedEnumString(
		map[string]json.RawMessage{},
		"currency",
		allowed,
		"PLN",
		"invalid currency",
	)
	if vErr != nil || got != "PLN" {
		t.Fatalf("got = %q vErr = %#v", got, vErr)
	}

	got, vErr = OptionalUpperTrimmedEnumString(
		map[string]json.RawMessage{"currency": json.RawMessage(`null`)},
		"currency",
		allowed,
		"PLN",
		"invalid currency",
	)
	if vErr != nil || got != "PLN" {
		t.Fatalf("got = %q vErr = %#v", got, vErr)
	}

	_, vErr = OptionalUpperTrimmedEnumString(
		map[string]json.RawMessage{"currency": json.RawMessage(`7`)},
		"currency",
		allowed,
		"PLN",
		"invalid currency",
	)
	requireValidation(t, vErr, "currency", "must be a string")

	_, vErr = OptionalUpperTrimmedEnumString(
		map[string]json.RawMessage{"currency": json.RawMessage(`"EUR"`)},
		"currency",
		allowed,
		"PLN",
		"invalid currency",
	)
	requireValidation(t, vErr, "currency", "invalid currency")
}

func TestOptionalUpperTrimmedEnumStringPtr(t *testing.T) {
	allowed := map[string]struct{}{"PLN": {}, "USD": {}}
	got, vErr := OptionalUpperTrimmedEnumStringPtr(
		map[string]json.RawMessage{"currency": json.RawMessage(`" pln "`)},
		"currency",
		allowed,
		"invalid currency",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != "PLN" {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalUpperTrimmedEnumStringPtr(
		map[string]json.RawMessage{},
		"currency",
		allowed,
		"invalid currency",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalUpperTrimmedEnumStringPtr(
		map[string]json.RawMessage{"currency": json.RawMessage(`null`)},
		"currency",
		allowed,
		"invalid currency",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalUpperTrimmedEnumStringPtr(
		map[string]json.RawMessage{"currency": json.RawMessage(`7`)},
		"currency",
		allowed,
		"invalid currency",
	)
	requireValidation(t, vErr, "currency", "must be a string")

	_, vErr = OptionalUpperTrimmedEnumStringPtr(
		map[string]json.RawMessage{"currency": json.RawMessage(`"EUR"`)},
		"currency",
		allowed,
		"invalid currency",
	)
	requireValidation(t, vErr, "currency", "invalid currency")
}

func TestRequiredDecimal(t *testing.T) {
	got, vErr := RequiredDecimal(
		map[string]json.RawMessage{"amount": json.RawMessage(`123.45`)},
		"amount",
		"required",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if !got.Equal(decimal.RequireFromString("123.45")) {
		t.Fatalf("got = %s", got)
	}

	_, vErr = RequiredDecimal(map[string]json.RawMessage{}, "amount", "required")
	requireValidation(t, vErr, "amount", "required")

	_, vErr = RequiredDecimal(
		map[string]json.RawMessage{"amount": json.RawMessage(`"123.45"`)},
		"amount",
		"required",
	)
	requireValidation(t, vErr, "amount", "must be a number")
}

func TestRequiredPositiveDecimal(t *testing.T) {
	got, vErr := RequiredPositiveDecimal(
		map[string]json.RawMessage{"amount": json.RawMessage(`123.45`)},
		"amount",
		"Field required",
		"Amount must be greater than 0",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if !got.Equal(decimal.RequireFromString("123.45")) {
		t.Fatalf("got = %s", got)
	}

	_, vErr = RequiredPositiveDecimal(
		map[string]json.RawMessage{},
		"amount",
		"Field required",
		"Amount must be greater than 0",
	)
	requireValidation(t, vErr, "amount", "Field required")

	_, vErr = RequiredPositiveDecimal(
		map[string]json.RawMessage{"amount": json.RawMessage(`0`)},
		"amount",
		"Field required",
		"Amount must be greater than 0",
	)
	requireValidation(t, vErr, "amount", "Amount must be greater than 0")
}

func TestRequiredNonNegativeDecimal(t *testing.T) {
	got, vErr := RequiredNonNegativeDecimal(
		map[string]json.RawMessage{"amount": json.RawMessage(`0`)},
		"amount",
		"Field required",
		"Amount must be non-negative",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if !got.Equal(decimal.Zero) {
		t.Fatalf("got = %s", got)
	}

	_, vErr = RequiredNonNegativeDecimal(
		map[string]json.RawMessage{},
		"amount",
		"Field required",
		"Amount must be non-negative",
	)
	requireValidation(t, vErr, "amount", "Field required")

	_, vErr = RequiredNonNegativeDecimal(
		map[string]json.RawMessage{"amount": json.RawMessage(`null`)},
		"amount",
		"Field required",
		"Amount must be non-negative",
	)
	requireValidation(t, vErr, "amount", "Field required")

	_, vErr = RequiredNonNegativeDecimal(
		map[string]json.RawMessage{"amount": json.RawMessage(`-0.01`)},
		"amount",
		"Field required",
		"Amount must be non-negative",
	)
	requireValidation(t, vErr, "amount", "Amount must be non-negative")

	_, vErr = RequiredNonNegativeDecimal(
		map[string]json.RawMessage{"amount": json.RawMessage(`"0"`)},
		"amount",
		"Field required",
		"Amount must be non-negative",
	)
	requireValidation(t, vErr, "amount", "must be a number")
}

func TestOptionalDecimal(t *testing.T) {
	got, vErr := OptionalDecimal(
		map[string]json.RawMessage{"fee": json.RawMessage(`1.25`)},
		"fee",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || !got.Equal(decimal.RequireFromString("1.25")) {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalDecimal(map[string]json.RawMessage{}, "fee")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalDecimal(map[string]json.RawMessage{"fee": json.RawMessage(`null`)}, "fee")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalDecimal(map[string]json.RawMessage{"fee": json.RawMessage(`"1.25"`)}, "fee")
	requireValidation(t, vErr, "fee", "must be a number")
}

func TestOptionalPositiveDecimal(t *testing.T) {
	got, vErr := OptionalPositiveDecimal(
		map[string]json.RawMessage{"fee": json.RawMessage(`1.25`)},
		"fee",
		"Fee must be greater than 0",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || !got.Equal(decimal.RequireFromString("1.25")) {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalPositiveDecimal(
		map[string]json.RawMessage{},
		"fee",
		"Fee must be greater than 0",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalPositiveDecimal(
		map[string]json.RawMessage{"fee": json.RawMessage(`null`)},
		"fee",
		"Fee must be greater than 0",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalPositiveDecimal(
		map[string]json.RawMessage{"fee": json.RawMessage(`0`)},
		"fee",
		"Fee must be greater than 0",
	)
	requireValidation(t, vErr, "fee", "Fee must be greater than 0")

	_, vErr = OptionalPositiveDecimal(
		map[string]json.RawMessage{"fee": json.RawMessage(`"1.25"`)},
		"fee",
		"Fee must be greater than 0",
	)
	requireValidation(t, vErr, "fee", "must be a number")
}

func TestOptionalNonNegativeDecimal(t *testing.T) {
	got, vErr := OptionalNonNegativeDecimal(
		map[string]json.RawMessage{"fee": json.RawMessage(`0`)},
		"fee",
		"Fee must be non-negative",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || !got.Equal(decimal.Zero) {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalNonNegativeDecimal(
		map[string]json.RawMessage{},
		"fee",
		"Fee must be non-negative",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalNonNegativeDecimal(
		map[string]json.RawMessage{"fee": json.RawMessage(`null`)},
		"fee",
		"Fee must be non-negative",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalNonNegativeDecimal(
		map[string]json.RawMessage{"fee": json.RawMessage(`-0.01`)},
		"fee",
		"Fee must be non-negative",
	)
	requireValidation(t, vErr, "fee", "Fee must be non-negative")

	_, vErr = OptionalNonNegativeDecimal(
		map[string]json.RawMessage{"fee": json.RawMessage(`"0"`)},
		"fee",
		"Fee must be non-negative",
	)
	requireValidation(t, vErr, "fee", "must be a number")
}

func TestRequiredDecimalStringOrNumber(t *testing.T) {
	got, vErr := RequiredDecimalStringOrNumber(
		map[string]json.RawMessage{"amount": json.RawMessage(`"123.45"`)},
		"amount",
		"required",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if !got.Equal(decimal.RequireFromString("123.45")) {
		t.Fatalf("got = %s", got)
	}

	_, vErr = RequiredDecimalStringOrNumber(map[string]json.RawMessage{}, "amount", "required")
	requireValidation(t, vErr, "amount", "required")

	_, vErr = RequiredDecimalStringOrNumber(
		map[string]json.RawMessage{"amount": json.RawMessage(`"abc"`)},
		"amount",
		"required",
	)
	requireValidation(t, vErr, "amount", "must be a number")
}

func TestOptionalDecimalStringOrNumber(t *testing.T) {
	got, vErr := OptionalDecimalStringOrNumber(
		map[string]json.RawMessage{"fee": json.RawMessage(`"1.25"`)},
		"fee",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || !got.Equal(decimal.RequireFromString("1.25")) {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalDecimalStringOrNumber(map[string]json.RawMessage{}, "fee")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = OptionalDecimalStringOrNumber(map[string]json.RawMessage{"fee": json.RawMessage(`"abc"`)}, "fee")
	requireValidation(t, vErr, "fee", "must be a number")
}

func requireValidation(t *testing.T, got *httputil.ValidationError, field, msg string) {
	t.Helper()
	if got == nil {
		t.Fatal("vErr is nil")
	}
	if got.Field != field || got.Msg != msg {
		t.Fatalf("vErr = %#v, want field=%q msg=%q", got, field, msg)
	}
}

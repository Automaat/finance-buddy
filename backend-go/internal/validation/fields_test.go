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

func TestRequiredDate(t *testing.T) {
	got, vErr := RequiredDate(map[string]json.RawMessage{"date": json.RawMessage(`"2026-05-26"`)}, "date")
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got.Format("2006-01-02") != "2026-05-26" {
		t.Fatalf("got = %s", got.Format("2006-01-02"))
	}

	_, vErr = RequiredDate(map[string]json.RawMessage{"date": json.RawMessage(`"26/05/2026"`)}, "date")
	requireValidation(t, vErr, "date", "must be YYYY-MM-DD")

	_, vErr = RequiredDate(map[string]json.RawMessage{"date": json.RawMessage(`20260526`)}, "date")
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

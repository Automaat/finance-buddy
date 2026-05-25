package debtpayments

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func rawJSON(s string) json.RawMessage { return json.RawMessage(s) }

func TestIsNull(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{`null`, true},
		{`  null  `, true},
		{`"null"`, false},
		{`0`, false},
		{`""`, false},
	}
	for _, c := range cases {
		if got := isNull(rawJSON(c.in)); got != c.want {
			t.Errorf("isNull(%q): want %v, got %v", c.in, c.want, got)
		}
	}
}

func TestRequireAmount(t *testing.T) {
	t.Run("rejects missing", func(t *testing.T) {
		_, vErr := requireAmount(map[string]json.RawMessage{})
		if vErr == nil || vErr.Msg != "Field required" {
			t.Errorf("want Field required, got %+v", vErr)
		}
	})
	t.Run("rejects null", func(t *testing.T) {
		_, vErr := requireAmount(map[string]json.RawMessage{"amount": rawJSON(`null`)})
		if vErr == nil {
			t.Fatal("want error on null")
		}
	})
	t.Run("rejects garbage", func(t *testing.T) {
		_, vErr := requireAmount(map[string]json.RawMessage{"amount": rawJSON(`"abc"`)})
		if vErr == nil || vErr.Msg != "must be a number" {
			t.Errorf("want must be a number, got %+v", vErr)
		}
	})
	t.Run("rejects zero", func(t *testing.T) {
		_, vErr := requireAmount(map[string]json.RawMessage{"amount": rawJSON(`0`)})
		if vErr == nil || vErr.Msg != "Amount must be greater than 0" {
			t.Errorf("want positive-amount error, got %+v", vErr)
		}
	})
	t.Run("rejects negative", func(t *testing.T) {
		_, vErr := requireAmount(map[string]json.RawMessage{"amount": rawJSON(`-5`)})
		if vErr == nil {
			t.Fatal("want error on negative")
		}
	})
	t.Run("accepts positive integer", func(t *testing.T) {
		d, vErr := requireAmount(map[string]json.RawMessage{"amount": rawJSON(`100`)})
		if vErr != nil {
			t.Fatalf("unexpected err: %+v", vErr)
		}
		if !d.Equal(dec("100")) {
			t.Errorf("want 100, got %s", d)
		}
	})
	t.Run("accepts positive decimal", func(t *testing.T) {
		d, vErr := requireAmount(map[string]json.RawMessage{"amount": rawJSON(`123.45`)})
		if vErr != nil {
			t.Fatalf("unexpected err: %+v", vErr)
		}
		if !d.Equal(dec("123.45")) {
			t.Errorf("want 123.45, got %s", d)
		}
	})
}

func TestRequireOwnerUserID(t *testing.T) {
	t.Run("missing key required", func(t *testing.T) {
		_, vErr := requireOwnerUserID(map[string]json.RawMessage{})
		if vErr == nil || vErr.Msg != "Field required" {
			t.Errorf("want Field required, got %+v", vErr)
		}
	})
	t.Run("null yields nil", func(t *testing.T) {
		got, vErr := requireOwnerUserID(map[string]json.RawMessage{"owner_user_id": rawJSON(`null`)})
		if vErr != nil {
			t.Fatalf("unexpected err: %+v", vErr)
		}
		if got != nil {
			t.Errorf("want nil, got %d", *got)
		}
	})
	t.Run("garbage rejected", func(t *testing.T) {
		_, vErr := requireOwnerUserID(map[string]json.RawMessage{"owner_user_id": rawJSON(`"x"`)})
		if vErr == nil {
			t.Fatal("want error on non-int")
		}
	})
	t.Run("integer accepted", func(t *testing.T) {
		got, vErr := requireOwnerUserID(map[string]json.RawMessage{"owner_user_id": rawJSON(`7`)})
		if vErr != nil {
			t.Fatalf("unexpected err: %+v", vErr)
		}
		if got == nil || *got != 7 {
			t.Errorf("want 7, got %v", got)
		}
	})
}

func TestRequireDateNotFuture(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		_, vErr := requireDateNotFuture(map[string]json.RawMessage{})
		if vErr == nil {
			t.Fatal("want error")
		}
	})
	t.Run("wrong format", func(t *testing.T) {
		_, vErr := requireDateNotFuture(map[string]json.RawMessage{"date": rawJSON(`"06/15/2025"`)})
		if vErr == nil || vErr.Msg != "must be YYYY-MM-DD" {
			t.Errorf("want must be YYYY-MM-DD, got %+v", vErr)
		}
	})
	t.Run("future rejected", func(t *testing.T) {
		// Far-future date avoids the UTC-midnight race a "today+1" would have
		// when the test straddles the day boundary.
		_, vErr := requireDateNotFuture(map[string]json.RawMessage{"date": rawJSON(`"3000-01-01"`)})
		if vErr == nil || vErr.Msg != "Date cannot be in the future" {
			t.Errorf("want future-rejection error, got %+v", vErr)
		}
	})
	t.Run("today accepted", func(t *testing.T) {
		today := time.Now().UTC().Format("2006-01-02")
		_, vErr := requireDateNotFuture(map[string]json.RawMessage{"date": rawJSON(`"` + today + `"`)})
		if vErr != nil {
			t.Errorf("today should be accepted, got %+v", vErr)
		}
	})
	t.Run("past accepted", func(t *testing.T) {
		_, vErr := requireDateNotFuture(map[string]json.RawMessage{"date": rawJSON(`"2020-01-01"`)})
		if vErr != nil {
			t.Errorf("past should be accepted, got %+v", vErr)
		}
	})
}

func TestBuildCreateRequest(t *testing.T) {
	t.Run("missing amount", func(t *testing.T) {
		_, vErr := buildCreateRequest(map[string]json.RawMessage{})
		if vErr == nil || vErr.Field != "amount" {
			t.Errorf("want amount field error, got %+v", vErr)
		}
	})
	t.Run("amount fails first", func(t *testing.T) {
		_, vErr := buildCreateRequest(map[string]json.RawMessage{
			"amount":        rawJSON(`-1`),
			"date":          rawJSON(`"3000-01-01"`),
			"owner_user_id": rawJSON(`"x"`),
		})
		if vErr == nil || vErr.Field != "amount" {
			t.Errorf("amount must fail first, got %+v", vErr)
		}
	})
	t.Run("full valid", func(t *testing.T) {
		got, vErr := buildCreateRequest(map[string]json.RawMessage{
			"amount":        rawJSON(`100`),
			"date":          rawJSON(`"2024-01-01"`),
			"owner_user_id": rawJSON(`5`),
		})
		if vErr != nil {
			t.Fatalf("unexpected err: %+v", vErr)
		}
		if !got.Amount.Equal(dec("100")) {
			t.Errorf("Amount: want 100, got %s", got.Amount)
		}
		want := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		if !got.Date.Equal(want) {
			t.Errorf("Date: want %v, got %v", want, got.Date)
		}
		if got.OwnerUserID == nil || *got.OwnerUserID != 5 {
			t.Errorf("OwnerUserID: want 5, got %v", got.OwnerUserID)
		}
	})
	t.Run("null owner allowed", func(t *testing.T) {
		got, vErr := buildCreateRequest(map[string]json.RawMessage{
			"amount":        rawJSON(`100`),
			"date":          rawJSON(`"2024-01-01"`),
			"owner_user_id": rawJSON(`null`),
		})
		if vErr != nil {
			t.Fatalf("unexpected err: %+v", vErr)
		}
		if got.OwnerUserID != nil {
			t.Errorf("OwnerUserID: want nil (Shared), got %d", *got.OwnerUserID)
		}
	})
}

func TestToResponse(t *testing.T) {
	owner := 3
	created := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	p := DebtPayment{
		ID: 7, AccountID: 42, Amount: dec("250.50"),
		Date:        time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		OwnerUserID: &owner,
		CreatedAt:   created,
	}
	r := toResponse(p, "Mortgage A")
	if r.ID != 7 || r.AccountID != 42 || r.AccountName != "Mortgage A" {
		t.Errorf("identity fields wrong: %+v", r)
	}
	if r.Amount != 250.5 {
		t.Errorf("Amount: want 250.5, got %v", r.Amount)
	}
	if r.OwnerUserID == nil || *r.OwnerUserID != 3 {
		t.Errorf("OwnerUserID: want 3, got %v", r.OwnerUserID)
	}
}

func TestIsoDateMarshal(t *testing.T) {
	d := isoDate(time.Date(2025, 1, 5, 14, 30, 0, 0, time.UTC))
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `"2025-01-05"` {
		t.Errorf("want \"2025-01-05\", got %s", got)
	}
}

func TestIsoNaiveMarshalStripsTimezone(t *testing.T) {
	// FixedZone instead of LoadLocation: keeps the test independent of the
	// container's zoneinfo/tzdata availability.
	loc := time.FixedZone("CET", 3600)
	tz := time.Date(2025, 1, 5, 14, 30, 0, 0, loc)
	got, err := json.Marshal(isoNaive(tz))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Naive (no tz suffix), preserves wall clock without offset.
	if !strings.HasPrefix(string(got), `"2025-01-05T14:30:00`) {
		t.Errorf("want prefix \"2025-01-05T14:30:00, got %s", got)
	}
	// Time portion (after T, before closing quote) must have no offset marker.
	body := strings.TrimSuffix(strings.TrimPrefix(string(got), `"`), `"`)
	timePart := body[strings.Index(body, "T")+1:]
	if strings.ContainsAny(timePart, "Z+-") {
		t.Errorf("want no tz marker in time portion, got %s", timePart)
	}
}

func TestPyFloatMarshal(t *testing.T) {
	cases := []struct {
		in   pyFloat
		want string
	}{
		{0, "0.0"},
		{1, "1.0"},
		{1.5, "1.5"},
		{-2.25, "-2.25"},
		{100, "100.0"},
	}
	for _, c := range cases {
		got, err := json.Marshal(c.in)
		if err != nil {
			t.Fatalf("marshal %v: %v", c.in, err)
		}
		if string(got) != c.want {
			t.Errorf("Marshal(%v): want %s, got %s", c.in, c.want, got)
		}
	}
}

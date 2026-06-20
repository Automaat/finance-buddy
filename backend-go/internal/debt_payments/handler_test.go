package debtpayments

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func rawJSON(s string) json.RawMessage { return json.RawMessage(s) }

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
	d := wire.IsoDate(time.Date(2025, 1, 5, 14, 30, 0, 0, time.UTC))
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
	got, err := json.Marshal(wire.IsoNaive(tz))
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
		in   wire.PyFloat
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

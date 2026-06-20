package dbutil

import (
	"reflect"
	"testing"
)

func TestWhereBuilder(t *testing.T) {
	where := NewWhereBuilder("is_active = true", "a.is_active = true")

	where.Add("owner_user_id = $%d", 7)
	where.Add("date >= $%d", "2026-01-01")

	if got, want := where.SQL(), "is_active = true AND a.is_active = true AND owner_user_id = $1 AND date >= $2"; got != want {
		t.Fatalf("SQL() = %q, want %q", got, want)
	}
	if got, want := where.Args(), []any{7, "2026-01-01"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Args() = %#v, want %#v", got, want)
	}
}

func TestWhereBuilderCopiesInitialConditions(t *testing.T) {
	conditions := []string{"is_active = true"}
	where := NewWhereBuilder(conditions...)
	conditions[0] = "mutated = true"

	if got, want := where.SQL(), "is_active = true"; got != want {
		t.Fatalf("SQL() = %q, want %q", got, want)
	}
}

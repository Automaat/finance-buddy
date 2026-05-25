package dbutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
)

var errSentinel = errors.New("sentinel")

func TestMapErrNil(t *testing.T) {
	if err := MapErr(nil, errSentinel, "x"); err != nil {
		t.Fatalf("nil → %v", err)
	}
}

func TestMapErrNoRows(t *testing.T) {
	err := MapErr(pgx.ErrNoRows, errSentinel, "load thing")
	if !errors.Is(err, errSentinel) {
		t.Fatalf("expected sentinel, got %v", err)
	}
}

func TestMapErrWrapsOthers(t *testing.T) {
	inner := errors.New("boom")
	err := MapErr(inner, errSentinel, "load thing")
	if !errors.Is(err, inner) {
		t.Fatalf("inner not wrapped: %v", err)
	}
	if got := err.Error(); got != "load thing: boom" {
		t.Fatalf("wrap text = %q", got)
	}
}

func TestMapErrPreservesWrappedNoRows(t *testing.T) {
	// A pgx error wrapped via fmt.Errorf should still map to the sentinel.
	wrapped := fmt.Errorf("scan: %w", pgx.ErrNoRows)
	err := MapErr(wrapped, errSentinel, "load thing")
	if !errors.Is(err, errSentinel) {
		t.Fatalf("expected sentinel for wrapped ErrNoRows, got %v", err)
	}
}

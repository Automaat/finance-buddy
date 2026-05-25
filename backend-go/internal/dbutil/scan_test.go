package dbutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestCollectRows(t *testing.T) {
	rows := &fakeRows{values: []int{1, 2}}
	got, err := CollectRows(rows, func(row pgx.Row) (int, error) {
		var n int
		return n, row.Scan(&n)
	}, "scan thing", "iterate things")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !rows.closed {
		t.Fatal("rows not closed")
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestCollectRowsScanError(t *testing.T) {
	inner := errors.New("bad scan")
	rows := &fakeRows{values: []int{1}, scanErr: inner}
	_, err := CollectRows(rows, func(row pgx.Row) (int, error) {
		var n int
		return n, row.Scan(&n)
	}, "scan thing", "iterate things")
	if !rows.closed {
		t.Fatal("rows not closed")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("inner not wrapped: %v", err)
	}
	if got := err.Error(); got != "scan thing: bad scan" {
		t.Fatalf("err = %q", got)
	}
}

func TestCollectRowsIterError(t *testing.T) {
	inner := errors.New("bad iter")
	rows := &fakeRows{iterErr: inner}
	_, err := CollectRows(rows, func(row pgx.Row) (int, error) {
		var n int
		return n, row.Scan(&n)
	}, "scan thing", "iterate things")
	if !rows.closed {
		t.Fatal("rows not closed")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("inner not wrapped: %v", err)
	}
	if got := err.Error(); got != "iterate things: bad iter" {
		t.Fatalf("err = %q", got)
	}
}

type fakeRows struct {
	values  []int
	index   int
	closed  bool
	scanErr error
	iterErr error
}

func (r *fakeRows) Close() {
	r.closed = true
}

func (r *fakeRows) Err() error {
	return r.iterErr
}

func (r *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakeRows) Next() bool {
	if r.index >= len(r.values) {
		return false
	}
	r.index++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	if len(dest) != 1 {
		return fmt.Errorf("dest len = %d", len(dest))
	}
	p, ok := dest[0].(*int)
	if !ok {
		return fmt.Errorf("dest type %T", dest[0])
	}
	*p = r.values[r.index-1]
	return nil
}

func (r *fakeRows) Values() ([]any, error) {
	return []any{r.values[r.index-1]}, nil
}

func (r *fakeRows) RawValues() [][]byte {
	return nil
}

func (r *fakeRows) Conn() *pgx.Conn {
	return nil
}

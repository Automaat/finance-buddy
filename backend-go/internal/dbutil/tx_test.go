package dbutil

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var errFakeTxUnsupported = errors.New("fake tx method unsupported")

func TestWithTxCommits(t *testing.T) {
	tx := &fakeTx{}
	db := &fakeBeginner{tx: tx}
	called := false

	err := WithTx(t.Context(), db, "begin thing", "commit thing", func(got pgx.Tx) error {
		called = true
		if got != tx {
			t.Fatalf("tx = %p, want %p", got, tx)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !called {
		t.Fatal("fn not called")
	}
	if !tx.committed {
		t.Fatal("tx not committed")
	}
	if tx.rolledBack {
		t.Fatal("tx rolled back after commit")
	}
}

func TestWithTxWrapsBeginError(t *testing.T) {
	inner := errors.New("down")
	db := &fakeBeginner{beginErr: inner}

	err := WithTx(t.Context(), db, "begin thing", "commit thing", func(pgx.Tx) error {
		t.Fatal("fn called")
		return nil
	})

	if err == nil {
		t.Fatal("err nil")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("inner not wrapped: %v", err)
	}
	if got := err.Error(); got != "begin thing: down" {
		t.Fatalf("err = %q", got)
	}
}

func TestWithTxRollsBackFnError(t *testing.T) {
	inner := errors.New("bad write")
	tx := &fakeTx{}
	db := &fakeBeginner{tx: tx}

	err := WithTx(t.Context(), db, "begin thing", "commit thing", func(pgx.Tx) error {
		return inner
	})

	if !errors.Is(err, inner) {
		t.Fatalf("inner not returned: %v", err)
	}
	if tx.committed {
		t.Fatal("tx committed")
	}
	if !tx.rolledBack {
		t.Fatal("tx not rolled back")
	}
}

func TestWithTxRollsBackCommitError(t *testing.T) {
	inner := errors.New("commit lost")
	tx := &fakeTx{commitErr: inner}
	db := &fakeBeginner{tx: tx}

	err := WithTx(t.Context(), db, "begin thing", "commit thing", func(pgx.Tx) error {
		return nil
	})

	if err == nil {
		t.Fatal("err nil")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("inner not wrapped: %v", err)
	}
	if got := err.Error(); got != "commit thing: commit lost" {
		t.Fatalf("err = %q", got)
	}
	if !tx.rolledBack {
		t.Fatal("tx not rolled back after commit error")
	}
}

type fakeBeginner struct {
	tx       pgx.Tx
	beginErr error
}

func (b *fakeBeginner) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	if b.beginErr != nil {
		return nil, b.beginErr
	}
	return b.tx, nil
}

type fakeTx struct {
	committed  bool
	rolledBack bool
	commitErr  error
}

func (tx *fakeTx) Begin(context.Context) (pgx.Tx, error) {
	return tx, nil
}

func (tx *fakeTx) Commit(context.Context) error {
	tx.committed = true
	return tx.commitErr
}

func (tx *fakeTx) Rollback(context.Context) error {
	tx.rolledBack = true
	return nil
}

func (tx *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (tx *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *fakeTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errFakeTxUnsupported
}

func (tx *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (tx *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errFakeTxUnsupported
}

func (tx *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (tx *fakeTx) Conn() *pgx.Conn {
	return nil
}

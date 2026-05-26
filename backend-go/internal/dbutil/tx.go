package dbutil

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type txBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// WithTx runs fn inside a pgx transaction, rolling back on every non-commit
// path and preserving caller-provided begin/commit error prefixes.
func WithTx(ctx context.Context, db txBeginner, beginPrefix, commitPrefix string, fn func(pgx.Tx) error) error {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%s: %w", beginPrefix, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: %w", commitPrefix, err)
	}
	committed = true
	return nil
}

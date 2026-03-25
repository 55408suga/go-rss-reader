package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey string

const txKey ctxKey = "pg_tx"

// PgTransactionManager implements usecase.TransactionManager using pgxpool.
type PgTransactionManager struct {
	pool *pgxpool.Pool
}

// NewPgTransactionManager creates a new PgTransactionManager.
func NewPgTransactionManager(pool *pgxpool.Pool) *PgTransactionManager {
	return &PgTransactionManager{pool: pool}
}

// WithTransaction executes fn within a database transaction.
// The transaction is committed if fn returns nil, or rolled back otherwise.
func (tm *PgTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	txCtx := context.WithValue(ctx, txKey, tx)
	if err := fn(txCtx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// TxFromContext extracts the pgx.Tx from context if present.
// Returns nil if no transaction is active.
func TxFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey).(pgx.Tx)
	return tx
}

package postgres

import (
	"context"
	"errors"
	"log/slog"
	applogger "rss_reader/internal/applog"
	"rss_reader/internal/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey string

const txKey ctxKey = "pg_tx"

// PgTransactionManager implements usecase.TransactionManager using pgxpool.
type PgTransactionManager struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPgTransactionManager creates a new PgTransactionManager.
func NewPgTransactionManager(pool *pgxpool.Pool, logger *slog.Logger) *PgTransactionManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &PgTransactionManager{pool: pool, logger: logger}
}

// WithTransaction executes fn within a database transaction.
// The transaction is committed if fn returns nil, or rolled back otherwise.
func (tm *PgTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	const op = "PgTransactionManager.WithTransaction"

	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return wrapAndLogDBError(ctx, tm.logger, op+".Begin", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			applogger.WithContext(ctx, tm.logger).ErrorContext(ctx,
				"transaction rollback failed",
				"op", op,
				"error", rollbackErr,
			)
		}
	}()

	txCtx := context.WithValue(ctx, txKey, tx)
	if err := fn(txCtx); err != nil {
		return apperror.Wrap(err, op)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return wrapAndLogDBError(ctx, tm.logger, op+".Commit", err)
	}

	committed = true
	return nil
}

// TxFromContext extracts the pgx.Tx from context if present.
// Returns nil if no transaction is active.
func TxFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey).(pgx.Tx)
	return tx
}

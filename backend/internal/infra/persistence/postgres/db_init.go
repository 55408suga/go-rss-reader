// Package postgres is a package for database operations.
package postgres

import (
	"context"
	"rss_reader/internal/apperror"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDB creates a new database connection pool.
func NewDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	const op = "postgres.NewDB"

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, apperror.NewInvalidArgument(op, "failed to parse config", err)
	}
	config.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, apperror.NewInternal(op, "failed to init connection", err)
	}

	const maxRetries = 5
	const interval = 2 * time.Second

	var lastErr error
	for i := range maxRetries {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

		err := pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return pool, nil
		}
		lastErr = err
		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				pool.Close()
				return nil, apperror.NewInternal(op, "database connection canceled", ctx.Err())
			case <-time.After(interval):
			}
		}

	}

	pool.Close()
	return nil, apperror.NewInternal(op, "failed to connect to database after retries", lastErr)
}

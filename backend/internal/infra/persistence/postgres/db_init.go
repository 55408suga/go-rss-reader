// Package postgres is a package for database operations.
package postgres

import (
	"context"
	"rss_reader/internal/apperror"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDB creates a new database connection pool.
func NewDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	const op = "postgres.NewDB"
	
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil{
		return nil, apperror.NewInvalidArgument(op, "failed to parse config", err)
	}
	config.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, apperror.NewInternal(op, "failed to init connection", err)
	}
	
	return pool, nil
}

package postgres

import (
	"context"
	"errors"
	"log/slog"
	"rss_reader/internal/apperror"
	applogger "rss_reader/internal/applog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func classifyDBError(err error, op string) *apperror.AppError {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(op, "resource not found", err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23505 is the PostgreSQL error code for unique_violation, which indicates a conflict due to duplicate keys
		if pgErr.Code == "23505" {
			return apperror.NewConflict(op, "resource already exists", err)
		}
	}

	return apperror.NewInternal(op, "database error", err)
}

func wrapAndLogDBError(ctx context.Context, logger *slog.Logger, op string, err error) error {
	appErr := classifyDBError(err, op)
	if appErr == nil {
		return nil
	}

	if appErr.Code != apperror.CodeNotFound {
		applogger.WithContext(ctx, logger).ErrorContext(ctx,
			"database operation failed",
			"op", op,
			"code", appErr.Code,
			"error", err,
		)
	}

	return appErr
}

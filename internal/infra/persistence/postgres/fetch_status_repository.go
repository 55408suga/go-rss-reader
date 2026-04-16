package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/infra/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FetchStatusRepository is a PostgreSQL-backed fetch status repository implementation.
type FetchStatusRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewFetchStatusRepository creates a FetchStatusRepository.
func NewFetchStatusRepository(pool *pgxpool.Pool, logger *slog.Logger) *FetchStatusRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &FetchStatusRepository{pool: pool, logger: logger}
}

// querier returns a Queries instance that uses the transaction from context if available.
func (r *FetchStatusRepository) querier(ctx context.Context) *generated.Queries {
	if tx := TxFromContext(ctx); tx != nil {
		return generated.New(tx)
	}
	return generated.New(r.pool)
}

// SaveFetchStatus upserts fetch status for a feed.
func (r *FetchStatusRepository) SaveFetchStatus(ctx context.Context, status *model.FetchStatus) error {
	const op = "FetchStatusRepository.SaveFetchStatus"

	statusCode, err := safeIntToInt32(status.StatusCode, "status_code")
	if err != nil {
		return apperror.NewInternal(op, "status code is out of range", err)
	}

	fetchIntervalHours, err := safeIntToInt32(status.FetchIntervalHours, "fetch_interval_hours")
	if err != nil {
		return apperror.NewInternal(op, "fetch interval is out of range", err)
	}

	failureCount, err := safeIntToInt32(status.FailureCount, "failure_count")
	if err != nil {
		return apperror.NewInternal(op, "failure count is out of range", err)
	}

	params := generated.SaveFeedFetchStatusParams{
		FeedID:             status.FeedID,
		LastFetchedAt:      status.LastFetchedAt,
		NextFetchAt:        status.NextFetchAt,
		StatusCode:         statusCode,
		ErrorMessage:       status.ErrorMessage,
		LastModified:       status.FeedCursor.LastModified,
		Etag:               status.FeedCursor.ETag,
		FetchIntervalHours: fetchIntervalHours,
		FailureCount:       failureCount,
	}

	err = r.querier(ctx).SaveFeedFetchStatus(ctx, params)
	if err != nil {
		return wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return nil
}

func safeIntToInt32(value int, fieldName string) (int32, error) {
	const maxInt32 = int(^uint32(0) >> 1)
	const minInt32 = -maxInt32 - 1

	if value > maxInt32 || value < minInt32 {
		return 0, fmt.Errorf("%s out of int32 range: %d", fieldName, value)
	}

	return int32(value), nil
}

// GetFetchStatusByFeedID retrieves fetch status for a feed.
func (r *FetchStatusRepository) GetFetchStatusByFeedID(
	ctx context.Context,
	feedID uuid.UUID,
) (*model.FetchStatus, error) {
	const op = "FetchStatusRepository.GetFetchStatusByFeedID"

	status, err := r.querier(ctx).GetFeedFetchStatusByFeedID(ctx, feedID)
	if err != nil {
		return nil, wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return toFetchStatusModel(status), nil
}

// GetDueFetchStatuses retrieves statuses that are due for refresh.
func (r *FetchStatusRepository) GetDueFetchStatuses(
	ctx context.Context,
	now time.Time,
	limit int32,
) ([]*model.FetchStatus, error) {
	const op = "FetchStatusRepository.GetDueFetchStatuses"

	rows, err := r.querier(ctx).GetDueFeedFetchStatuses(ctx, generated.GetDueFeedFetchStatusesParams{
		NextFetchAt: now,
		Limit:       limit,
	})
	if err != nil {
		return nil, wrapAndLogDBError(ctx, r.logger, op, err)
	}

	statuses := make([]*model.FetchStatus, 0, len(rows))
	for _, row := range rows {
		statuses = append(statuses, toFetchStatusModel(row))
	}
	return statuses, nil
}

func toFetchStatusModel(status generated.FeedFetchStatus) *model.FetchStatus {
	return &model.FetchStatus{
		FeedID:        status.FeedID,
		LastFetchedAt: status.LastFetchedAt,
		NextFetchAt:   status.NextFetchAt,
		StatusCode:    int(status.StatusCode),
		ErrorMessage:  status.ErrorMessage,
		FeedCursor: model.FeedCursor{
			LastModified: status.LastModified,
			ETag:         status.Etag,
		},
		FetchIntervalHours: int(status.FetchIntervalHours),
		FailureCount:       int(status.FailureCount),
	}
}

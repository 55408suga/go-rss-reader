package postgres

import (
	"context"
	"log/slog"
	"time"

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

	params := generated.SaveFeedFetchStatusParams{
		FeedID:             status.FeedID,
		LastFetchedAt:      status.LastFetchedAt,
		NextFetchAt:        status.NextFetchAt,
		StatusCode:         status.StatusCode,
		ErrorMessage:       status.ErrorMessage,
		LastModified:       status.FeedCursor.LastModified,
		Etag:               status.FeedCursor.ETag,
		FetchIntervalHours: status.FetchIntervalHours,
		FailureCount:       status.FailureCount,
	}

	err := r.querier(ctx).SaveFeedFetchStatus(ctx, params)
	if err != nil {
		return wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return nil
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

// GetDueFeeds retrieves feeds (with current fetch status) that are due for refresh.
func (r *FetchStatusRepository) GetDueFeeds(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]*model.DueFeed, error) {
	const op = "FetchStatusRepository.GetDueFeeds"

	rows, err := r.querier(ctx).GetDueFeedFetchStatuses(ctx, generated.GetDueFeedFetchStatusesParams{
		NextFetchAt: now,
		Column2:     limit,
	})
	if err != nil {
		return nil, wrapAndLogDBError(ctx, r.logger, op, err)
	}

	dueFeeds := make([]*model.DueFeed, 0, len(rows))
	for _, row := range rows {
		dueFeeds = append(dueFeeds, toDueFeedModel(row))
	}
	return dueFeeds, nil
}

func toFetchStatusModel(status generated.FeedFetchStatus) *model.FetchStatus {
	return &model.FetchStatus{
		FeedID:        status.FeedID,
		LastFetchedAt: status.LastFetchedAt,
		NextFetchAt:   status.NextFetchAt,
		StatusCode:    status.StatusCode,
		ErrorMessage:  status.ErrorMessage,
		FeedCursor: model.FeedCursor{
			LastModified: status.LastModified,
			ETag:         status.Etag,
		},
		FetchIntervalHours: status.FetchIntervalHours,
		FailureCount:       status.FailureCount,
	}
}

func toDueFeedModel(row generated.GetDueFeedFetchStatusesRow) *model.DueFeed {
	return &model.DueFeed{
		FeedURL: row.FeedUrl,
		Status: &model.FetchStatus{
			FeedID:        row.FeedID,
			LastFetchedAt: row.LastFetchedAt,
			NextFetchAt:   row.NextFetchAt,
			StatusCode:    row.StatusCode,
			ErrorMessage:  row.ErrorMessage,
			FeedCursor: model.FeedCursor{
				LastModified: row.LastModified,
				ETag:         row.Etag,
			},
			FetchIntervalHours: row.FetchIntervalHours,
			FailureCount:       row.FailureCount,
		},
	}
}

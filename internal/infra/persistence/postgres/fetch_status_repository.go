package postgres

import (
	"context"
	"time"

	"rss_reader/internal/domain/model"
	"rss_reader/internal/infra/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FetchStatusRepository struct {
	pool *pgxpool.Pool
}

func NewFetchStatusRepository(pool *pgxpool.Pool) *FetchStatusRepository {
	return &FetchStatusRepository{pool: pool}
}

// querier returns a Queries instance that uses the transaction from context if available.
func (r *FetchStatusRepository) querier(ctx context.Context) *generated.Queries {
	if tx := TxFromContext(ctx); tx != nil {
		return generated.New(tx)
	}
	return generated.New(r.pool)
}

func (r *FetchStatusRepository) SaveFetchStatus(ctx context.Context, status *model.FetchStatus) error {
	params := generated.SaveFeedFetchStatusParams{
		FeedID:             status.FeedID,
		LastFetchedAt:      status.LastFetchedAt,
		NextFetchAt:        status.NextFetchAt,
		StatusCode:         int32(status.StatusCode),
		ErrorMessage:       status.ErrorMessage,
		LastModified:       status.FeedCursor.LastModified,
		Etag:               status.FeedCursor.ETag,
		FetchIntervalHours: int32(status.FetchIntervalHours),
		FailureCount:       int32(status.FailureCount),
	}

	return r.querier(ctx).SaveFeedFetchStatus(ctx, params)
}

func (r *FetchStatusRepository) GetFetchStatusByFeedID(ctx context.Context, feedID uuid.UUID) (*model.FetchStatus, error) {
	status, err := r.querier(ctx).GetFeedFetchStatusByFeedID(ctx, feedID)
	if err != nil {
		return nil, err
	}

	return toFetchStatusModel(status), nil
}

func (r *FetchStatusRepository) GetDueFetchStatuses(ctx context.Context, now time.Time, limit int32) ([]*model.FetchStatus, error) {
	rows, err := r.querier(ctx).GetDueFeedFetchStatuses(ctx, generated.GetDueFeedFetchStatusesParams{
		NextFetchAt: now,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
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

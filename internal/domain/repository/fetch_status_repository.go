package repository

import (
	"context"
	"rss_reader/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

type FetchStatusRepository interface {
	SaveFetchStatus(ctx context.Context, status *model.FetchStatus) error
	GetFetchStatusByFeedID(ctx context.Context, feedID uuid.UUID) (*model.FetchStatus, error)
	GetDueFetchStatuses(ctx context.Context, now time.Time, limit int32) ([]*model.FetchStatus, error)
}

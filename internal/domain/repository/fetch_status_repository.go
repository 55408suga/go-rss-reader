package repository

import (
	"context"
	"rss_reader/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// FetchStatusRepository defines persistence operations for fetch status records.
type FetchStatusRepository interface {
	SaveFetchStatus(ctx context.Context, status *model.FetchStatus) error
	GetFetchStatusByFeedID(ctx context.Context, feedID uuid.UUID) (*model.FetchStatus, error)
	GetDueFetchStatuses(ctx context.Context, now time.Time, limit int) ([]*model.FetchStatus, error)
}

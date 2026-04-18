package usecase

import "context"

type FeedJobUsecase interface {
	RefreshDueFeeds(ctx context.Context) error
}

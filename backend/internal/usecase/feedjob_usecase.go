package usecase

import "context"

// FeedJobUsecase defines the interface for scheduled feed refresh jobs.
type FeedJobUsecase interface {
	RefreshDueFeeds(ctx context.Context) error
}

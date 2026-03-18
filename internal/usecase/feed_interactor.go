package usecase

import (
	"context"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/domain/repository"

	"github.com/google/uuid"
)

// FeedInteractor implements FeedUsecase interface.
type FeedInteractor struct {
	feedRepo repository.FeedRepository
	fetcher  FeedFetcher
}

// NewFeedInteractor represents constructor of FeedInteractor.
func NewFeedInteractor(
	feedRepo repository.FeedRepository,
	fetcher FeedFetcher,
) *FeedInteractor {
	return &FeedInteractor{
		feedRepo: feedRepo,
		fetcher:  fetcher,
	}
}

// RegisterFeed fetches a feed from the given URL and saves it.
func (i *FeedInteractor) RegisterFeed(ctx context.Context, feedURL string) (*model.Feed, error) {
	feedData, err := i.fetcher.FetchFeed(ctx, feedURL)
	if err != nil {
		return nil, err
	}
	if err := i.feedRepo.SaveFeed(ctx, feedData); err != nil {
		return nil, err
	}
	return feedData, nil
}

// GetFeedByID returns a feed by its ID.
func (i *FeedInteractor) GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error) {
	return i.feedRepo.GetFeed(ctx, feedID)
}

// GetAllFeeds returns all feeds.
func (i *FeedInteractor) GetAllFeeds(ctx context.Context) ([]*model.Feed, error) {
	return i.feedRepo.GetAllFeeds(ctx)
}

// RefreshFeed fetches latest feed for the given feed and saves it.
func (i *FeedInteractor) RefreshFeed(ctx context.Context, feedID string) error {
	feedData, err := i.fetcher.FetchFeed(ctx, feedID)
	if err != nil {
		return err
	}
	if err := i.feedRepo.UpdateFeed(ctx, feedData); err != nil {
		return err
	}
	return nil
}

// DeleteFeed deletes a feed by its ID.
func (i *FeedInteractor) DeleteFeed(ctx context.Context, feedID uuid.UUID) error {
	return i.feedRepo.DeleteFeed(ctx, feedID)
}

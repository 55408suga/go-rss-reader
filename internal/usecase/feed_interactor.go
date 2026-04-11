package usecase

import (
	"context"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/domain/repository"

	"github.com/google/uuid"
)

// FeedInteractor implements FeedUsecase interface.
type FeedInteractor struct {
	feedRepo    repository.FeedRepository
	articleRepo repository.ArticleRepository
	fetcher     RSSFetcher
	txManager   TransactionManager
}

// NewFeedInteractor represents constructor of FeedInteractor.
func NewFeedInteractor(
	feedRepo repository.FeedRepository,
	articleRepo repository.ArticleRepository,
	fetcher RSSFetcher,
	txManager TransactionManager,
) *FeedInteractor {
	return &FeedInteractor{
		feedRepo:    feedRepo,
		articleRepo: articleRepo,
		fetcher:     fetcher,
		txManager:   txManager,
	}
}

// RegisterFeed fetches a feed with articles from the given URL and saves them atomically.
func (i *FeedInteractor) RegisterFeed(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, error) {
	feed, articles, err := i.fetcher.FetchFeedWithArticles(ctx, feedURL)
	if err != nil {
		return nil, nil, err
	}

	err = i.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := i.feedRepo.SaveFeed(txCtx, feed); err != nil {
			return err
		}
		for _, article := range articles {
			article.FeedID = feed.ID
			if err := i.articleRepo.SaveArticle(txCtx, article); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return feed, articles, nil
}

// GetFeedByID returns a feed by its ID.
func (i *FeedInteractor) GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error) {
	return i.feedRepo.GetFeedByID(ctx, feedID)
}

// GetAllFeeds returns all feeds.
func (i *FeedInteractor) GetAllFeeds(ctx context.Context) ([]*model.Feed, error) {
	return i.feedRepo.GetAllFeeds(ctx)
}

// RefreshFeed fetches latest feed metadata and articles for the given feed and saves them atomically.
func (i *FeedInteractor) RefreshFeed(ctx context.Context, feedID uuid.UUID) error {
	currentFeed, err := i.feedRepo.GetFeedByID(ctx, feedID)
	if err != nil {
		return err
	}

	feed, articles, err := i.fetcher.FetchFeedWithArticles(ctx, currentFeed.FeedURL)
	if err != nil {
		return err
	}
	feed.ID = currentFeed.ID

	return i.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := i.feedRepo.UpdateFeed(txCtx, feed); err != nil {
			return err
		}
		for _, article := range articles {
			article.FeedID = feed.ID
			if err := i.articleRepo.SaveArticle(txCtx, article); err != nil {
				return err
			}
		}
		return nil
	})
}

// RefreshAllFeeds refreshes all registered feeds and their articles.
func (i *FeedInteractor) RefreshAllFeeds(ctx context.Context) error {
	feeds, err := i.feedRepo.GetAllFeeds(ctx)
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		if err := i.RefreshFeed(ctx, feed.ID); err != nil {
			return err
		}
	}
	return nil
}

// DeleteFeed deletes a feed by its ID.
func (i *FeedInteractor) DeleteFeed(ctx context.Context, feedID uuid.UUID) error {
	return i.feedRepo.DeleteFeed(ctx, feedID)
}

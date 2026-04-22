package usecase

import (
	"context"
	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/domain/repository"

	"github.com/google/uuid"
)

// FeedInteractor implements FeedUsecase interface.
type FeedInteractor struct {
	feedRepo       repository.FeedRepository
	articleRepo    repository.ArticleRepository
	feedStatusRepo repository.FetchStatusRepository
	fetcher        RSSFetcher
	txManager      TransactionManager
}

// NewFeedInteractor represents constructor of FeedInteractor.
func NewFeedInteractor(
	feedRepo repository.FeedRepository,
	articleRepo repository.ArticleRepository,
	feedStatusRepo repository.FetchStatusRepository,
	fetcher RSSFetcher,
	txManager TransactionManager,
) *FeedInteractor {
	return &FeedInteractor{
		feedRepo:       feedRepo,
		articleRepo:    articleRepo,
		feedStatusRepo: feedStatusRepo,
		fetcher:        fetcher,
		txManager:      txManager,
	}
}

// RegisterFeed fetches a feed and its articles, then saves feed/articles/fetch-status atomically.
func (i *FeedInteractor) RegisterFeed(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, error) {
	const op = "FeedInteractor.RegisterFeed"

	feed, articles, feedCursor, err := i.fetcher.FetchNewFeed(ctx, feedURL)
	if err != nil {
		return nil, nil, apperror.Wrap(err, op)
	}

	cursor := model.FeedCursor{}
	if feedCursor != nil {
		cursor = *feedCursor
	}
	fetchStatus := model.NewFetchStatus(feed.ID, cursor)

	err = i.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := i.feedRepo.SaveFeed(txCtx, feed); err != nil {
			return apperror.Wrap(err, op+".SaveFeed")
		}

		for _, article := range articles {
			if err := i.articleRepo.SaveArticle(txCtx, article); err != nil {
				return apperror.Wrap(err, op+".SaveArticle")
			}
		}

		if err := i.feedStatusRepo.SaveFetchStatus(txCtx, fetchStatus); err != nil {
			return apperror.Wrap(err, op+".SaveFetchStatus")
		}

		return nil
	})
	if err != nil {
		return nil, nil, apperror.Wrap(err, op)
	}

	return feed, articles, nil
}

// GetFeedByID returns a feed by its ID.
func (i *FeedInteractor) GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error) {
	const op = "FeedInteractor.GetFeedByID"

	feed, err := i.feedRepo.GetFeedByID(ctx, feedID)
	if err != nil {
		return nil, apperror.Wrap(err, op)
	}

	return feed, nil
}

// ListFeeds returns feeds up to limit starting after cursor (nil = first page).
func (i *FeedInteractor) ListFeeds(ctx context.Context, cursor *model.PageCursor, limit int) ([]*model.Feed, error) {
	const op = "FeedInteractor.ListFeeds"

	feeds, err := i.feedRepo.ListFeeds(ctx, cursor, limit)
	if err != nil {
		return nil, apperror.Wrap(err, op)
	}

	return feeds, nil
}

// RefreshFeed fetches latest feed metadata and articles for the given feed and saves them atomically.
func (i *FeedInteractor) RefreshFeed(ctx context.Context, feedID uuid.UUID) error {
	const op = "FeedInteractor.RefreshFeed"

	currentFeed, err := i.feedRepo.GetFeedByID(ctx, feedID)
	if err != nil {
		return apperror.Wrap(err, op+".GetFeedByID")
	}

	feed, articles, _, err := i.fetcher.FetchNewFeed(ctx, currentFeed.FeedURL)
	if err != nil {
		return apperror.Wrap(err, op+".FetchFeedWithArticles")
	}
	feed.ID = currentFeed.ID

	err = i.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := i.feedRepo.UpdateFeed(txCtx, feed); err != nil {
			return apperror.Wrap(err, op+".UpdateFeed")
		}
		for _, article := range articles {
			article.FeedID = feed.ID
			if err := i.articleRepo.SaveArticle(txCtx, article); err != nil {
				return apperror.Wrap(err, op+".SaveArticle")
			}
		}
		return nil
	})
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return nil
}

// DeleteFeed deletes a feed by its ID.
func (i *FeedInteractor) DeleteFeed(ctx context.Context, feedID uuid.UUID) error {
	const op = "FeedInteractor.DeleteFeed"

	err := i.feedRepo.DeleteFeed(ctx, feedID)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return nil
}

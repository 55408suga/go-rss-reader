package usecase

import (
	"context"
	"log/slog"
	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/domain/repository"
	applogger "rss_reader/internal/infra/logger"

	"github.com/google/uuid"
)

// FeedInteractor implements FeedUsecase interface.
type FeedInteractor struct {
	feedRepo       repository.FeedRepository
	articleRepo    repository.ArticleRepository
	feedStatusRepo repository.FetchStatusRepository
	fetcher        RSSFetcher
	txManager      TransactionManager
	logger         *slog.Logger
}

// NewFeedInteractor represents constructor of FeedInteractor.
func NewFeedInteractor(
	feedRepo repository.FeedRepository,
	articleRepo repository.ArticleRepository,
	feedStatusRepo repository.FetchStatusRepository,
	fetcher RSSFetcher,
	txManager TransactionManager,
	logger *slog.Logger,
) *FeedInteractor {
	if logger == nil {
		logger = slog.Default()
	}

	return &FeedInteractor{
		feedRepo:       feedRepo,
		articleRepo:    articleRepo,
		feedStatusRepo: feedStatusRepo,
		fetcher:        fetcher,
		txManager:      txManager,
		logger:         logger,
	}
}

// RegisterFeed fetches a feed and its articles, then saves feed/articles/fetch-status atomically.
func (i *FeedInteractor) RegisterFeed(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, error) {
	const op = "FeedInteractor.RegisterFeed"

	feed, articles, feedCursor, err := i.fetcher.FetchFeedWithArticles(ctx, feedURL)
	if err != nil {
		applogger.WithContext(ctx, i.logger).ErrorContext(ctx,
			"register feed failed while fetching external feed",
			"feed_url", feedURL,
			"error", err,
		)
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
			article.FeedID = feed.ID
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
		applogger.WithContext(ctx, i.logger).ErrorContext(ctx,
			"register feed failed while saving data",
			"feed_url", feedURL,
			"error", err,
		)
		return nil, nil, apperror.Wrap(err, op)
	}

	return feed, articles, nil
}

// GetFeedByID returns a feed by its ID.
func (i *FeedInteractor) GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error) {
	const op = "FeedInteractor.GetFeedByID"

	feed, err := i.feedRepo.GetFeedByID(ctx, feedID)
	if err != nil {
		applogger.WithContext(ctx, i.logger).WarnContext(ctx,
			"get feed by id failed",
			"feed_id", feedID,
			"error", err,
		)
		return nil, apperror.Wrap(err, op)
	}

	return feed, nil
}

// GetAllFeeds returns all feeds.
func (i *FeedInteractor) GetAllFeeds(ctx context.Context) ([]*model.Feed, error) {
	const op = "FeedInteractor.GetAllFeeds"

	feeds, err := i.feedRepo.GetAllFeeds(ctx)
	if err != nil {
		applogger.WithContext(ctx, i.logger).ErrorContext(ctx,
			"get all feeds failed",
			"error", err,
		)
		return nil, apperror.Wrap(err, op)
	}

	return feeds, nil
}

// RefreshFeed fetches latest feed metadata and articles for the given feed and saves them atomically.
func (i *FeedInteractor) RefreshFeed(ctx context.Context, feedID uuid.UUID) error {
	const op = "FeedInteractor.RefreshFeed"

	currentFeed, err := i.feedRepo.GetFeedByID(ctx, feedID)
	if err != nil {
		applogger.WithContext(ctx, i.logger).WarnContext(ctx,
			"refresh feed failed while loading current feed",
			"feed_id", feedID,
			"error", err,
		)
		return apperror.Wrap(err, op+".GetFeedByID")
	}

	feed, articles, _, err := i.fetcher.FetchFeedWithArticles(ctx, currentFeed.FeedURL)
	if err != nil {
		applogger.WithContext(ctx, i.logger).ErrorContext(ctx,
			"refresh feed failed while fetching external feed",
			"feed_id", feedID,
			"feed_url", currentFeed.FeedURL,
			"error", err,
		)
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
		applogger.WithContext(ctx, i.logger).ErrorContext(ctx,
			"refresh feed failed while saving data",
			"feed_id", feedID,
			"error", err,
		)
		return apperror.Wrap(err, op)
	}

	return nil
}

// RefreshAllFeeds refreshes all registered feeds and their articles.
func (i *FeedInteractor) RefreshAllFeeds(ctx context.Context) error {
	const op = "FeedInteractor.RefreshAllFeeds"

	feeds, err := i.feedRepo.GetAllFeeds(ctx)
	if err != nil {
		applogger.WithContext(ctx, i.logger).ErrorContext(ctx,
			"refresh all feeds failed while loading feeds",
			"error", err,
		)
		return apperror.Wrap(err, op+".GetAllFeeds")
	}
	for _, feed := range feeds {
		if err := i.RefreshFeed(ctx, feed.ID); err != nil {
			applogger.WithContext(ctx, i.logger).ErrorContext(ctx,
				"refresh all feeds failed while refreshing a feed",
				"feed_id", feed.ID,
				"error", err,
			)
			return apperror.Wrap(err, op)
		}
	}
	return nil
}

// DeleteFeed deletes a feed by its ID.
func (i *FeedInteractor) DeleteFeed(ctx context.Context, feedID uuid.UUID) error {
	const op = "FeedInteractor.DeleteFeed"

	err := i.feedRepo.DeleteFeed(ctx, feedID)
	if err != nil {
		applogger.WithContext(ctx, i.logger).WarnContext(ctx,
			"delete feed failed",
			"feed_id", feedID,
			"error", err,
		)
		return apperror.Wrap(err, op)
	}

	return nil
}

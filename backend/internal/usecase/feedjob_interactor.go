package usecase

import (
	"context"
	"log/slog"
	"rss_reader/internal/apperror"
	applogger "rss_reader/internal/applog"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/domain/repository"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	// refreshDueConcurrency caps in-flight feed refreshes per tick.
	refreshDueConcurrency = 10

	// refreshDueLimit caps how many feeds one RefreshDueFeeds call processes.
	refreshDueLimit = 100

	// refreshNextIntervalHours is the default offset used for NextFetchAt.
	// TODO: replace with a dynamic interval algorithm later.
	refreshNextIntervalHours = 12
	refreshNextInterval      = refreshNextIntervalHours * time.Hour
)

// FeedJobInteractor orchestrates scheduled feed refreshes.
// It intentionally does not depend on FeedRepository: the batch refresh only adds new
// articles and updates fetch status. Feed metadata (title, description, language) changes
// rarely and is re-synced via the user-initiated FeedInteractor.RefreshFeed path instead.
type FeedJobInteractor struct {
	fetcher         RSSFetcher
	articleRepo     repository.ArticleRepository
	fetchStatusRepo repository.FetchStatusRepository
	txManager       TransactionManager
	logger          *slog.Logger
}

// NewFeedJobInteractor wires dependencies for the feed refresh job.
func NewFeedJobInteractor(
	fetcher RSSFetcher,
	articleRepo repository.ArticleRepository,
	fetchStatusRepo repository.FetchStatusRepository,
	txManager TransactionManager,
	logger *slog.Logger,
) *FeedJobInteractor {
	if logger == nil {
		logger = slog.Default()
	}
	return &FeedJobInteractor{
		fetcher:         fetcher,
		articleRepo:     articleRepo,
		fetchStatusRepo: fetchStatusRepo,
		txManager:       txManager,
		logger:          logger,
	}
}

// RefreshDueFeeds fetches due feeds in parallel and persists new articles and updated fetch status.
// Individual feed failures are logged and swallowed so one bad feed does not stop the batch.
func (i *FeedJobInteractor) RefreshDueFeeds(ctx context.Context) error {
	const op = "FeedJobInteractor.RefreshDueFeeds"

	logger := applogger.WithContext(ctx, i.logger)

	dueFeeds, err := i.fetchStatusRepo.GetDueFeeds(ctx, time.Now().UTC(), refreshDueLimit)
	if err != nil {
		return apperror.Wrap(err, op+".GetDueFeeds")
	}
	if len(dueFeeds) == 0 {
		logger.InfoContext(ctx, "no due feeds to refresh")
		return nil
	}
	logger.InfoContext(ctx, "refreshing due feeds", "count", len(dueFeeds))

	var g errgroup.Group
	g.SetLimit(refreshDueConcurrency)

	for _, feed := range dueFeeds {
		g.Go(func() error {
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorContext(ctx, "refreshOne panicked",
						"feed_url", feed.FeedURL,
						"feed_id", feed.Status.FeedID,
						"error", r,
					)
				}
			}()
			if err := i.refreshOne(ctx, feed); err != nil {
				logger.WarnContext(ctx, "failed to refresh feed",
					"feed_url", feed.FeedURL,
					"feed_id", feed.Status.FeedID,
					"error", err,
				)
			}
			return nil
		})
	}
	_ = g.Wait()

	return ctx.Err()
}

// refreshOne refreshes a single due feed: fetch → save articles → update fetch status.
// Even on fetch failure the fetch status is persisted so NextFetchAt advances and the
// same feed is not immediately re-picked on the next tick.
func (i *FeedJobInteractor) refreshOne(ctx context.Context, due *model.DueFeed) error {
	const op = "FeedJobInteractor.refreshOne"

	now := time.Now().UTC()

	feed, articles, newCursor, err := i.fetcher.FetchFeedWithCursor(ctx, due.FeedURL, &due.Status.FeedCursor)
	if err != nil {
		msg := err.Error()
		status := i.buildFetchStatus(due, now, 0, &msg, due.Status.FeedCursor, due.Status.FailureCount+1)
		if saveErr := i.fetchStatusRepo.SaveFetchStatus(ctx, status); saveErr != nil {
			return apperror.Wrap(saveErr, op+".SaveFetchStatus")
		}
		return apperror.Wrap(err, op+".FetchFeedWithCursor")
	}

	// 304 Not Modified: no articles, cursor unchanged — single write, no tx needed.
	if feed == nil {
		status := i.buildFetchStatus(due, now, 304, nil, *newCursor, 0)
		if err := i.fetchStatusRepo.SaveFetchStatus(ctx, status); err != nil {
			return apperror.Wrap(err, op+".SaveFetchStatus")
		}
		return nil
	}

	// Success: persist new articles and fetch status atomically.
	err = i.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, article := range articles {
			if err := i.articleRepo.SaveArticle(txCtx, article); err != nil {
				return apperror.Wrap(err, op+".SaveArticle")
			}
		}
		status := i.buildFetchStatus(due, now, 200, nil, *newCursor, 0)
		if err := i.fetchStatusRepo.SaveFetchStatus(txCtx, status); err != nil {
			return apperror.Wrap(err, op+".SaveFetchStatus")
		}
		return nil
	})
	if err != nil {
		return apperror.Wrap(err, op)
	}
	return nil
}

// buildFetchStatus constructs a FetchStatus with shared defaults (feed_id, now timestamps,
// interval). Callers supply the outcome-specific fields: statusCode, errMsg, cursor, failureCount.
func (i *FeedJobInteractor) buildFetchStatus(
	due *model.DueFeed,
	now time.Time,
	statusCode int,
	errMsg *string,
	cursor model.FeedCursor,
	failureCount int,
) *model.FetchStatus {
	return model.NewFetchStatusWith(
		due.Status.FeedID,
		now,
		now.Add(refreshNextInterval),
		statusCode,
		errMsg,
		cursor,
		refreshNextIntervalHours,
		failureCount,
	)
}

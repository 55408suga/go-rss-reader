package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/domain/repository"
)

// FeedInteractor implements FeedUsecase interface.
type FeedInteractor struct {
	feedRepo       repository.FeedRepository
	articleRepo    repository.ArticleRepository
	feedStatusRepo repository.FetchStatusRepository
	fetcher        RSSFetcher
	discoverer     FeedDiscoverer
	txManager      TransactionManager
}

// NewFeedInteractor represents constructor of FeedInteractor.
func NewFeedInteractor(
	feedRepo repository.FeedRepository,
	articleRepo repository.ArticleRepository,
	feedStatusRepo repository.FetchStatusRepository,
	fetcher RSSFetcher,
	discoverer FeedDiscoverer,
	txManager TransactionManager,
) *FeedInteractor {
	return &FeedInteractor{
		feedRepo:       feedRepo,
		articleRepo:    articleRepo,
		feedStatusRepo: feedStatusRepo,
		fetcher:        fetcher,
		discoverer:     discoverer,
		txManager:      txManager,
	}
}

// RegisterFeed fetches a feed and its articles, then saves feed/articles/fetch-status atomically.
func (i *FeedInteractor) RegisterFeed(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, error) {
	const op = "FeedInteractor.RegisterFeed"

	exists, err := i.feedRepo.CheckFeedExistsByURL(ctx, feedURL)
	if err != nil {
		return nil, nil, apperror.Wrap(err, op)
	}
	if exists {
		return nil, nil, apperror.NewConflict(op, "this feed is already registered", nil)
	}

	feed, articles, feedCursor, err := i.fetcher.FetchNewFeed(ctx, feedURL)
	if err != nil {
		return nil, nil, apperror.Wrap(err, op)
	}

	fetchStatus := model.NewFetchStatus(feed.ID, *feedCursor)

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

// DiscoverAndRegisterFeed resolves a website URL to its advertised feed and
// subscribes to it: a DB fast path detects already-subscribed websites
// without touching the network, then the HTML is scanned for autodiscovery
// links and the first (highest-priority) candidate is registered through the
// existing RegisterFeed flow. All discovered candidates are returned so the
// caller can offer alternatives.
func (i *FeedInteractor) DiscoverAndRegisterFeed(
	ctx context.Context,
	websiteURL string,
) (*model.Feed, []*model.Article, []model.FeedCandidate, error) {
	const op = "FeedInteractor.DiscoverAndRegisterFeed"

	_, err := i.feedRepo.GetFeedByWebsiteURL(ctx, websiteURLVariants(websiteURL))
	if err == nil {
		return nil, nil, nil, apperror.NewConflict(op, "this website is already subscribed", nil)
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeNotFound {
		return nil, nil, nil, apperror.Wrap(err, op)
	}
	// not_found here means "not subscribed yet": fall through to discovery.

	candidates, err := i.discoverer.DiscoverFeedURLs(ctx, websiteURL)
	if err != nil {
		return nil, nil, nil, apperror.Wrap(err, op)
	}
	if len(candidates) == 0 {
		// Defensive: the gateway classifies this itself, but a discoverer
		// returning (nil, nil) must not panic on candidates[0] below.
		return nil, nil, nil, apperror.NewNotFound(op, "no rss/atom feed found at this website", nil)
	}

	feed, articles, err := i.RegisterFeed(ctx, candidates[0].FeedURL)
	if err != nil {
		return nil, nil, nil, apperror.Wrap(err, op)
	}

	return feed, articles, candidates, nil
}

// websiteURLVariants returns the input URL plus its trailing-slash twin, so
// the single-query DB fast path tolerates the most common mismatch between
// what users type and the channel link the feed self-reported. A DB miss is
// not proof the site is unsubscribed — RegisterFeed's feed_url check stays
// the source of truth.
func websiteURLVariants(websiteURL string) []string {
	if bare, ok := strings.CutSuffix(websiteURL, "/"); ok {
		return []string{websiteURL, bare}
	}
	return []string{websiteURL, websiteURL + "/"}
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

// ListFeeds returns one keyset page of feeds starting after cursor (nil = first
// page). It over-fetches by one row (limit+1) so paginate can report has_more
// and the next cursor without a second query.
func (i *FeedInteractor) ListFeeds(
	ctx context.Context,
	cursor *model.PageCursor,
	limit int,
) (*model.Page[*model.Feed], error) {
	const op = "FeedInteractor.ListFeeds"

	if err := validateLimit(op, limit); err != nil {
		return nil, err
	}

	feeds, err := i.feedRepo.ListFeeds(ctx, cursor, limit+1)
	if err != nil {
		return nil, apperror.Wrap(err, op)
	}

	// Feeds are ordered by (registered_at DESC, id DESC); the cursor mirrors
	// that keyset so the next page resumes strictly after the last feed.
	return paginate(feeds, limit, func(f *model.Feed) model.PageCursor {
		return model.PageCursor{At: f.RegisteredAt, ID: f.ID}
	}), nil
}

// RefreshFeed fetches latest feed metadata and articles for the given feed and saves them atomically.
func (i *FeedInteractor) RefreshFeed(ctx context.Context, feedID uuid.UUID) error {
	const op = "FeedInteractor.RefreshFeed"

	currentFeed, err := i.feedRepo.GetFeedByID(ctx, feedID)
	if err != nil {
		return apperror.Wrap(err, op+".GetFeedByID")
	}

	feed, articles, feedCursor, err := i.fetcher.FetchNewFeed(ctx, currentFeed.FeedURL)
	if err != nil {
		return apperror.Wrap(err, op+".FetchNewFeed")
	}
	// overwrite feed ID
	feed.ID = currentFeed.ID
	fetchStatus := model.NewFetchStatus(feed.ID, *feedCursor)

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
		if err := i.feedStatusRepo.SaveFetchStatus(txCtx, fetchStatus); err != nil {
			return apperror.Wrap(err, op+".SaveFetchStatus")
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

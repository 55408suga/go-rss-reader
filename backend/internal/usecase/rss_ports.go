// Package usecase defines the application's input and output ports.
//
// Input ports are interfaces consumed by the delivery layer (HTTP handlers,
// schedulers) and implemented by Interactor types in this package.
//
// Output ports are interfaces consumed by Interactors and implemented by
// the infrastructure layer (gateways, repositories).
package usecase

import (
	"context"
	"rss_reader/internal/domain/model"

	"github.com/google/uuid"
)

// =====================================================================
// Input ports — implemented by Interactors, consumed by handlers / cmd
// =====================================================================

// FeedUsecase defines feed-related use cases driven by HTTP handlers.
type FeedUsecase interface {
	RegisterFeed(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, error)
	GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error)
	ListFeeds(ctx context.Context, cursor *model.PageCursor, limit int) ([]*model.Feed, error)
	RefreshFeed(ctx context.Context, feedID uuid.UUID) error
	DeleteFeed(ctx context.Context, feedID uuid.UUID) error
}

// ArticleUsecase defines article-related use cases driven by HTTP handlers.
type ArticleUsecase interface {
	ListArticlesByFeedID(
		ctx context.Context,
		feedID uuid.UUID,
		cursor *model.PageCursor,
		limit int,
	) ([]*model.Article, error)
	ListArticles(ctx context.Context, cursor *model.PageCursor, limit int) ([]*model.Article, error)
}

// FeedJobUsecase defines scheduled feed refresh jobs driven by the scheduler.
type FeedJobUsecase interface {
	RefreshDueFeeds(ctx context.Context) error
}

// =====================================================================
// Output ports — implemented by infra/, consumed by Interactors
// =====================================================================

// RSSFetcher fetches feed metadata and articles from external RSS sources.
// Implemented by gateway.RSSGateway.
type RSSFetcher interface {
	FetchNewFeed(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, *model.FeedCursor, error)
	FetchFeedWithCursor(
		ctx context.Context,
		feedURL string,
		feedCursor *model.FeedCursor,
	) (*model.Feed, []*model.Article, *model.FeedCursor, error)
}

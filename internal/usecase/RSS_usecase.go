package usecase

import (
	"context"
	"rss_reader/internal/domain/model"

	"github.com/google/uuid"
)

// RSSFetcher defines the interface for fetching feed and articles from external RSS sources.
type RSSFetcher interface {
	FetchNewFeed(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, *model.FeedCursor, error)
	FetchFeedWithCursor(
		ctx context.Context,
		feedURL string,
		feedCursor *model.FeedCursor,
	) (*model.Feed, []*model.Article, *model.FeedCursor, error)
}

// FeedUsecase defines the interface for feed-related use cases.
type FeedUsecase interface {
	RegisterFeed(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, error)
	GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error)
	ListFeeds(ctx context.Context, cursor *model.PageCursor, limit int) ([]*model.Feed, error)
	RefreshFeed(ctx context.Context, feedID uuid.UUID) error
	RefreshAllFeeds(ctx context.Context) error
	DeleteFeed(ctx context.Context, feedID uuid.UUID) error
}

// ArticleUsecase defines the interface for article-related use cases.
type ArticleUsecase interface {
	ListArticlesByFeedID(ctx context.Context, feedID uuid.UUID, cursor *model.PageCursor, limit int) ([]*model.Article, error)
	ListArticles(ctx context.Context, cursor *model.PageCursor, limit int) ([]*model.Article, error)
}

package usecase

import (
	"context"
	"rss_reader/internal/domain/model"

	"github.com/google/uuid"
)

// FeedFetcher defines the interface for fetching feed metadata from external sources.
type FeedFetcher interface {
	FetchFeed(ctx context.Context, feedURL string) (*model.Feed, error)
}

// ArticleFetcher defines the interface for fetching articles from external sources.
type ArticleFetcher interface {
	FetchArticles(ctx context.Context, feedID uuid.UUID, feedURL string) ([]*model.Article, error)
}

// FeedUsecase defines the interface for feed-related use cases.
type FeedUsecase interface {
	RegisterFeed(ctx context.Context, feedURL string) (*model.Feed, error)
	GetFeedByID(ctx context.Context, feedID string) (*model.Feed, error)
	GetAllFeeds(ctx context.Context) ([]*model.Feed, error)
	RefreshFeed(ctx context.Context, feedID string) error
	DeleteFeed(ctx context.Context, feedID string) error
}

// ArticleUsecase defines the interface for article-related use cases.
type ArticleUsecase interface {
	GetArticlesByFeedID(ctx context.Context, feedID string) ([]*model.Article, error)
	RefreshArticles(ctx context.Context) error
}

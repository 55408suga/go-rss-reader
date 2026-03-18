// Package repository provides abstracts of database operations.
package repository

import (
	"context"
	"rss_reader/internal/domain/model"

	"github.com/google/uuid"
)

// FeedRepository defines the interface for feed repository.
type FeedRepository interface {
	SaveFeed(ctx context.Context, feed *model.Feed) error
	GetFeed(ctx context.Context, feedID uuid.UUID) (*model.Feed, error)
	GetAllFeeds(ctx context.Context) ([]*model.Feed, error)
	GetArticles(ctx context.Context, feedID uuid.UUID) ([]*model.Article, error)
	UpdateFeed(ctx context.Context, feed *model.Feed) error
	DeleteFeed(ctx context.Context, feedID uuid.UUID) error
}

// ArticleRepository defines the interface for article repository.
type ArticleRepository interface {
	RegisterArticle(ctx context.Context, article *model.Article) error
	GetArticle(ctx context.Context, articleID uuid.UUID) (*model.Article, error)
	GetAllArticles(ctx context.Context) ([]*model.Article, error)
	UpdateArticle(ctx context.Context, article *model.Article) error
	DeleteArticle(ctx context.Context, articleID uuid.UUID) error
}

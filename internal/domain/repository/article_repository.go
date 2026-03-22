package repository

import (
	"context"
	"rss_reader/internal/domain/model"

	"github.com/google/uuid"
)

// ArticleRepository defines the interface for article repository.
type ArticleRepository interface {
	RegisterArticle(ctx context.Context, article *model.Article) error
	GetArticle(ctx context.Context, articleID uuid.UUID) (*model.Article, error)
	GetArticlesByFeedID(ctx context.Context, feedID uuid.UUID) ([]*model.Article, error)
	GetAllArticles(ctx context.Context) ([]*model.Article, error)
	UpdateArticle(ctx context.Context, article *model.Article) error
	DeleteArticle(ctx context.Context, articleID uuid.UUID) error
}

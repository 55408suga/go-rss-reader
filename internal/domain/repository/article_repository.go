package repository

import (
	"context"
	"rss_reader/internal/domain/model"

	"github.com/google/uuid"
)

// ArticleRepository defines the interface for article repository.
type ArticleRepository interface {
	SaveArticle(ctx context.Context, article *model.Article) error
	GetArticleByID(ctx context.Context, articleID uuid.UUID) (*model.Article, error)
	ListArticlesByFeedID(ctx context.Context, feedID uuid.UUID, cursor *model.PageCursor, limit int) ([]*model.Article, error)
	ListArticles(ctx context.Context, cursor *model.PageCursor, limit int) ([]*model.Article, error)
	UpdateArticle(ctx context.Context, article *model.Article) error
	DeleteArticle(ctx context.Context, articleID uuid.UUID) error
}

// Package usecase implements the business logic of the application.
package usecase

import (
	"context"
	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/domain/repository"

	"github.com/google/uuid"
)

// ArticleInteractor implements ArticleUsecase interface.
type ArticleInteractor struct {
	articleRepo repository.ArticleRepository
}

// NewArticleInteractor represents constructor of ArticleInteractor.
func NewArticleInteractor(
	articleRepo repository.ArticleRepository,
) *ArticleInteractor {
	return &ArticleInteractor{
		articleRepo: articleRepo,
	}
}

// ListArticlesByFeedID returns articles for the given feed starting after cursor (nil = first page).
func (i *ArticleInteractor) ListArticlesByFeedID(ctx context.Context, feedID uuid.UUID, cursor *model.PageCursor, limit int) ([]*model.Article, error) {
	const op = "ArticleInteractor.ListArticlesByFeedID"

	articles, err := i.articleRepo.ListArticlesByFeedID(ctx, feedID, cursor, limit)
	if err != nil {
		return nil, apperror.Wrap(err, op)
	}

	return articles, nil
}

// ListArticles returns articles starting after cursor (nil = first page).
func (i *ArticleInteractor) ListArticles(ctx context.Context, cursor *model.PageCursor, limit int) ([]*model.Article, error) {
	const op = "ArticleInteractor.ListArticles"

	articles, err := i.articleRepo.ListArticles(ctx, cursor, limit)
	if err != nil {
		return nil, apperror.Wrap(err, op)
	}

	return articles, nil
}

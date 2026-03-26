// Package usecase implements the business logic of the application.
package usecase

import (
	"context"
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

// GetArticlesByFeedID returns articles belonging to the given feed.
func (i *ArticleInteractor) GetArticlesByFeedID(ctx context.Context, feedID uuid.UUID) ([]*model.Article, error) {
	return i.articleRepo.GetArticlesByFeedID(ctx, feedID)
}

func (i *ArticleInteractor) GetAllArticles(ctx context.Context) ([]*model.Article, error) {
	return i.articleRepo.GetAllArticles(ctx)
}

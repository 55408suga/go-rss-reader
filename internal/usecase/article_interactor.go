// Package usecase implements the business logic of the application.
package usecase

import (
	"context"
	"log/slog"
	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/domain/repository"
	applogger "rss_reader/internal/infra/logger"

	"github.com/google/uuid"
)

// ArticleInteractor implements ArticleUsecase interface.
type ArticleInteractor struct {
	articleRepo repository.ArticleRepository
	logger      *slog.Logger
}

// NewArticleInteractor represents constructor of ArticleInteractor.
func NewArticleInteractor(
	articleRepo repository.ArticleRepository,
	logger *slog.Logger,
) *ArticleInteractor {
	if logger == nil {
		logger = slog.Default()
	}

	return &ArticleInteractor{
		articleRepo: articleRepo,
		logger:      logger,
	}
}

// GetArticlesByFeedID returns articles belonging to the given feed.
func (i *ArticleInteractor) GetArticlesByFeedID(ctx context.Context, feedID uuid.UUID) ([]*model.Article, error) {
	const op = "ArticleInteractor.GetArticlesByFeedID"

	articles, err := i.articleRepo.GetArticlesByFeedID(ctx, feedID)
	if err != nil {
		applogger.WithContext(ctx, i.logger).WarnContext(ctx,
			"get articles by feed id failed",
			"feed_id", feedID,
			"error", err,
		)
		return nil, apperror.Wrap(err, op)
	}

	return articles, nil
}

// GetAllArticles returns all stored articles.
func (i *ArticleInteractor) GetAllArticles(ctx context.Context) ([]*model.Article, error) {
	const op = "ArticleInteractor.GetAllArticles"

	articles, err := i.articleRepo.GetAllArticles(ctx)
	if err != nil {
		applogger.WithContext(ctx, i.logger).ErrorContext(ctx,
			"get all articles failed",
			"error", err,
		)
		return nil, apperror.Wrap(err, op)
	}

	return articles, nil
}

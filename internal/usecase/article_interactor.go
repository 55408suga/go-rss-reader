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
	feedRepo    repository.FeedRepository
	fetcher     ArticleFetcher
}

// NewArticleInteractor represents constructor of ArticleInteractor.
func NewArticleInteractor(
	articleRepo repository.ArticleRepository,
	feedRepo repository.FeedRepository,
	fetcher ArticleFetcher,
) *ArticleInteractor {
	return &ArticleInteractor{
		articleRepo: articleRepo,
		feedRepo:    feedRepo,
		fetcher:     fetcher,
	}
}

// GetArticlesByFeedID returns articles belonging to the given feed.
func (i *ArticleInteractor) GetArticlesByFeedID(ctx context.Context, feedID uuid.UUID) ([]*model.Article, error) {
	return i.feedRepo.GetArticles(ctx, feedID)
}

// RefreshArticles fetches latest articles for the given feed and saves them.
func (i *ArticleInteractor) RefreshArticles(ctx context.Context) error {
	feeds, err := i.feedRepo.GetAllFeeds(ctx)
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		articles, err := i.fetcher.FetchArticles(ctx, feed.ID, feed.FeedURL)
		if err != nil {
			return err
		}
		for _, article := range articles {
			if err := i.articleRepo.RegisterArticle(ctx, article); err != nil {
				return err
			}
		}
	}
	return nil
}

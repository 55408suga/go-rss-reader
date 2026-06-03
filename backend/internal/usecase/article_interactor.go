// Package usecase implements the business logic of the application.
package usecase

import (
	"context"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/domain/repository"
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

// articlePageCursor maps an article to its keyset position. Articles are
// ordered by (published_at DESC, id DESC), so the cursor pairs those fields.
func articlePageCursor(a *model.Article) model.PageCursor {
	return model.PageCursor{At: a.PublishedAt, ID: a.ID}
}

// ListArticlesByFeedID returns one keyset page of a feed's articles starting
// after cursor (nil = first page). It over-fetches by one row (limit+1) so
// paginate can report has_more and the next cursor in a single query.
func (i *ArticleInteractor) ListArticlesByFeedID(
	ctx context.Context,
	feedID uuid.UUID,
	cursor *model.PageCursor,
	limit int,
) (*model.Page[*model.Article], error) {
	const op = "ArticleInteractor.ListArticlesByFeedID"

	articles, err := i.articleRepo.ListArticlesByFeedID(ctx, feedID, cursor, limit+1)
	if err != nil {
		return nil, apperror.Wrap(err, op)
	}

	return paginate(articles, limit, articlePageCursor), nil
}

// ListArticles returns one keyset page of articles starting after cursor
// (nil = first page), over-fetching limit+1 to detect a further page.
func (i *ArticleInteractor) ListArticles(
	ctx context.Context,
	cursor *model.PageCursor,
	limit int,
) (*model.Page[*model.Article], error) {
	const op = "ArticleInteractor.ListArticles"

	articles, err := i.articleRepo.ListArticles(ctx, cursor, limit+1)
	if err != nil {
		return nil, apperror.Wrap(err, op)
	}

	return paginate(articles, limit, articlePageCursor), nil
}

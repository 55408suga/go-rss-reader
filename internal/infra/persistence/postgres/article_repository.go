package postgres

import (
	"context"
	"log/slog"
	"time"

	"rss_reader/internal/domain/model"
	"rss_reader/internal/infra/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ArticleRepository is a PostgreSQL-backed article repository implementation.
type ArticleRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewArticleRepository creates an ArticleRepository.
func NewArticleRepository(pool *pgxpool.Pool, logger *slog.Logger) *ArticleRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &ArticleRepository{
		pool:   pool,
		logger: logger,
	}
}

// querier returns a Queries instance that uses the transaction from context if available.
func (r *ArticleRepository) querier(ctx context.Context) *generated.Queries {
	if tx := TxFromContext(ctx); tx != nil {
		return generated.New(tx)
	}
	return generated.New(r.pool)
}

// SaveArticle persists an article.
func (r *ArticleRepository) SaveArticle(ctx context.Context, article *model.Article) error {
	const op = "ArticleRepository.SaveArticle"

	params := generated.SaveArticleParams{
		ID:          article.ID,
		Title:       article.Title,
		Description: article.Description,
		PublishedAt: article.PublishedAt,
		WebsiteUrl:  article.WebsiteURL,
		Content:     article.Content,
		FeedID:      article.FeedID,
		ExternalID:  article.ExternalID,
	}
	err := r.querier(ctx).SaveArticle(ctx, params)
	if err != nil {
		return wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return nil
}

// GetArticleByID retrieves an article by ID.
func (r *ArticleRepository) GetArticleByID(ctx context.Context, articleID uuid.UUID) (*model.Article, error) {
	const op = "ArticleRepository.GetArticleByID"

	article, err := r.querier(ctx).GetArticleByID(ctx, articleID)
	if err != nil {
		return nil, wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return newArticleModel(
		article.ID,
		article.Title,
		article.Description,
		article.PublishedAt,
		article.WebsiteUrl,
		article.Content,
		article.FeedID,
		article.ExternalID,
	), nil
}

// GetArticlesByFeedID retrieves articles for a specific feed.
func (r *ArticleRepository) GetArticlesByFeedID(ctx context.Context, feedID uuid.UUID) ([]*model.Article, error) {
	const op = "ArticleRepository.GetArticlesByFeedID"

	articles, err := r.querier(ctx).GetArticles(ctx, feedID)
	if err != nil {
		return nil, wrapAndLogDBError(ctx, r.logger, op, err)
	}

	articleModels := make([]*model.Article, 0, len(articles))
	for _, article := range articles {
		articleModels = append(articleModels, newArticleModel(
			article.ID,
			article.Title,
			article.Description,
			article.PublishedAt,
			article.WebsiteUrl,
			article.Content,
			article.FeedID,
			article.ExternalID,
		))
	}
	return articleModels, nil
}

// GetAllArticles retrieves all articles ordered by publish time.
func (r *ArticleRepository) GetAllArticles(ctx context.Context) ([]*model.Article, error) {
	const op = "ArticleRepository.GetAllArticles"

	articles, err := r.querier(ctx).GetAllArticles(ctx)
	if err != nil {
		return nil, wrapAndLogDBError(ctx, r.logger, op, err)
	}

	articleModels := make([]*model.Article, 0, len(articles))
	for _, article := range articles {
		articleModels = append(articleModels, newArticleModel(
			article.ID,
			article.Title,
			article.Description,
			article.PublishedAt,
			article.WebsiteUrl,
			article.Content,
			article.FeedID,
			article.ExternalID,
		))
	}
	return articleModels, nil
}

// UpdateArticle updates an existing article.
func (r *ArticleRepository) UpdateArticle(ctx context.Context, article *model.Article) error {
	const op = "ArticleRepository.UpdateArticle"

	params := generated.UpdateArticleParams{
		Title:       article.Title,
		Description: article.Description,
		PublishedAt: article.PublishedAt,
		WebsiteUrl:  article.WebsiteURL,
		Content:     article.Content,
		FeedID:      article.FeedID,
		ID:          article.ID,
	}
	err := r.querier(ctx).UpdateArticle(ctx, params)
	if err != nil {
		return wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return nil
}

// DeleteArticle removes an article by ID.
func (r *ArticleRepository) DeleteArticle(ctx context.Context, articleID uuid.UUID) error {
	const op = "ArticleRepository.DeleteArticle"

	err := r.querier(ctx).DeleteArticle(ctx, articleID)
	if err != nil {
		return wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return nil
}

func newArticleModel(
	id uuid.UUID,
	title string,
	description string,
	publishedAt time.Time,
	websiteURL string,
	content string,
	feedID uuid.UUID,
	externalID string,
) *model.Article {
	return &model.Article{
		ID:          id,
		Title:       title,
		Description: description,
		PublishedAt: publishedAt,
		WebsiteURL:  websiteURL,
		Content:     content,
		FeedID:      feedID,
		ExternalID:  externalID,
	}
}

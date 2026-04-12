package postgres

import (
	"context"
	"time"

	"rss_reader/internal/domain/model"
	"rss_reader/internal/infra/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ArticleRepository struct {
	pool *pgxpool.Pool
}

func NewArticleRepository(pool *pgxpool.Pool) *ArticleRepository {
	return &ArticleRepository{
		pool: pool,
	}
}

// querier returns a Queries instance that uses the transaction from context if available.
func (r *ArticleRepository) querier(ctx context.Context) *generated.Queries {
	if tx := TxFromContext(ctx); tx != nil {
		return generated.New(tx)
	}
	return generated.New(r.pool)
}

func (r *ArticleRepository) SaveArticle(ctx context.Context, article *model.Article) error {
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
	return r.querier(ctx).SaveArticle(ctx, params)
}

func (r *ArticleRepository) GetArticleByID(ctx context.Context, articleID uuid.UUID) (*model.Article, error) {
	article, err := r.querier(ctx).GetArticleByID(ctx, articleID)
	if err != nil {
		return nil, err
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

func (r *ArticleRepository) GetArticlesByFeedID(ctx context.Context, feedID uuid.UUID) ([]*model.Article, error) {
	articles, err := r.querier(ctx).GetArticles(ctx, feedID)
	if err != nil {
		return nil, err
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

func (r *ArticleRepository) GetAllArticles(ctx context.Context) ([]*model.Article, error) {
	articles, err := r.querier(ctx).GetAllArticles(ctx)
	if err != nil {
		return nil, err
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

func (r *ArticleRepository) UpdateArticle(ctx context.Context, article *model.Article) error {
	params := generated.UpdateArticleParams{
		Title:       article.Title,
		Description: article.Description,
		PublishedAt: article.PublishedAt,
		WebsiteUrl:  article.WebsiteURL,
		Content:     article.Content,
		FeedID:      article.FeedID,
		ID:          article.ID,
	}
	return r.querier(ctx).UpdateArticle(ctx, params)
}

func (r *ArticleRepository) DeleteArticle(ctx context.Context, articleID uuid.UUID) error {
	return r.querier(ctx).DeleteArticle(ctx, articleID)
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

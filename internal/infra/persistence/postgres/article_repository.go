package postgres

import (
	"context"

	"rss_reader/internal/domain/model"
	"rss_reader/internal/infra/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ArticleRepository struct {
	queries *generated.Queries
}

func NewArticleRepository(db *pgxpool.Pool) *ArticleRepository {
	return &ArticleRepository{
		queries: generated.New(db),
	}
}

func (r *ArticleRepository) RegisterArticle(ctx context.Context, article *model.Article) error {
	params := generated.RegisterArticleParams{
		ID:          article.ID,
		Title:       article.Title,
		Description: pgtype.Text{String: article.Description, Valid: article.Description != ""},
		PublishedAt: article.PublishedAt,
		WebsiteUrl:  article.WebsiteURL,
		Content:     pgtype.Text{String: article.Content, Valid: article.Content != ""},
		FeedID:      article.FeedID,
	}
	return r.queries.RegisterArticle(ctx, params)
}

func (r *ArticleRepository) GetArticle(ctx context.Context, articleID uuid.UUID) (*model.Article, error) {
	article, err := r.queries.GetArticleByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	return toArticleModel(article), nil
}

func (r *ArticleRepository) GetArticlesByFeedID(ctx context.Context, feedID uuid.UUID) ([]*model.Article, error) {
	articles, err := r.queries.GetArticles(ctx, feedID)
	if err != nil {
		return nil, err
	}

	articleModels := make([]*model.Article, 0, len(articles))
	for _, article := range articles {
		articleModels = append(articleModels, toArticleModel(article))
	}
	return articleModels, nil
}

func (r *ArticleRepository) GetAllArticles(ctx context.Context) ([]*model.Article, error) {
	articles, err := r.queries.GetAllArticles(ctx)
	if err != nil {
		return nil, err
	}

	articleModels := make([]*model.Article, 0, len(articles))
	for _, article := range articles {
		articleModels = append(articleModels, toArticleModel(article))
	}
	return articleModels, nil
}

func (r *ArticleRepository) UpdateArticle(ctx context.Context, article *model.Article) error {
	params := generated.UpdateArticleParams{
		Title:       article.Title,
		Description: pgtype.Text{String: article.Description, Valid: article.Description != ""},
		PublishedAt: article.PublishedAt,
		WebsiteUrl:  article.WebsiteURL,
		Content:     pgtype.Text{String: article.Content, Valid: article.Content != ""},
		FeedID:      article.FeedID,
		ID:          article.ID,
	}
	return r.queries.UpdateArticle(ctx, params)
}

func (r *ArticleRepository) DeleteArticle(ctx context.Context, articleID uuid.UUID) error {
	return r.queries.DeleteArticle(ctx, articleID)
}

func toArticleModel(article generated.Article) *model.Article {
	return &model.Article{
		ID:          article.ID,
		Title:       article.Title,
		Description: article.Description.String,
		PublishedAt: article.PublishedAt,
		WebsiteURL:  article.WebsiteUrl,
		Content:     article.Content.String,
		FeedID:      article.FeedID,
	}
}

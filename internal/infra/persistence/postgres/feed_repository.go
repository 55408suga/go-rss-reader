package postgres

import (
	"context"

	"rss_reader/internal/domain/model"
	"rss_reader/internal/infra/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedRepository struct {
	queries *generated.Queries
}

func NewFeedRepository(db *pgxpool.Pool) *FeedRepository {
	return &FeedRepository{
		queries: generated.New(db),
	}
}

func (r *FeedRepository) SaveFeed(ctx context.Context, feed *model.Feed) error {
	params := generated.SaveFeedParams{
		ID:          feed.ID,
		Title:       feed.Title,
		UpdatedAt:   feed.UpdatedAt,
		FeedUrl:     feed.FeedURL,
		WebsiteUrl:  feed.WebsiteURL,
		Description: pgtype.Text{String: feed.Description, Valid: feed.Description != ""},
		Language:    pgtype.Text{String: feed.Language, Valid: feed.Language != ""},
	}
	return r.queries.SaveFeed(ctx, params)
}

func (r *FeedRepository) GetFeed(ctx context.Context, feedID uuid.UUID) (*model.Feed, error) {
	feed, err := r.queries.GetFeedByID(ctx, feedID)
	if err != nil {
		return nil, err
	}

	return &model.Feed{
		ID:          feed.ID,
		Title:       feed.Title,
		UpdatedAt:   feed.UpdatedAt,
		FeedURL:     feed.FeedUrl,
		WebsiteURL:  feed.WebsiteUrl,
		Description: feed.Description.String,
		Language:    feed.Language.String,
	}, nil
}

func (r *FeedRepository) GetAllFeeds(ctx context.Context) ([]*model.Feed, error) {
	feeds, err := r.queries.GetAllFeeds(ctx)
	if err != nil {
		return nil, err
	}

	// convert generated.Feed to model.Feed
	feedModels := make([]*model.Feed, 0, len(feeds))
	for _, feed := range feeds {
		feedModel := &model.Feed{
			ID:          feed.ID,
			Title:       feed.Title,
			UpdatedAt:   feed.UpdatedAt,
			FeedURL:     feed.FeedUrl,
			WebsiteURL:  feed.WebsiteUrl,
			Description: feed.Description.String,
			Language:    feed.Language.String,
		}
		feedModels = append(feedModels, feedModel)
	}
	return feedModels, nil
}

func (r *FeedRepository) UpdateFeed(ctx context.Context, feed *model.Feed) error {
	params := generated.UpdateFeedParams{
		ID:          feed.ID,
		Title:       feed.Title,
		UpdatedAt:   feed.UpdatedAt,
		FeedUrl:     feed.FeedURL,
		WebsiteUrl:  feed.WebsiteURL,
		Description: pgtype.Text{String: feed.Description, Valid: feed.Description != ""},
		Language:    pgtype.Text{String: feed.Language, Valid: feed.Language != ""},
	}
	return r.queries.UpdateFeed(ctx, params)
}

func (r *FeedRepository) DeleteFeed(ctx context.Context, feedID uuid.UUID) error {
	return r.queries.DeleteFeed(ctx, feedID)
}

package postgres

import (
	"context"

	"rss_reader/internal/domain/model"
	"rss_reader/internal/infra/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedRepository struct {
	pool *pgxpool.Pool
}

func NewFeedRepository(pool *pgxpool.Pool) *FeedRepository {
	return &FeedRepository{
		pool: pool,
	}
}

// querier returns a Queries instance that uses the transaction from context if available.
func (r *FeedRepository) querier(ctx context.Context) *generated.Queries {
	if tx := TxFromContext(ctx); tx != nil {
		return generated.New(tx)
	}
	return generated.New(r.pool)
}

func (r *FeedRepository) SaveFeed(ctx context.Context, feed *model.Feed) error {
	params := generated.SaveFeedParams{
		ID:          feed.ID,
		Title:       feed.Title,
		UpdatedAt:   feed.UpdatedAt,
		FeedUrl:     feed.FeedURL,
		WebsiteUrl:  feed.WebsiteURL,
		Description: feed.Description,
		Language:    feed.Language,
	}
	return r.querier(ctx).SaveFeed(ctx, params)
}

func (r *FeedRepository) GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error) {
	feed, err := r.querier(ctx).GetFeedByID(ctx, feedID)
	if err != nil {
		return nil, err
	}

	return &model.Feed{
		ID:          feed.ID,
		Title:       feed.Title,
		UpdatedAt:   feed.UpdatedAt,
		FeedURL:     feed.FeedUrl,
		WebsiteURL:  feed.WebsiteUrl,
		Description: feed.Description,
		Language:    feed.Language,
	}, nil
}

func (r *FeedRepository) GetAllFeeds(ctx context.Context) ([]*model.Feed, error) {
	feeds, err := r.querier(ctx).GetAllFeeds(ctx)
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
			Description: feed.Description,
			Language:    feed.Language,
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
		Description: feed.Description,
		Language:    feed.Language,
	}
	return r.querier(ctx).UpdateFeed(ctx, params)
}

func (r *FeedRepository) DeleteFeed(ctx context.Context, feedID uuid.UUID) error {
	return r.querier(ctx).DeleteFeed(ctx, feedID)
}

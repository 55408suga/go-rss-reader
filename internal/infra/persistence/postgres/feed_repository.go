package postgres

import (
	"context"
	"log/slog"

	"rss_reader/internal/domain/model"
	"rss_reader/internal/infra/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FeedRepository is a PostgreSQL-backed feed repository implementation.
type FeedRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewFeedRepository creates a FeedRepository.
func NewFeedRepository(pool *pgxpool.Pool, logger *slog.Logger) *FeedRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &FeedRepository{
		pool:   pool,
		logger: logger,
	}
}

// querier returns a Queries instance that uses the transaction from context if available.
func (r *FeedRepository) querier(ctx context.Context) *generated.Queries {
	if tx := TxFromContext(ctx); tx != nil {
		return generated.New(tx)
	}
	return generated.New(r.pool)
}

// SaveFeed persists a feed.
func (r *FeedRepository) SaveFeed(ctx context.Context, feed *model.Feed) error {
	const op = "FeedRepository.SaveFeed"

	params := generated.SaveFeedParams{
		ID:          feed.ID,
		Title:       feed.Title,
		UpdatedAt:   feed.UpdatedAt,
		FeedUrl:     feed.FeedURL,
		WebsiteUrl:  feed.WebsiteURL,
		Description: feed.Description,
		Language:    feed.Language,
	}
	err := r.querier(ctx).SaveFeed(ctx, params)
	if err != nil {
		return wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return nil
}

// GetFeedByID retrieves a feed by ID.
func (r *FeedRepository) GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error) {
	const op = "FeedRepository.GetFeedByID"

	feed, err := r.querier(ctx).GetFeedByID(ctx, feedID)
	if err != nil {
		return nil, wrapAndLogDBError(ctx, r.logger, op, err)
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

// GetAllFeeds retrieves all feeds ordered by registration date.
func (r *FeedRepository) GetAllFeeds(ctx context.Context) ([]*model.Feed, error) {
	const op = "FeedRepository.GetAllFeeds"

	feeds, err := r.querier(ctx).GetAllFeeds(ctx)
	if err != nil {
		return nil, wrapAndLogDBError(ctx, r.logger, op, err)
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

// UpdateFeed updates an existing feed.
func (r *FeedRepository) UpdateFeed(ctx context.Context, feed *model.Feed) error {
	const op = "FeedRepository.UpdateFeed"

	params := generated.UpdateFeedParams{
		ID:          feed.ID,
		Title:       feed.Title,
		UpdatedAt:   feed.UpdatedAt,
		FeedUrl:     feed.FeedURL,
		WebsiteUrl:  feed.WebsiteURL,
		Description: feed.Description,
		Language:    feed.Language,
	}
	err := r.querier(ctx).UpdateFeed(ctx, params)
	if err != nil {
		return wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return nil
}

// DeleteFeed removes a feed by ID.
func (r *FeedRepository) DeleteFeed(ctx context.Context, feedID uuid.UUID) error {
	const op = "FeedRepository.DeleteFeed"

	err := r.querier(ctx).DeleteFeed(ctx, feedID)
	if err != nil {
		return wrapAndLogDBError(ctx, r.logger, op, err)
	}

	return nil
}

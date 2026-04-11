// Package repository provides abstracts of database operations.
package repository

import (
	"context"
	"rss_reader/internal/domain/model"

	"github.com/google/uuid"
)

// FeedRepository defines the interface for feed repository.
type FeedRepository interface {
	SaveFeed(ctx context.Context, feed *model.Feed) error
	GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error)
	GetAllFeeds(ctx context.Context) ([]*model.Feed, error)
	UpdateFeed(ctx context.Context, feed *model.Feed) error
	DeleteFeed(ctx context.Context, feedID uuid.UUID) error
}

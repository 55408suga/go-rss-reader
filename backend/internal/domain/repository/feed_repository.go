// Package repository provides abstracts of database operations.
package repository

import (
	"context"

	"github.com/google/uuid"

	"rss_reader/internal/domain/model"
)

// FeedRepository defines the interface for feed repository.
type FeedRepository interface {
	SaveFeed(ctx context.Context, feed *model.Feed) error
	GetFeedByID(ctx context.Context, feedID uuid.UUID) (*model.Feed, error)
	CheckFeedExistsByURL(ctx context.Context, feedURL string) (bool, error)
	// GetFeedByWebsiteURL looks up a feed by any of the given website URL
	// variants. Returns not_found when no variant matches.
	GetFeedByWebsiteURL(ctx context.Context, websiteURLs []string) (*model.Feed, error)
	ListFeeds(ctx context.Context, cursor *model.PageCursor, limit int) ([]*model.Feed, error)
	UpdateFeed(ctx context.Context, feed *model.Feed) error
	DeleteFeed(ctx context.Context, feedID uuid.UUID) error
}

// Package model defines the core domain entities for the RSS reader,
// including Feed, Article, FetchStatus, and FeedCursor.
//
// Entities in this package have no outward dependencies; they are
// constructed and consumed by the usecase layer and persisted by the
// infra layer.
package model

import (
	"time"

	"github.com/google/uuid"
)

// Feed represents an RSS feed
type Feed struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	FeedURL      string    `json:"feed_url"`
	WebsiteURL   string    `json:"website_url"`
	Description  string    `json:"description"`
	RegisteredAt time.Time `json:"registered_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Language     string    `json:"language"`
}

// NewFeed creates a new feed instance with generating uuidv7
func NewFeed(title, feedURL, websiteURL, description, language string, updatedAt time.Time) (*Feed, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	registeredAt := time.Now().UTC()

	return &Feed{
		ID:           id,
		Title:        title,
		FeedURL:      feedURL,
		WebsiteURL:   websiteURL,
		Description:  description,
		RegisteredAt: registeredAt,
		Language:     language,
		UpdatedAt:    updatedAt,
	}, nil
}

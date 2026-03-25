package model

import (
	"time"

	"github.com/google/uuid"
)

// Feed represents an RSS feed
type Feed struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	FeedURL     string    `json:"feed_url"`
	WebsiteURL  string    `json:"website_url"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
	Language    string    `json:"language"`
}

// NewFeed creates a new feed instance with generating uuidv7
func NewFeed(title, feedURL, websiteURL, description, language string, updatedAt time.Time) (*Feed, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &Feed{
		ID:          id,
		Title:       title,
		FeedURL:     feedURL,
		WebsiteURL:  websiteURL,
		Description: description,
		Language:    language,
		UpdatedAt:   updatedAt,
	}, nil
}

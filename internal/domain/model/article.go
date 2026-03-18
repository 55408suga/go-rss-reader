// Package model provides domain model for RSS reader
package model

import (
	"time"

	"github.com/google/uuid"
)

// Article represents an article in an RSS feed
type Article struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	WebsiteURL  string    `json:"website_url"`
	PublishedAt time.Time `json:"published_at"`
	FeedID      uuid.UUID `json:"feed_id"`
}

// NewArticle creates a new article instance with generating uuidv7
func NewArticle(title, description, content, websiteURL string, publishedAt time.Time, feedID uuid.UUID) (*Article, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &Article{
		ID:          id,
		Title:       title,
		Description: description,
		Content:     content,
		WebsiteURL:  websiteURL,
		PublishedAt: publishedAt,
		FeedID:      feedID,
	}, nil
}

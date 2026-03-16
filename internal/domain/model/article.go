// Package model provides domain model for RSS reader
package model

import (
	"time"
)

// Article represents an article in an RSS feed
type Article struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	WebsiteURL  string    `json:"website_url"`
	PublishedAt time.Time `json:"published_at"`
	FeedID      string    `json:"feed_id"`
}

package model

import (
	"time"
)

// Feed represents an RSS feed
type Feed struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	FeedURL     string    `json:"feed_url"`
	WebsiteURL  string    `json:"website_url"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
	Language    string    `json:"language"`
	Articles    []Article `json:"articles"`
}

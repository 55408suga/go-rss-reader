package model

import (
	"time"

	"github.com/google/uuid"
)

type FeedCursor struct {
	ETag         *string
	LastModified *time.Time
}
type FetchStatus struct {
	FeedID             uuid.UUID `json:"feed_id"`
	LastFetchedAt      time.Time `json:"last_fetched_at"`
	NextFetchAt        time.Time `json:"next_fetch_at"`
	StatusCode         int       `json:"status_code"`
	ErrorMessage       *string   `json:"error_message,omitempty"`
	FeedCursor         FeedCursor
	FetchIntervalHours int `json:"fetch_interval_hours"`
	FailureCount       int `json:"failure_count"`
}

package model

import (
	"time"

	"github.com/google/uuid"
)

// PageCursor identifies a position in a keyset-paginated result set.
type PageCursor struct {
	At time.Time
	ID uuid.UUID
}

// FeedCursor stores HTTP cache headers used for conditional feed fetch.
type FeedCursor struct {
	ETag         *string
	LastModified *time.Time
}

// FetchStatus tracks fetch timing and health for a feed.
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

// DueFeed bundles a feed's URL with its current fetch status for scheduled refresh.
type DueFeed struct {
	FeedURL string
	Status  *FetchStatus
}

const (
	defaultFetchIntervalHours = 24
	defaultFetchStatusCode    = 200
)

// NewFetchStatusWith creates FetchStatus with all fields explicitly specified.
func NewFetchStatusWith(
	feedID uuid.UUID,
	lastFetchedAt time.Time,
	nextFetchAt time.Time,
	statusCode int,
	errorMessage *string,
	feedCursor FeedCursor,
	fetchIntervalHours int,
	failureCount int,
) *FetchStatus {
	return &FetchStatus{
		FeedID:             feedID,
		LastFetchedAt:      lastFetchedAt,
		NextFetchAt:        nextFetchAt,
		StatusCode:         statusCode,
		ErrorMessage:       errorMessage,
		FeedCursor:         feedCursor,
		FetchIntervalHours: fetchIntervalHours,
		FailureCount:       failureCount,
	}
}

// NewFetchStatus creates FetchStatus with default scheduling values.
func NewFetchStatus(feedID uuid.UUID, feedCursor FeedCursor) *FetchStatus {
	lastFetchedAt := time.Now().UTC()

	return NewFetchStatusWith(
		feedID,
		lastFetchedAt,
		lastFetchedAt.Add(time.Duration(defaultFetchIntervalHours)*time.Hour),
		defaultFetchStatusCode,
		nil,
		feedCursor,
		defaultFetchIntervalHours,
		0,
	)
}

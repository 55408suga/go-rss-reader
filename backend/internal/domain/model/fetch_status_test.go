package model

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestNewFetchStatusWith(t *testing.T) {
	t.Parallel()

	feedID := uuid.New()
	last := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	next := last.Add(12 * time.Hour)
	errMsg := "fetch failed"
	etag := `"v1"`
	lastMod := time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC)

	// NewFetchStatusWith is a pure constructor: every field is caller-supplied,
	// so the table feeds want's fields straight back in and expects them verbatim.
	tests := []struct {
		name string
		want *FetchStatus
	}{
		{
			name: "successful fetch carries cursor and zero failures",
			want: &FetchStatus{
				FeedID:             feedID,
				LastFetchedAt:      last,
				NextFetchAt:        next,
				StatusCode:         200,
				ErrorMessage:       nil,
				FeedCursor:         FeedCursor{ETag: &etag, LastModified: &lastMod},
				FetchIntervalHours: 12,
				FailureCount:       0,
			},
		},
		{
			name: "failed fetch records error message and failure count",
			want: &FetchStatus{
				FeedID:             feedID,
				LastFetchedAt:      last,
				NextFetchAt:        next,
				StatusCode:         0,
				ErrorMessage:       &errMsg,
				FeedCursor:         FeedCursor{},
				FetchIntervalHours: 12,
				FailureCount:       3,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := NewFetchStatusWith(
				tc.want.FeedID,
				tc.want.LastFetchedAt,
				tc.want.NextFetchAt,
				tc.want.StatusCode,
				tc.want.ErrorMessage,
				tc.want.FeedCursor,
				tc.want.FetchIntervalHours,
				tc.want.FailureCount,
			)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("NewFetchStatusWith() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewFetchStatusAppliesSchedulingDefaults(t *testing.T) {
	t.Parallel()

	feedID := uuid.New()
	etag := `"abc"`
	cursor := FeedCursor{ETag: &etag}

	before := time.Now().UTC()
	got := NewFetchStatus(feedID, cursor)
	after := time.Now().UTC()

	if got.FeedID != feedID {
		t.Errorf("FeedID = %v, want %v", got.FeedID, feedID)
	}
	if got.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200 (default)", got.StatusCode)
	}
	if got.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want 0", got.FailureCount)
	}
	if got.FetchIntervalHours != 24 {
		t.Errorf("FetchIntervalHours = %d, want 24 (default)", got.FetchIntervalHours)
	}
	if got.ErrorMessage != nil {
		t.Errorf("ErrorMessage = %q, want nil", *got.ErrorMessage)
	}

	assertWithinUTCWindow(t, "LastFetchedAt", got.LastFetchedAt, before, after)

	// Invariant: NextFetchAt is exactly one default interval (24h) after
	// LastFetchedAt, independent of when the test runs.
	if d := got.NextFetchAt.Sub(got.LastFetchedAt); d != 24*time.Hour {
		t.Errorf("NextFetchAt - LastFetchedAt = %v, want 24h", d)
	}

	if diff := cmp.Diff(cursor, got.FeedCursor); diff != "" {
		t.Errorf("FeedCursor mismatch (-want +got):\n%s", diff)
	}
}

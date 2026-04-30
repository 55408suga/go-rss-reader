package postgres

import (
	"testing"
	"time"

	"rss_reader/internal/infra/persistence/postgres/generated"

	"github.com/google/uuid"
)

func TestGeneratedFetchStatusParamsUseInt(t *testing.T) {
	params := generated.SaveFeedFetchStatusParams{
		StatusCode:         503,
		FetchIntervalHours: 12,
		FailureCount:       3,
	}

	if _, ok := any(params.StatusCode).(int); !ok {
		t.Fatalf("StatusCode should use int, got %T", params.StatusCode)
	}
	if _, ok := any(params.FetchIntervalHours).(int); !ok {
		t.Fatalf("FetchIntervalHours should use int, got %T", params.FetchIntervalHours)
	}
	if _, ok := any(params.FailureCount).(int); !ok {
		t.Fatalf("FailureCount should use int, got %T", params.FailureCount)
	}

	dueParams := generated.GetDueFeedFetchStatusesParams{
		Now:   time.Now(),
		Limit: 10,
	}

	if _, ok := any(dueParams.Limit).(int); !ok {
		t.Fatalf("Limit should use int, got %T", dueParams.Limit)
	}
}

func TestGeneratedSaveFeedParamsAcceptsRegisteredAt(t *testing.T) {
	registeredAt := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)

	params := generated.SaveFeedParams{
		RegisteredAt: registeredAt,
	}

	if params.RegisteredAt != registeredAt {
		t.Fatalf("RegisteredAt mismatch: got %v want %v", params.RegisteredAt, registeredAt)
	}
}

func TestToFetchStatusModelPreservesGeneratedIntFields(t *testing.T) {
	lastModified := time.Unix(1_700_000_000, 0).UTC()
	errorMessage := "temporary failure"
	etag := "etag-value"
	lastFetchedAt := time.Unix(1_700_100_000, 0).UTC()
	nextFetchAt := lastFetchedAt.Add(30 * time.Minute)

	status := generated.FeedFetchStatus{
		FeedID:             uuid.New(),
		LastFetchedAt:      lastFetchedAt,
		NextFetchAt:        nextFetchAt,
		StatusCode:         429,
		ErrorMessage:       &errorMessage,
		LastModified:       &lastModified,
		Etag:               &etag,
		FetchIntervalHours: 6,
		FailureCount:       4,
	}

	got := toFetchStatusModel(status)

	if got.StatusCode != status.StatusCode {
		t.Fatalf("StatusCode mismatch: got %d want %d", got.StatusCode, status.StatusCode)
	}
	if got.FetchIntervalHours != status.FetchIntervalHours {
		t.Fatalf("FetchIntervalHours mismatch: got %d want %d", got.FetchIntervalHours, status.FetchIntervalHours)
	}
	if got.FailureCount != status.FailureCount {
		t.Fatalf("FailureCount mismatch: got %d want %d", got.FailureCount, status.FailureCount)
	}
	if got.FeedCursor.LastModified != status.LastModified {
		t.Fatalf("LastModified mismatch: got %v want %v", got.FeedCursor.LastModified, status.LastModified)
	}
	if got.FeedCursor.ETag != status.Etag {
		t.Fatalf("ETag mismatch: got %v want %v", got.FeedCursor.ETag, status.Etag)
	}
}

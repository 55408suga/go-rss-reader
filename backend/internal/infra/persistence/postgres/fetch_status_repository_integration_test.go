//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"rss_reader/internal/domain/model"
)

func TestFetchStatusRepositoryUpsert(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	feed := saveParentFeed(t, "fs")
	repo := NewFetchStatusRepository(testPool, quietLogger())

	etag := `"v1"`
	lastMod := time.Now().UTC().Truncate(time.Second)
	now := time.Now().UTC()
	initial := model.NewFetchStatusWith(
		feed.ID, now, now.Add(12*time.Hour),
		200, nil, model.FeedCursor{ETag: &etag, LastModified: &lastMod}, 12, 0,
	)
	if err := repo.SaveFetchStatus(ctx, initial); err != nil {
		t.Fatalf("SaveFetchStatus initial: %v", err)
	}

	got, err := repo.GetFetchStatusByFeedID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("GetFetchStatusByFeedID: %v", err)
	}
	if got.StatusCode != 200 || got.FailureCount != 0 {
		t.Errorf("initial status = %+v, want code 200 failures 0", got)
	}
	if got.FeedCursor.ETag == nil || *got.FeedCursor.ETag != etag {
		t.Errorf("ETag = %v, want %q", got.FeedCursor.ETag, etag)
	}
	if got.FeedCursor.LastModified == nil || !got.FeedCursor.LastModified.Equal(lastMod) {
		t.Errorf("LastModified = %v, want %v", got.FeedCursor.LastModified, lastMod)
	}

	// Upsert on feed_id (ON CONFLICT (feed_id) DO UPDATE) with new values.
	failMsg := "boom"
	updated := model.NewFetchStatusWith(
		feed.ID, now, now.Add(6*time.Hour),
		503, &failMsg, model.FeedCursor{}, 6, 3,
	)
	if err := repo.SaveFetchStatus(ctx, updated); err != nil {
		t.Fatalf("SaveFetchStatus update: %v", err)
	}

	got, err = repo.GetFetchStatusByFeedID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("GetFetchStatusByFeedID after upsert: %v", err)
	}
	if got.StatusCode != 503 || got.FailureCount != 3 {
		t.Errorf("after upsert status = %+v, want code 503 failures 3", got)
	}
	if got.ErrorMessage == nil || *got.ErrorMessage != "boom" {
		t.Errorf("ErrorMessage = %v, want boom", got.ErrorMessage)
	}
	if got.FeedCursor.ETag != nil {
		t.Errorf("ETag after upsert = %v, want nil", got.FeedCursor.ETag)
	}
}

func TestFetchStatusRepositoryGetDueFeeds(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := NewFetchStatusRepository(testPool, quietLogger())

	now := time.Now().UTC()
	dueFeed := saveParentFeed(t, "due")
	notDueFeed := saveParentFeed(t, "future")

	// Due: next_fetch_at in the past.
	due := model.NewFetchStatusWith(
		dueFeed.ID, now.Add(-time.Hour), now.Add(-time.Minute),
		200, nil, model.FeedCursor{}, 12, 0,
	)
	if err := repo.SaveFetchStatus(ctx, due); err != nil {
		t.Fatalf("save due: %v", err)
	}
	// Not due: next_fetch_at in the future.
	future := model.NewFetchStatusWith(
		notDueFeed.ID, now, now.Add(time.Hour),
		200, nil, model.FeedCursor{}, 12, 0,
	)
	if err := repo.SaveFetchStatus(ctx, future); err != nil {
		t.Fatalf("save future: %v", err)
	}

	dueFeeds, err := repo.GetDueFeeds(ctx, now, 10)
	if err != nil {
		t.Fatalf("GetDueFeeds: %v", err)
	}
	if len(dueFeeds) != 1 {
		t.Fatalf("due feeds = %d, want 1 (only the past-due feed)", len(dueFeeds))
	}
	if dueFeeds[0].Status.FeedID != dueFeed.ID {
		t.Errorf("due feed id = %s, want %s", dueFeeds[0].Status.FeedID, dueFeed.ID)
	}
	if dueFeeds[0].FeedURL != dueFeed.FeedURL {
		t.Errorf("due feed url = %q, want %q (joined from feeds)", dueFeeds[0].FeedURL, dueFeed.FeedURL)
	}
}

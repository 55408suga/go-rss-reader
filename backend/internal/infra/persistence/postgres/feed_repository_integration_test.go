//go:build integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestFeedRepositoryRoundTrip(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := NewFeedRepository(testPool, quietLogger())

	feed := makeFeed(t, "a")
	if err := repo.SaveFeed(ctx, feed); err != nil {
		t.Fatalf("SaveFeed: %v", err)
	}

	got, err := repo.GetFeedByID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID: %v", err)
	}
	if got.Title != feed.Title || got.FeedURL != feed.FeedURL || got.WebsiteURL != feed.WebsiteURL {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, feed)
	}

	exists, err := repo.CheckFeedExistsByURL(ctx, feed.FeedURL)
	if err != nil {
		t.Fatalf("CheckFeedExistsByURL(existing): %v", err)
	}
	if !exists {
		t.Error("CheckFeedExistsByURL(existing) = false, want true")
	}
	missing, err := repo.CheckFeedExistsByURL(ctx, "https://absent.example/feed.xml")
	if err != nil {
		t.Fatalf("CheckFeedExistsByURL(absent): %v", err)
	}
	if missing {
		t.Error("CheckFeedExistsByURL(absent) = true, want false")
	}
}

func TestFeedRepositoryDuplicateURLConflict(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := NewFeedRepository(testPool, quietLogger())

	first := makeFeed(t, "dup")
	if err := repo.SaveFeed(ctx, first); err != nil {
		t.Fatalf("SaveFeed first: %v", err)
	}

	// Same URLs, different ID -> unique violation classified as conflict.
	second := makeFeed(t, "dup")
	err := repo.SaveFeed(ctx, second)
	assertAppErrorCode(t, err, apperror.CodeConflict)
}

func TestFeedRepositoryGetMissingNotFound(t *testing.T) {
	resetDB(t)
	repo := NewFeedRepository(testPool, quietLogger())
	_, err := repo.GetFeedByID(context.Background(), uuid.New())
	assertAppErrorCode(t, err, apperror.CodeNotFound)
}

func TestFeedRepositoryUpdateAndDelete(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := NewFeedRepository(testPool, quietLogger())

	feed := makeFeed(t, "ud")
	if err := repo.SaveFeed(ctx, feed); err != nil {
		t.Fatalf("SaveFeed: %v", err)
	}

	feed.Title = "Updated Title"
	if err := repo.UpdateFeed(ctx, feed); err != nil {
		t.Fatalf("UpdateFeed: %v", err)
	}
	got, err := repo.GetFeedByID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID after update: %v", err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("Title after update = %q, want Updated Title", got.Title)
	}

	if err := repo.DeleteFeed(ctx, feed.ID); err != nil {
		t.Fatalf("DeleteFeed: %v", err)
	}
	_, err = repo.GetFeedByID(ctx, feed.ID)
	assertAppErrorCode(t, err, apperror.CodeNotFound)
}

func TestFeedRepositoryListKeysetPagination(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := NewFeedRepository(testPool, quietLogger())

	const total = 5
	for i := range total {
		if err := repo.SaveFeed(ctx, makeFeed(t, fmt.Sprintf("pg%d", i))); err != nil {
			t.Fatalf("SaveFeed %d: %v", i, err)
		}
	}

	// Page through 2 at a time; the (registered_at, id) cursor must visit every
	// feed exactly once even when registered_at values tie.
	seen := make(map[uuid.UUID]struct{})
	var cursor *model.PageCursor
	for {
		page, err := repo.ListFeeds(ctx, cursor, 2)
		if err != nil {
			t.Fatalf("ListFeeds: %v", err)
		}
		if len(page) == 0 {
			break
		}
		for _, f := range page {
			if _, dup := seen[f.ID]; dup {
				t.Fatalf("feed %s returned on more than one page", f.ID)
			}
			seen[f.ID] = struct{}{}
		}
		last := page[len(page)-1]
		cursor = &model.PageCursor{At: last.RegisteredAt, ID: last.ID}
		if len(page) < 2 {
			break
		}
	}

	if len(seen) != total {
		t.Errorf("paginated through %d feeds, want %d", len(seen), total)
	}
}

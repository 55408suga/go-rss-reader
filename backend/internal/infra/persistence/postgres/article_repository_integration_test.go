//go:build integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
)

func TestArticleRepositorySaveIsIdempotent(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	feed := saveParentFeed(t, "art")
	repo := NewArticleRepository(testPool, quietLogger())

	article := makeArticle(t, feed.ID, "ext-1")
	if err := repo.SaveArticle(ctx, article); err != nil {
		t.Fatalf("SaveArticle first: %v", err)
	}

	// Same (feed_id, external_id), different row ID -> ON CONFLICT DO NOTHING.
	dup := makeArticle(t, feed.ID, "ext-1")
	if err := repo.SaveArticle(ctx, dup); err != nil {
		t.Fatalf("SaveArticle duplicate: %v (conflict must be ignored, not returned)", err)
	}

	got, err := repo.ListArticlesByFeedID(ctx, feed.ID, nil, 10)
	if err != nil {
		t.Fatalf("ListArticlesByFeedID: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("articles after duplicate save = %d, want 1 (idempotent)", len(got))
	}
	// DO NOTHING keeps the first-written row.
	if got[0].ID != article.ID {
		t.Errorf("stored article ID = %s, want the first-written %s", got[0].ID, article.ID)
	}
}

func TestArticleRepositoryGetByIDAndNotFound(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	feed := saveParentFeed(t, "g")
	repo := NewArticleRepository(testPool, quietLogger())

	article := makeArticle(t, feed.ID, "ext-1")
	if err := repo.SaveArticle(ctx, article); err != nil {
		t.Fatalf("SaveArticle: %v", err)
	}

	got, err := repo.GetArticleByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("GetArticleByID: %v", err)
	}
	if got.ExternalID != "ext-1" || got.FeedID != feed.ID {
		t.Errorf("round-trip mismatch: got %+v", got)
	}

	_, err = repo.GetArticleByID(ctx, uuid.New())
	assertAppErrorCode(t, err, apperror.CodeNotFound)
}

func TestArticleRepositoryListScoping(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	repo := NewArticleRepository(testPool, quietLogger())

	feedA := saveParentFeed(t, "A")
	feedB := saveParentFeed(t, "B")

	for i := range 3 {
		if err := repo.SaveArticle(ctx, makeArticle(t, feedA.ID, fmt.Sprintf("a%d", i))); err != nil {
			t.Fatalf("save A%d: %v", i, err)
		}
	}
	for i := range 2 {
		if err := repo.SaveArticle(ctx, makeArticle(t, feedB.ID, fmt.Sprintf("b%d", i))); err != nil {
			t.Fatalf("save B%d: %v", i, err)
		}
	}

	byFeedA, err := repo.ListArticlesByFeedID(ctx, feedA.ID, nil, 10)
	if err != nil {
		t.Fatalf("ListArticlesByFeedID(A): %v", err)
	}
	if len(byFeedA) != 3 {
		t.Errorf("feed A articles = %d, want 3", len(byFeedA))
	}
	for _, a := range byFeedA {
		if a.FeedID != feedA.ID {
			t.Errorf("ListArticlesByFeedID(A) returned an article for feed %s", a.FeedID)
		}
	}

	all, err := repo.ListArticles(ctx, nil, 10)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("all articles = %d, want 5", len(all))
	}
}

package usecase

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRefreshOne(t *testing.T) {
	t.Parallel()

	feedID := uuid.New()
	oldEtag := `"old"`
	newEtag := `"new"`

	// makeDue returns a due feed that has already failed twice and carries an
	// existing cursor, so each branch can assert how those values evolve.
	makeDue := func() *model.DueFeed {
		return &model.DueFeed{
			FeedURL: "https://x/feed.xml",
			Status: &model.FetchStatus{
				FeedID:       feedID,
				FailureCount: 2,
				FeedCursor:   model.FeedCursor{ETag: &oldEtag},
			},
		}
	}

	t.Run("fetch error persists failed status with incremented failure count", func(t *testing.T) {
		t.Parallel()
		statusRepo := &fakeFetchStatusRepo{}
		articleRepo := &fakeArticleRepo{}
		fetcher := &fakeFetcher{withCursorFunc: func(
			_ string, _ *model.FeedCursor,
		) (*model.Feed, []*model.Article, *model.FeedCursor, error) {
			return nil, nil, nil, apperror.NewExternalUnavailable("gw", "down", nil)
		}}
		interactor := NewFeedJobInteractor(fetcher, articleRepo, statusRepo, fakeTxManager{}, quietLogger())

		err := interactor.refreshOne(context.Background(), makeDue())

		assertAppErrorCode(t, err, apperror.CodeExternalUnavailable)
		if len(articleRepo.savedArticles) != 0 {
			t.Errorf("SaveArticle calls = %d, want 0", len(articleRepo.savedArticles))
		}
		if len(statusRepo.savedStatuses) != 1 {
			t.Fatalf("SaveFetchStatus calls = %d, want 1 (failure must still advance NextFetchAt)",
				len(statusRepo.savedStatuses))
		}
		got := statusRepo.savedStatuses[0]
		if got.StatusCode != 0 {
			t.Errorf("StatusCode = %d, want 0", got.StatusCode)
		}
		if got.FailureCount != 3 {
			t.Errorf("FailureCount = %d, want 3 (2+1)", got.FailureCount)
		}
		if got.ErrorMessage == nil {
			t.Error("ErrorMessage = nil, want a recorded message")
		}
	})

	t.Run("not modified persists status 304 and resets failures", func(t *testing.T) {
		t.Parallel()
		statusRepo := &fakeFetchStatusRepo{}
		articleRepo := &fakeArticleRepo{}
		newCursor := &model.FeedCursor{ETag: &newEtag}
		fetcher := &fakeFetcher{withCursorFunc: func(
			_ string, _ *model.FeedCursor,
		) (*model.Feed, []*model.Article, *model.FeedCursor, error) {
			return nil, nil, newCursor, nil // feed == nil signals 304 Not Modified
		}}
		interactor := NewFeedJobInteractor(fetcher, articleRepo, statusRepo, fakeTxManager{}, quietLogger())

		if err := interactor.refreshOne(context.Background(), makeDue()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(articleRepo.savedArticles) != 0 {
			t.Errorf("SaveArticle calls = %d, want 0", len(articleRepo.savedArticles))
		}
		if len(statusRepo.savedStatuses) != 1 {
			t.Fatalf("SaveFetchStatus calls = %d, want 1", len(statusRepo.savedStatuses))
		}
		got := statusRepo.savedStatuses[0]
		if got.StatusCode != 304 {
			t.Errorf("StatusCode = %d, want 304", got.StatusCode)
		}
		if got.FailureCount != 0 {
			t.Errorf("FailureCount = %d, want 0", got.FailureCount)
		}
		if got.FeedCursor.ETag == nil || *got.FeedCursor.ETag != newEtag {
			t.Errorf("FeedCursor.ETag = %v, want %q", got.FeedCursor.ETag, newEtag)
		}
	})

	t.Run("success persists articles and status 200", func(t *testing.T) {
		t.Parallel()
		statusRepo := &fakeFetchStatusRepo{}
		articleRepo := &fakeArticleRepo{}
		newCursor := &model.FeedCursor{ETag: &newEtag}
		fetcher := &fakeFetcher{withCursorFunc: func(
			_ string, _ *model.FeedCursor,
		) (*model.Feed, []*model.Article, *model.FeedCursor, error) {
			feed := &model.Feed{ID: feedID}
			arts := []*model.Article{{ID: uuid.New()}, {ID: uuid.New()}}
			return feed, arts, newCursor, nil
		}}
		interactor := NewFeedJobInteractor(fetcher, articleRepo, statusRepo, fakeTxManager{}, quietLogger())

		if err := interactor.refreshOne(context.Background(), makeDue()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(articleRepo.savedArticles) != 2 {
			t.Errorf("SaveArticle calls = %d, want 2", len(articleRepo.savedArticles))
		}
		if len(statusRepo.savedStatuses) != 1 {
			t.Fatalf("SaveFetchStatus calls = %d, want 1", len(statusRepo.savedStatuses))
		}
		got := statusRepo.savedStatuses[0]
		if got.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", got.StatusCode)
		}
		if got.FailureCount != 0 {
			t.Errorf("FailureCount = %d, want 0", got.FailureCount)
		}
	})
}

func TestRefreshDueFeeds(t *testing.T) {
	t.Parallel()

	t.Run("no due feeds is a no-op", func(t *testing.T) {
		t.Parallel()
		statusRepo := &fakeFetchStatusRepo{dueFeeds: nil}
		interactor := NewFeedJobInteractor(
			&fakeFetcher{}, &fakeArticleRepo{}, statusRepo, fakeTxManager{}, quietLogger(),
		)
		if err := interactor.RefreshDueFeeds(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(statusRepo.savedStatuses) != 0 {
			t.Errorf("SaveFetchStatus calls = %d, want 0", len(statusRepo.savedStatuses))
		}
	})

	t.Run("GetDueFeeds error is wrapped", func(t *testing.T) {
		t.Parallel()
		statusRepo := &fakeFetchStatusRepo{dueErr: apperror.NewInternal("repo", "boom", nil)}
		interactor := NewFeedJobInteractor(
			&fakeFetcher{}, &fakeArticleRepo{}, statusRepo, fakeTxManager{}, quietLogger(),
		)
		err := interactor.RefreshDueFeeds(context.Background())
		assertAppErrorCode(t, err, apperror.CodeInternal)
	})

	t.Run("one failing feed does not stop the batch", func(t *testing.T) {
		t.Parallel()

		due := func(url string) *model.DueFeed {
			return &model.DueFeed{
				FeedURL: url,
				Status:  &model.FetchStatus{FeedID: uuid.New(), FeedCursor: model.FeedCursor{}},
			}
		}
		const okURL, badURL, notModURL = "https://ok/feed", "https://bad/feed", "https://nm/feed"
		statusRepo := &fakeFetchStatusRepo{
			dueFeeds: []*model.DueFeed{due(okURL), due(badURL), due(notModURL)},
		}
		articleRepo := &fakeArticleRepo{}
		cursor := &model.FeedCursor{}
		fetcher := &fakeFetcher{withCursorFunc: func(
			url string, _ *model.FeedCursor,
		) (*model.Feed, []*model.Article, *model.FeedCursor, error) {
			switch url {
			case badURL:
				return nil, nil, nil, apperror.NewExternalUnavailable("gw", "down", nil)
			case notModURL:
				return nil, nil, cursor, nil
			default:
				return &model.Feed{ID: uuid.New()}, []*model.Article{{ID: uuid.New()}}, cursor, nil
			}
		}}
		interactor := NewFeedJobInteractor(fetcher, articleRepo, statusRepo, fakeTxManager{}, quietLogger())

		if err := interactor.RefreshDueFeeds(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Every feed (success, failure, 304) writes exactly one status row.
		if len(statusRepo.savedStatuses) != 3 {
			t.Errorf("SaveFetchStatus calls = %d, want 3", len(statusRepo.savedStatuses))
		}
		// Only the successful feed contributes an article.
		if len(articleRepo.savedArticles) != 1 {
			t.Errorf("SaveArticle calls = %d, want 1", len(articleRepo.savedArticles))
		}
	})
}

package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

// fakeFeedRepo is a hand-written test double for repository.FeedRepository.
type fakeFeedRepo struct {
	checkExistsResult bool
	checkExistsErr    error
	checkCalls        int
	saveCalls         int
	saveErr           error
}

func (f *fakeFeedRepo) CheckFeedExistsByURL(_ context.Context, _ string) (bool, error) {
	f.checkCalls++
	return f.checkExistsResult, f.checkExistsErr
}

func (f *fakeFeedRepo) SaveFeed(_ context.Context, _ *model.Feed) error {
	f.saveCalls++
	return f.saveErr
}

func (f *fakeFeedRepo) GetFeedByID(_ context.Context, _ uuid.UUID) (*model.Feed, error) {
	return nil, nil
}

func (f *fakeFeedRepo) ListFeeds(_ context.Context, _ *model.PageCursor, _ int) ([]*model.Feed, error) {
	return nil, nil
}

func (f *fakeFeedRepo) UpdateFeed(_ context.Context, _ *model.Feed) error { return nil }
func (f *fakeFeedRepo) DeleteFeed(_ context.Context, _ uuid.UUID) error   { return nil }

// fakeArticleRepo records SaveArticle invocations.
type fakeArticleRepo struct {
	saveCalls int
}

func (f *fakeArticleRepo) SaveArticle(_ context.Context, _ *model.Article) error {
	f.saveCalls++
	return nil
}

func (f *fakeArticleRepo) GetArticleByID(_ context.Context, _ uuid.UUID) (*model.Article, error) {
	return nil, nil
}

func (f *fakeArticleRepo) ListArticlesByFeedID(
	_ context.Context, _ uuid.UUID, _ *model.PageCursor, _ int,
) ([]*model.Article, error) {
	return nil, nil
}

func (f *fakeArticleRepo) ListArticles(
	_ context.Context, _ *model.PageCursor, _ int,
) ([]*model.Article, error) {
	return nil, nil
}

func (f *fakeArticleRepo) UpdateArticle(_ context.Context, _ *model.Article) error { return nil }
func (f *fakeArticleRepo) DeleteArticle(_ context.Context, _ uuid.UUID) error      { return nil }

// fakeFetchStatusRepo records SaveFetchStatus invocations.
type fakeFetchStatusRepo struct {
	saveCalls int
}

func (f *fakeFetchStatusRepo) SaveFetchStatus(_ context.Context, _ *model.FetchStatus) error {
	f.saveCalls++
	return nil
}

func (f *fakeFetchStatusRepo) GetFetchStatusByFeedID(
	_ context.Context, _ uuid.UUID,
) (*model.FetchStatus, error) {
	return nil, nil
}

func (f *fakeFetchStatusRepo) GetDueFeeds(
	_ context.Context, _ time.Time, _ int,
) ([]*model.DueFeed, error) {
	return nil, nil
}

// fakeFetcher records FetchNewFeed calls so we can assert it is NOT invoked
// when the pre-check finds a duplicate URL.
type fakeFetcher struct {
	fetchCalls int
}

func (f *fakeFetcher) FetchNewFeed(
	_ context.Context, _ string,
) (*model.Feed, []*model.Article, *model.FeedCursor, error) {
	f.fetchCalls++
	id := uuid.New()
	feed := &model.Feed{
		ID:           id,
		Title:        "stub feed",
		FeedURL:      "https://example.com/feed.xml",
		WebsiteURL:   "https://example.com",
		Description:  "",
		RegisteredAt: time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		Language:     "ja",
	}
	cursor := &model.FeedCursor{}
	return feed, nil, cursor, nil
}

func (f *fakeFetcher) FetchFeedWithCursor(
	_ context.Context, _ string, _ *model.FeedCursor,
) (*model.Feed, []*model.Article, *model.FeedCursor, error) {
	return nil, nil, nil, nil
}

// fakeTxManager runs fn directly without an actual transaction.
type fakeTxManager struct{}

func (fakeTxManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestRegisterFeed_SkipsFetchWhenURLAlreadyExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		checkExistsResult bool
		checkExistsErr    error
		wantFetchCalls    int
		wantSaveCalls     int
		wantErr           bool
		wantCode          apperror.Code
	}{
		{
			name:              "new URL: fetch and save proceed",
			checkExistsResult: false,
			checkExistsErr:    nil,
			wantFetchCalls:    1,
			wantSaveCalls:     1,
			wantErr:           false,
		},
		{
			name:              "duplicate URL: fetch is skipped, conflict returned",
			checkExistsResult: true,
			checkExistsErr:    nil,
			wantFetchCalls:    0,
			wantSaveCalls:     0,
			wantErr:           true,
			wantCode:          apperror.CodeConflict,
		},
		{
			name:              "check fails: fetch is skipped, error propagates",
			checkExistsResult: false,
			checkExistsErr:    errors.New("db unreachable"),
			wantFetchCalls:    0,
			wantSaveCalls:     0,
			wantErr:           true,
			wantCode:          apperror.CodeInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			feedRepo := &fakeFeedRepo{
				checkExistsResult: tc.checkExistsResult,
				checkExistsErr:    tc.checkExistsErr,
			}
			articleRepo := &fakeArticleRepo{}
			fetchStatusRepo := &fakeFetchStatusRepo{}
			fetcher := &fakeFetcher{}
			txManager := fakeTxManager{}

			interactor := NewFeedInteractor(feedRepo, articleRepo, fetchStatusRepo, fetcher, txManager)

			_, _, err := interactor.RegisterFeed(context.Background(), "https://example.com/feed.xml")

			if feedRepo.checkCalls != 1 {
				t.Errorf("CheckFeedExistsByURL called %d times, want 1", feedRepo.checkCalls)
			}
			if fetcher.fetchCalls != tc.wantFetchCalls {
				t.Errorf("FetchNewFeed called %d times, want %d", fetcher.fetchCalls, tc.wantFetchCalls)
			}
			if feedRepo.saveCalls != tc.wantSaveCalls {
				t.Errorf("SaveFeed called %d times, want %d", feedRepo.saveCalls, tc.wantSaveCalls)
			}

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var appErr *apperror.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected *apperror.AppError, got %T: %v", err, err)
				}
				if appErr.Code != tc.wantCode {
					t.Errorf("error code = %q, want %q", appErr.Code, tc.wantCode)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

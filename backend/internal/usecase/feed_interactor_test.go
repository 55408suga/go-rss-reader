package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestRegisterFeed(t *testing.T) {
	t.Parallel()

	const feedURL = "https://example.com/feed.xml"

	tests := []struct {
		name              string
		existsResult      bool
		existsErr         error
		fetchErr          error
		saveFeedErr       error
		wantErr           bool
		wantCode          apperror.Code
		wantNewCalls      int
		wantSavedFeeds    int
		wantSavedArticles int
		wantSavedStatuses int
	}{
		{
			name:              "new url registers feed, articles and status",
			wantNewCalls:      1,
			wantSavedFeeds:    1,
			wantSavedArticles: 2,
			wantSavedStatuses: 1,
		},
		{
			name:         "duplicate url returns conflict and skips fetch",
			existsResult: true,
			wantErr:      true,
			wantCode:     apperror.CodeConflict,
		},
		{
			name:      "exists-check failure propagates as internal",
			existsErr: errors.New("db unreachable"),
			wantErr:   true,
			wantCode:  apperror.CodeInternal,
		},
		{
			name:         "fetch failure propagates and skips persistence",
			fetchErr:     apperror.NewExternalUnavailable("gw", "down", nil),
			wantErr:      true,
			wantCode:     apperror.CodeExternalUnavailable,
			wantNewCalls: 1,
		},
		{
			name:           "save-feed failure aborts the transaction",
			saveFeedErr:    apperror.NewInternal("repo", "boom", nil),
			wantErr:        true,
			wantCode:       apperror.CodeInternal,
			wantNewCalls:   1,
			wantSavedFeeds: 1, // SaveFeed is attempted (records the call) then errors
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fetchedFeed := &model.Feed{ID: uuid.New(), FeedURL: feedURL}
			articles := []*model.Article{
				{ID: uuid.New(), ExternalID: "a1"},
				{ID: uuid.New(), ExternalID: "a2"},
			}

			feedRepo := &fakeFeedRepo{
				existsResult: tc.existsResult,
				existsErr:    tc.existsErr,
				saveErr:      tc.saveFeedErr,
			}
			articleRepo := &fakeArticleRepo{}
			statusRepo := &fakeFetchStatusRepo{}
			fetcher := &fakeFetcher{
				newFeed:     fetchedFeed,
				newArticles: articles,
				newCursor:   &model.FeedCursor{},
				newErr:      tc.fetchErr,
			}

			interactor := NewFeedInteractor(feedRepo, articleRepo, statusRepo, fetcher, fakeTxManager{})

			feed, gotArticles, err := interactor.RegisterFeed(context.Background(), feedURL)

			if feedRepo.checkCalls != 1 {
				t.Errorf("CheckFeedExistsByURL calls = %d, want 1", feedRepo.checkCalls)
			}
			if fetcher.newCalls != tc.wantNewCalls {
				t.Errorf("FetchNewFeed calls = %d, want %d", fetcher.newCalls, tc.wantNewCalls)
			}
			if len(feedRepo.savedFeeds) != tc.wantSavedFeeds {
				t.Errorf("SaveFeed calls = %d, want %d", len(feedRepo.savedFeeds), tc.wantSavedFeeds)
			}
			if len(articleRepo.savedArticles) != tc.wantSavedArticles {
				t.Errorf("SaveArticle calls = %d, want %d", len(articleRepo.savedArticles), tc.wantSavedArticles)
			}
			if len(statusRepo.savedStatuses) != tc.wantSavedStatuses {
				t.Errorf("SaveFetchStatus calls = %d, want %d", len(statusRepo.savedStatuses), tc.wantSavedStatuses)
			}

			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if feed != fetchedFeed {
				t.Errorf("returned feed = %v, want the fetched feed", feed)
			}
			if len(gotArticles) != 2 {
				t.Errorf("returned articles = %d, want 2", len(gotArticles))
			}
		})
	}
}

func TestGetFeedByID(t *testing.T) {
	t.Parallel()

	wantFeed := &model.Feed{ID: uuid.New(), FeedURL: "https://x/feed.xml"}

	tests := []struct {
		name     string
		getFeed  *model.Feed
		getErr   error
		wantErr  bool
		wantCode apperror.Code
	}{
		{name: "found", getFeed: wantFeed},
		{
			name:     "not found is propagated with code preserved",
			getErr:   apperror.NewNotFound("repo", "missing", nil),
			wantErr:  true,
			wantCode: apperror.CodeNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			interactor := NewFeedInteractor(
				&fakeFeedRepo{getFeed: tc.getFeed, getErr: tc.getErr},
				&fakeArticleRepo{}, &fakeFetchStatusRepo{}, &fakeFetcher{}, fakeTxManager{},
			)
			got, err := interactor.GetFeedByID(context.Background(), uuid.New())
			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.getFeed {
				t.Errorf("got = %v, want %v", got, tc.getFeed)
			}
		})
	}
}

func TestListFeeds(t *testing.T) {
	t.Parallel()

	feeds := []*model.Feed{{ID: uuid.New()}, {ID: uuid.New()}}
	// The interactor over-fetches limit+1; with limit 10 a repo result of 11
	// feeds means a further page exists, so the page is trimmed to 10.
	manyFeeds := make([]*model.Feed, 11)
	for i := range manyFeeds {
		manyFeeds[i] = &model.Feed{ID: uuid.New()}
	}

	tests := []struct {
		name        string
		listFeeds   []*model.Feed
		listErr     error
		wantLen     int
		wantHasMore bool
		wantErr     bool
		wantCode    apperror.Code
	}{
		{name: "returns feeds", listFeeds: feeds, wantLen: 2, wantHasMore: false},
		{name: "over limit reports has_more", listFeeds: manyFeeds, wantLen: 10, wantHasMore: true},
		{
			name:     "error preserved",
			listErr:  apperror.NewInternal("repo", "boom", nil),
			wantErr:  true,
			wantCode: apperror.CodeInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := &fakeFeedRepo{listFeeds: tc.listFeeds, listErr: tc.listErr}
			interactor := NewFeedInteractor(
				repo,
				&fakeArticleRepo{}, &fakeFetchStatusRepo{}, &fakeFetcher{}, fakeTxManager{},
			)
			got, err := interactor.ListFeeds(context.Background(), nil, 10)
			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// The over-fetch is the contract that makes has_more detectable.
			if repo.gotListLimit != 11 {
				t.Errorf("repo limit = %d, want 11 (limit+1 over-fetch)", repo.gotListLimit)
			}
			if len(got.Items) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(got.Items), tc.wantLen)
			}
			if got.HasMore != tc.wantHasMore {
				t.Errorf("HasMore = %v, want %v", got.HasMore, tc.wantHasMore)
			}
		})
	}
}

func TestListFeedsRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	repo := &fakeFeedRepo{}
	interactor := NewFeedInteractor(
		repo, &fakeArticleRepo{}, &fakeFetchStatusRepo{}, &fakeFetcher{}, fakeTxManager{},
	)

	_, err := interactor.ListFeeds(context.Background(), nil, 0)

	assertAppErrorCode(t, err, apperror.CodeInvalidArgument)
	// The guard must fire before any repository work happens.
	if repo.gotListLimit != 0 {
		t.Errorf("repo called with limit %d, want no repo call", repo.gotListLimit)
	}
}

func TestRefreshFeed(t *testing.T) {
	t.Parallel()

	t.Run("success reconciles ids and persists atomically", func(t *testing.T) {
		t.Parallel()

		existingID := uuid.New()
		currentFeed := &model.Feed{ID: existingID, FeedURL: "https://x/feed.xml"}
		fetchedFeed := &model.Feed{ID: uuid.New(), FeedURL: "https://x/feed.xml"} // different ID
		article := &model.Article{ID: uuid.New(), FeedID: uuid.New(), ExternalID: "a1"}

		feedRepo := &fakeFeedRepo{getFeed: currentFeed}
		articleRepo := &fakeArticleRepo{}
		statusRepo := &fakeFetchStatusRepo{}
		fetcher := &fakeFetcher{
			newFeed:     fetchedFeed,
			newArticles: []*model.Article{article},
			newCursor:   &model.FeedCursor{},
		}
		interactor := NewFeedInteractor(feedRepo, articleRepo, statusRepo, fetcher, fakeTxManager{})

		if err := interactor.RefreshFeed(context.Background(), existingID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(feedRepo.updatedFeeds) != 1 {
			t.Fatalf("UpdateFeed calls = %d, want 1", len(feedRepo.updatedFeeds))
		}
		// The fetched feed's freshly generated ID must be overwritten with the existing one.
		if got := feedRepo.updatedFeeds[0].ID; got != existingID {
			t.Errorf("updated feed ID = %s, want existing %s", got, existingID)
		}
		if len(articleRepo.savedArticles) != 1 {
			t.Fatalf("SaveArticle calls = %d, want 1", len(articleRepo.savedArticles))
		}
		// Articles must be re-parented to the existing feed ID.
		if got := articleRepo.savedArticles[0].FeedID; got != existingID {
			t.Errorf("article FeedID = %s, want existing %s", got, existingID)
		}
		if len(statusRepo.savedStatuses) != 1 {
			t.Errorf("SaveFetchStatus calls = %d, want 1", len(statusRepo.savedStatuses))
		}
	})

	t.Run("propagates errors and skips later steps", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			getErr     error
			fetchErr   error
			updateErr  error
			wantCode   apperror.Code
			wantFetch  int
			wantUpdate int
		}{
			{
				name:     "get feed not found",
				getErr:   apperror.NewNotFound("repo", "missing", nil),
				wantCode: apperror.CodeNotFound,
			},
			{
				name:      "fetch unavailable",
				fetchErr:  apperror.NewExternalUnavailable("gw", "down", nil),
				wantCode:  apperror.CodeExternalUnavailable,
				wantFetch: 1,
			},
			{
				name:       "update fails",
				updateErr:  apperror.NewInternal("repo", "boom", nil),
				wantCode:   apperror.CodeInternal,
				wantFetch:  1,
				wantUpdate: 1,
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				feedRepo := &fakeFeedRepo{
					getFeed:   &model.Feed{ID: uuid.New(), FeedURL: "https://x/feed.xml"},
					getErr:    tc.getErr,
					updateErr: tc.updateErr,
				}
				fetcher := &fakeFetcher{
					newFeed:   &model.Feed{ID: uuid.New()},
					newCursor: &model.FeedCursor{},
					newErr:    tc.fetchErr,
				}
				interactor := NewFeedInteractor(
					feedRepo, &fakeArticleRepo{}, &fakeFetchStatusRepo{}, fetcher, fakeTxManager{},
				)
				err := interactor.RefreshFeed(context.Background(), uuid.New())
				assertAppErrorCode(t, err, tc.wantCode)
				if fetcher.newCalls != tc.wantFetch {
					t.Errorf("FetchNewFeed calls = %d, want %d", fetcher.newCalls, tc.wantFetch)
				}
				if len(feedRepo.updatedFeeds) != tc.wantUpdate {
					t.Errorf("UpdateFeed calls = %d, want %d", len(feedRepo.updatedFeeds), tc.wantUpdate)
				}
			})
		}
	})
}

func TestDeleteFeed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		deleteErr error
		wantErr   bool
		wantCode  apperror.Code
	}{
		{name: "success"},
		{
			name:      "error preserved",
			deleteErr: apperror.NewNotFound("repo", "missing", nil),
			wantErr:   true,
			wantCode:  apperror.CodeNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			feedRepo := &fakeFeedRepo{deleteErr: tc.deleteErr}
			interactor := NewFeedInteractor(
				feedRepo, &fakeArticleRepo{}, &fakeFetchStatusRepo{}, &fakeFetcher{}, fakeTxManager{},
			)
			id := uuid.New()
			err := interactor.DeleteFeed(context.Background(), id)
			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(feedRepo.deletedIDs) != 1 || feedRepo.deletedIDs[0] != id {
				t.Errorf("deletedIDs = %v, want [%s]", feedRepo.deletedIDs, id)
			}
		})
	}
}

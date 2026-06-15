package usecase

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestDiscoverAndRegisterFeed(t *testing.T) {
	t.Parallel()

	const websiteURL = "https://example.com/blog"

	candidates := []model.FeedCandidate{
		{FeedURL: "https://example.com/feed.xml", Title: "Example RSS", MIMEType: "application/rss+xml"},
		{FeedURL: "https://example.com/atom.xml", Title: "Example Atom", MIMEType: "application/atom+xml"},
	}

	tests := []struct {
		name          string
		byWebsiteFeed *model.Feed
		byWebsiteErr  error
		discovered    []model.FeedCandidate
		discoverErr   error
		existsResult  bool // RegisterFeed's feed_url duplicate check

		wantCode          apperror.Code
		wantDiscoverCalls int
		wantSavedFeeds    int
		wantCandidates    int
	}{
		{
			name:          "already subscribed website conflicts without external fetch",
			byWebsiteFeed: &model.Feed{ID: uuid.New(), WebsiteURL: websiteURL},
			wantCode:      apperror.CodeConflict,
		},
		{
			name:              "db miss discovers and registers the first candidate",
			byWebsiteErr:      apperror.NewNotFound("repo", "no rows", nil),
			discovered:        candidates,
			wantDiscoverCalls: 1,
			wantSavedFeeds:    1,
			wantCandidates:    2,
		},
		{
			name:              "discovery not_found propagates",
			byWebsiteErr:      apperror.NewNotFound("repo", "no rows", nil),
			discoverErr:       apperror.NewNotFound("gw", "no feed link", nil),
			wantCode:          apperror.CodeNotFound,
			wantDiscoverCalls: 1,
		},
		{
			name:              "zero candidates without error is classified not_found",
			byWebsiteErr:      apperror.NewNotFound("repo", "no rows", nil),
			discovered:        nil,
			wantCode:          apperror.CodeNotFound,
			wantDiscoverCalls: 1,
		},
		{
			name:              "already registered feed_url conflicts via RegisterFeed",
			byWebsiteErr:      apperror.NewNotFound("repo", "no rows", nil),
			discovered:        candidates,
			existsResult:      true,
			wantCode:          apperror.CodeConflict,
			wantDiscoverCalls: 1,
		},
		{
			name:         "website lookup failure other than not_found propagates",
			byWebsiteErr: apperror.NewInternal("repo", "db down", nil),
			wantCode:     apperror.CodeInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			feedRepo := &fakeFeedRepo{
				byWebsiteFeed: tc.byWebsiteFeed,
				byWebsiteErr:  tc.byWebsiteErr,
				existsResult:  tc.existsResult,
			}
			discoverer := &fakeDiscoverer{candidates: tc.discovered, err: tc.discoverErr}
			fetcher := &fakeFetcher{
				newFeed:     &model.Feed{ID: uuid.New(), FeedURL: candidates[0].FeedURL},
				newArticles: []*model.Article{{ID: uuid.New(), ExternalID: "a1"}},
				newCursor:   &model.FeedCursor{},
			}
			interactor := NewFeedInteractor(
				feedRepo, &fakeArticleRepo{}, &fakeFetchStatusRepo{},
				fetcher, discoverer, fakeTxManager{},
			)

			feed, articles, gotCandidates, err := interactor.DiscoverAndRegisterFeed(
				context.Background(), websiteURL,
			)

			if tc.wantCode != "" {
				assertAppErrorCode(t, err, tc.wantCode)
			} else if err != nil {
				t.Fatalf("DiscoverAndRegisterFeed: %v", err)
			}
			if discoverer.calls != tc.wantDiscoverCalls {
				t.Errorf("DiscoverFeedURLs calls = %d, want %d", discoverer.calls, tc.wantDiscoverCalls)
			}
			if len(feedRepo.savedFeeds) != tc.wantSavedFeeds {
				t.Errorf("SaveFeed calls = %d, want %d", len(feedRepo.savedFeeds), tc.wantSavedFeeds)
			}
			if len(gotCandidates) != tc.wantCandidates {
				t.Errorf("returned candidates = %d, want %d", len(gotCandidates), tc.wantCandidates)
			}

			if tc.wantCode != "" {
				return
			}
			// Success path: the first (highest-priority) candidate is the one
			// registered, and feed/articles flow through the RegisterFeed path.
			if fetcher.gotNewURL != candidates[0].FeedURL {
				t.Errorf("fetched url = %q, want first candidate %q", fetcher.gotNewURL, candidates[0].FeedURL)
			}
			if feed == nil {
				t.Error("feed = nil, want registered feed")
			}
			if len(articles) != 1 {
				t.Errorf("articles = %d, want 1", len(articles))
			}
		})
	}
}

// The DB fast path must look up the input URL and its trailing-slash twin in
// one query, and must not touch the network when the lookup hits.
func TestDiscoverAndRegisterFeedWebsiteURLVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "input without trailing slash also tries the slashed form",
			input: "https://example.com/blog",
			want:  []string{"https://example.com/blog", "https://example.com/blog/"},
		},
		{
			name:  "input with trailing slash also tries the bare form",
			input: "https://example.com/blog/",
			want:  []string{"https://example.com/blog/", "https://example.com/blog"},
		},
		{
			// Appending "/" after a query string would build a URL no feed's
			// channel link ever reports; complex URLs get no twin.
			name:  "input with query string is looked up as-is only",
			input: "https://example.com/blog?page=1",
			want:  []string{"https://example.com/blog?page=1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			feedRepo := &fakeFeedRepo{byWebsiteFeed: &model.Feed{ID: uuid.New()}}
			discoverer := &fakeDiscoverer{}
			interactor := NewFeedInteractor(
				feedRepo, &fakeArticleRepo{}, &fakeFetchStatusRepo{},
				&fakeFetcher{}, discoverer, fakeTxManager{},
			)

			_, _, _, err := interactor.DiscoverAndRegisterFeed(context.Background(), tc.input)

			assertAppErrorCode(t, err, apperror.CodeConflict)
			if diff := cmp.Diff(tc.want, feedRepo.gotWebsiteURLs); diff != "" {
				t.Errorf("website url variants mismatch (-want +got):\n%s", diff)
			}
			if discoverer.calls != 0 {
				t.Errorf("DiscoverFeedURLs calls = %d, want 0 (fast path must skip the network)", discoverer.calls)
			}
		})
	}
}

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"rss_reader/internal/apiresp"
	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestFeedHandlerDiscoverAndRegisterFeed(t *testing.T) {
	t.Parallel()

	const websiteURL = "https://example.com/blog"
	feed := &model.Feed{ID: uuid.New(), WebsiteURL: websiteURL}
	candidates := []model.FeedCandidate{
		{FeedURL: "https://example.com/feed.xml", Title: "Example RSS", MIMEType: "application/rss+xml"},
		{FeedURL: "https://example.com/atom.xml", MIMEType: "application/atom+xml"},
	}

	tests := []struct {
		name        string
		body        string
		usecase     *fakeFeedUsecase
		wantStatus  int
		wantErr     bool
		wantCode    apperror.Code
		wantDetails string // expected error.details[].field, "" = no assertion
	}{
		{
			name: "valid request returns 201 with candidates",
			body: `{"website_url":"https://example.com/blog"}`,
			usecase: &fakeFeedUsecase{
				discoverFeed:       feed,
				discoverArticles:   []*model.Article{{ID: uuid.New()}},
				discoverCandidates: candidates,
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:     "malformed json is invalid argument",
			body:     `{`,
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:        "missing website_url fails validation",
			body:        `{}`,
			usecase:     &fakeFeedUsecase{},
			wantErr:     true,
			wantCode:    apperror.CodeInvalidArgument,
			wantDetails: "website_url",
		},
		{
			name:        "non-url value fails validation",
			body:        `{"website_url":"not a url"}`,
			usecase:     &fakeFeedUsecase{},
			wantErr:     true,
			wantCode:    apperror.CodeInvalidArgument,
			wantDetails: "website_url",
		},
		{
			name:        "non-http scheme is rejected",
			body:        `{"website_url":"ftp://example.com/"}`,
			usecase:     &fakeFeedUsecase{},
			wantErr:     true,
			wantCode:    apperror.CodeInvalidArgument,
			wantDetails: "website_url",
		},
		{
			name:     "usecase conflict is propagated",
			body:     `{"website_url":"https://example.com/blog"}`,
			usecase:  &fakeFeedUsecase{discoverErr: apperror.NewConflict("uc", "dup", nil)},
			wantErr:  true,
			wantCode: apperror.CodeConflict,
		},
		{
			name:     "usecase not_found is propagated",
			body:     `{"website_url":"https://example.com/blog"}`,
			usecase:  &fakeFeedUsecase{discoverErr: apperror.NewNotFound("uc", "no feed", nil)},
			wantErr:  true,
			wantCode: apperror.CodeNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, rec := newEchoContext(t, http.MethodPost, "/api/v1/feeds/discover", tc.body)
			h := NewFeedHandler(tc.usecase, quietLogger())

			err := h.DiscoverAndRegisterFeed(c)

			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				if tc.wantDetails != "" {
					assertDetailsField(t, err, tc.wantDetails)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			var env apiresp.Envelope[DiscoverFeedResponse]
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if env.Data.Feed == nil || env.Data.Feed.ID != feed.ID {
				t.Errorf("response feed = %v, want id %s", env.Data.Feed, feed.ID)
			}
			if len(env.Data.Articles) != 1 {
				t.Errorf("response articles = %d, want 1", len(env.Data.Articles))
			}
			if len(env.Data.Candidates) != 2 {
				t.Errorf("response candidates = %d, want 2", len(env.Data.Candidates))
			}
			if env.Data.Candidates[0].FeedURL != candidates[0].FeedURL {
				t.Errorf("first candidate = %q, want %q", env.Data.Candidates[0].FeedURL, candidates[0].FeedURL)
			}
			if tc.usecase.gotURL != websiteURL {
				t.Errorf("usecase received url = %q, want %q", tc.usecase.gotURL, websiteURL)
			}
		})
	}
}

// Article-less feeds must serialize data.articles as [] rather than null
// (same contract as POST /feeds; raised in PR #5 review).
func TestFeedHandlerDiscoverAndRegisterFeedNormalizesNilArticles(t *testing.T) {
	t.Parallel()

	c, rec := newEchoContext(t, http.MethodPost, "/api/v1/feeds/discover",
		`{"website_url":"https://example.com/blog"}`)
	h := NewFeedHandler(&fakeFeedUsecase{
		discoverFeed:       &model.Feed{ID: uuid.New()},
		discoverArticles:   nil, // usecase may legitimately return no articles
		discoverCandidates: []model.FeedCandidate{{FeedURL: "https://example.com/feed.xml"}},
	}, quietLogger())

	if err := h.DiscoverAndRegisterFeed(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(rec.Body.String(), `"articles":[]`) {
		t.Errorf("articles must serialize as [], got body: %s", rec.Body.String())
	}
}

// assertDetailsField fails unless err is an AppError carrying a field
// violation for want.
func assertDetailsField(t *testing.T, err error, want string) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	for _, v := range appErr.Details {
		if v.Field == want {
			return
		}
	}
	t.Errorf("details = %+v, want a violation for field %q", appErr.Details, want)
}

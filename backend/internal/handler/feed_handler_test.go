package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"rss_reader/internal/apiresp"
	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestFeedHandlerRegisterFeed(t *testing.T) {
	t.Parallel()

	const url = "https://example.com/feed.xml"
	feed := &model.Feed{ID: uuid.New(), FeedURL: url}
	articles := []*model.Article{{ID: uuid.New()}}

	tests := []struct {
		name       string
		body       string
		usecase    *fakeFeedUsecase
		wantStatus int
		wantErr    bool
		wantCode   apperror.Code
	}{
		{
			name:       "valid request returns 201",
			body:       `{"feed_url":"https://example.com/feed.xml"}`,
			usecase:    &fakeFeedUsecase{registerFeed: feed, registerArticles: articles},
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
			name:     "missing url fails validation",
			body:     `{}`,
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "non-url value fails validation",
			body:     `{"feed_url":"not a url"}`,
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			// The url validator tag accepts any scheme; the http/https
			// allowlist must hold for the direct registration path too.
			name:     "non-http scheme fails validation",
			body:     `{"feed_url":"ftp://example.com/feed.xml"}`,
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "usecase conflict is propagated",
			body:     `{"feed_url":"https://example.com/feed.xml"}`,
			usecase:  &fakeFeedUsecase{registerErr: apperror.NewConflict("uc", "dup", nil)},
			wantErr:  true,
			wantCode: apperror.CodeConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, rec := newEchoContext(t, http.MethodPost, "/api/v1/feeds", tc.body)
			h := NewFeedHandler(tc.usecase, quietLogger())

			err := h.RegisterFeed(c)

			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			var env apiresp.Envelope[RegisterFeedResponse]
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if env.Data.Feed == nil || env.Data.Feed.ID != feed.ID {
				t.Errorf("response feed = %v, want id %s", env.Data.Feed, feed.ID)
			}
			if len(env.Data.Articles) != 1 {
				t.Errorf("response articles = %d, want 1", len(env.Data.Articles))
			}
			if tc.usecase.gotURL != url {
				t.Errorf("usecase received url = %q, want %q", tc.usecase.gotURL, url)
			}
		})
	}
}

func TestFeedHandlerListFeeds(t *testing.T) {
	t.Parallel()

	cursorID := uuid.New()
	cursorAt := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	validToken := *encodeCursor(&model.PageCursor{At: cursorAt, ID: cursorID})

	tests := []struct {
		name       string
		query      string
		usecase    *fakeFeedUsecase
		wantStatus int
		wantErr    bool
		wantCode   apperror.Code
		wantLimit  int
		wantCursor bool
	}{
		{
			name:       "no limit defaults to 10",
			usecase:    &fakeFeedUsecase{listResult: []*model.Feed{}},
			wantStatus: http.StatusOK,
			wantLimit:  10,
		},
		{
			name:       "explicit limit is forwarded",
			query:      "?limit=5",
			usecase:    &fakeFeedUsecase{listResult: []*model.Feed{}},
			wantStatus: http.StatusOK,
			wantLimit:  5,
		},
		{
			name:       "cursor token builds a cursor",
			query:      "?limit=5&cursor=" + validToken,
			usecase:    &fakeFeedUsecase{listResult: []*model.Feed{}},
			wantStatus: http.StatusOK,
			wantLimit:  5,
			wantCursor: true,
		},
		{
			name:     "malformed cursor is invalid argument",
			query:    "?cursor=@@@",
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "limit over max fails validation",
			query:    "?limit=101",
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "non-numeric limit is a bind error",
			query:    "?limit=abc",
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "usecase error is propagated",
			query:    "?limit=5",
			usecase:  &fakeFeedUsecase{listErr: apperror.NewInternal("uc", "boom", nil)},
			wantErr:  true,
			wantCode: apperror.CodeInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, rec := newEchoContext(t, http.MethodGet, "/api/v1/feeds"+tc.query, "")
			h := NewFeedHandler(tc.usecase, quietLogger())

			err := h.ListFeeds(c)

			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.usecase.gotLimit != tc.wantLimit {
				t.Errorf("forwarded limit = %d, want %d", tc.usecase.gotLimit, tc.wantLimit)
			}
			if (tc.usecase.gotCursor != nil) != tc.wantCursor {
				t.Errorf("cursor present = %v, want %v", tc.usecase.gotCursor != nil, tc.wantCursor)
			}
			if tc.wantCursor && tc.usecase.gotCursor != nil {
				// The full keyset must survive the round trip; losing At
				// would silently restart pagination from the epoch.
				if !tc.usecase.gotCursor.At.Equal(cursorAt) || tc.usecase.gotCursor.ID != cursorID {
					t.Errorf("forwarded cursor = %+v, want {%v %s}",
						tc.usecase.gotCursor, cursorAt, cursorID)
				}
			}
		})
	}
}

func TestFeedHandlerListFeedsEnvelope(t *testing.T) {
	t.Parallel()

	feeds := []*model.Feed{{ID: uuid.New()}, {ID: uuid.New()}}
	next := &model.PageCursor{At: time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC), ID: uuid.New()}
	uc := &fakeFeedUsecase{listResult: feeds, listNext: next, listHasMore: true}

	c, rec := newEchoContext(t, http.MethodGet, "/api/v1/feeds?limit=2", "")
	h := NewFeedHandler(uc, quietLogger())

	if err := h.ListFeeds(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env apiresp.Envelope[feedListData]
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if len(env.Data.Feeds) != 2 {
		t.Errorf("data.feeds = %d, want 2", len(env.Data.Feeds))
	}
	if env.Meta.Pagination == nil {
		t.Fatal("meta.pagination is nil, want present")
	}
	if !env.Meta.Pagination.HasMore {
		t.Error("meta.pagination.has_more = false, want true")
	}
	if env.Meta.Pagination.NextCursor == nil || *env.Meta.Pagination.NextCursor == "" {
		t.Fatal("meta.pagination.next_cursor is empty, want an opaque token")
	}
	// The opaque token must round-trip back to the cursor the usecase returned.
	got, err := decodeCursor(*env.Meta.Pagination.NextCursor)
	if err != nil {
		t.Fatalf("decode next_cursor: %v", err)
	}
	if !got.At.Equal(next.At) || got.ID != next.ID {
		t.Errorf("next_cursor = %+v, want %+v", got, next)
	}
}

func TestFeedHandlerListFeedsEmptyIsArrayNotNull(t *testing.T) {
	t.Parallel()

	c, rec := newEchoContext(t, http.MethodGet, "/api/v1/feeds", "")
	h := NewFeedHandler(&fakeFeedUsecase{listResult: nil}, quietLogger())

	if err := h.ListFeeds(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"feeds":[]`) {
		t.Errorf("body = %s, want data.feeds to be []", body)
	}
}

func TestFeedHandlerGetFeedByID(t *testing.T) {
	t.Parallel()

	feed := &model.Feed{ID: uuid.New()}

	tests := []struct {
		name       string
		paramID    string
		usecase    *fakeFeedUsecase
		wantStatus int
		wantErr    bool
		wantCode   apperror.Code
	}{
		{
			name:       "found returns 200",
			paramID:    feed.ID.String(),
			usecase:    &fakeFeedUsecase{getFeed: feed},
			wantStatus: http.StatusOK,
		},
		{
			name:     "invalid uuid is invalid argument",
			paramID:  "not-a-uuid",
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "not found is propagated",
			paramID:  feed.ID.String(),
			usecase:  &fakeFeedUsecase{getErr: apperror.NewNotFound("uc", "missing", nil)},
			wantErr:  true,
			wantCode: apperror.CodeNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, rec := newEchoContext(t, http.MethodGet, "/api/v1/feeds/"+tc.paramID, "")
			setPathParam(c, "id", tc.paramID)
			h := NewFeedHandler(tc.usecase, quietLogger())

			err := h.GetFeedByID(c)

			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

func TestFeedHandlerRefreshFeed(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	tests := []struct {
		name       string
		paramID    string
		usecase    *fakeFeedUsecase
		wantStatus int
		wantErr    bool
		wantCode   apperror.Code
	}{
		{
			name:       "success returns 204",
			paramID:    id.String(),
			usecase:    &fakeFeedUsecase{},
			wantStatus: http.StatusNoContent,
		},
		{
			name:     "invalid uuid is invalid argument",
			paramID:  "nope",
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "usecase external error is propagated",
			paramID:  id.String(),
			usecase:  &fakeFeedUsecase{refreshErr: apperror.NewExternalUnavailable("uc", "down", nil)},
			wantErr:  true,
			wantCode: apperror.CodeExternalUnavailable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, rec := newEchoContext(t, http.MethodPost, "/api/v1/feeds/"+tc.paramID+"/refresh", "")
			setPathParam(c, "id", tc.paramID)
			h := NewFeedHandler(tc.usecase, quietLogger())

			err := h.RefreshFeed(c)

			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if rec.Body.Len() != 0 {
				t.Errorf("body = %q, want empty", rec.Body.String())
			}
		})
	}
}

func TestFeedHandlerDeleteFeed(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	tests := []struct {
		name       string
		paramID    string
		usecase    *fakeFeedUsecase
		wantStatus int
		wantErr    bool
		wantCode   apperror.Code
	}{
		{
			name:       "success returns 204",
			paramID:    id.String(),
			usecase:    &fakeFeedUsecase{},
			wantStatus: http.StatusNoContent,
		},
		{
			name:     "invalid uuid is invalid argument",
			paramID:  "nope",
			usecase:  &fakeFeedUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "usecase not found is propagated",
			paramID:  id.String(),
			usecase:  &fakeFeedUsecase{deleteErr: apperror.NewNotFound("uc", "missing", nil)},
			wantErr:  true,
			wantCode: apperror.CodeNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, rec := newEchoContext(t, http.MethodDelete, "/api/v1/feeds/"+tc.paramID, "")
			setPathParam(c, "id", tc.paramID)
			h := NewFeedHandler(tc.usecase, quietLogger())

			err := h.DeleteFeed(c)

			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

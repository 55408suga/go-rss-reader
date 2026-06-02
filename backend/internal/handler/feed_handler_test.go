package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

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
			var resp RegisterFeedResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if resp.Feed == nil || resp.Feed.ID != feed.ID {
				t.Errorf("response feed = %v, want id %s", resp.Feed, feed.ID)
			}
			if len(resp.Articles) != 1 {
				t.Errorf("response articles = %d, want 1", len(resp.Articles))
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
			name:       "both cursor params build a cursor",
			query:      "?limit=5&cursor_at=2026-01-01T00:00:00Z&cursor_id=" + cursorID.String(),
			usecase:    &fakeFeedUsecase{listResult: []*model.Feed{}},
			wantStatus: http.StatusOK,
			wantLimit:  5,
			wantCursor: true,
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
		})
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

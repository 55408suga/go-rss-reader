package handler

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"rss_reader/internal/apiresp"
	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestArticleHandlerListArticlesByFeedID(t *testing.T) {
	t.Parallel()

	feedID := uuid.New()
	articles := []*model.Article{{ID: uuid.New()}}
	cursorID := uuid.New()
	cursorAt := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	validToken := *encodeCursor(&model.PageCursor{At: cursorAt, ID: cursorID})

	tests := []struct {
		name       string
		paramID    string
		query      string
		usecase    *fakeArticleUsecase
		wantStatus int
		wantErr    bool
		wantCode   apperror.Code
		wantLimit  int
		wantCursor bool
	}{
		{
			name:       "success defaults limit to 10",
			paramID:    feedID.String(),
			usecase:    &fakeArticleUsecase{listByFeed: articles},
			wantStatus: http.StatusOK,
			wantLimit:  10,
		},
		{
			name:       "cursor token builds a cursor",
			paramID:    feedID.String(),
			query:      "?cursor=" + validToken,
			usecase:    &fakeArticleUsecase{listByFeed: articles},
			wantStatus: http.StatusOK,
			wantLimit:  10,
			wantCursor: true,
		},
		{
			name:     "malformed cursor is invalid argument",
			paramID:  feedID.String(),
			query:    "?cursor=@@@",
			usecase:  &fakeArticleUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "invalid feed id is invalid argument",
			paramID:  "not-a-uuid",
			usecase:  &fakeArticleUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "limit over max fails validation",
			paramID:  feedID.String(),
			query:    "?limit=101",
			usecase:  &fakeArticleUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "usecase error is propagated",
			paramID:  feedID.String(),
			usecase:  &fakeArticleUsecase{listByFeedErr: apperror.NewInternal("uc", "boom", nil)},
			wantErr:  true,
			wantCode: apperror.CodeInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			target := "/api/v1/feeds/" + tc.paramID + "/articles" + tc.query
			c, rec := newEchoContext(t, http.MethodGet, target, "")
			setPathParam(c, "feed_id", tc.paramID)
			h := NewArticleHandler(tc.usecase, quietLogger())

			err := h.ListArticlesByFeedID(c)

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
			if tc.usecase.gotFeedID != feedID {
				t.Errorf("forwarded feedID = %s, want %s", tc.usecase.gotFeedID, feedID)
			}
			if tc.usecase.gotLimit != tc.wantLimit {
				t.Errorf("forwarded limit = %d, want %d", tc.usecase.gotLimit, tc.wantLimit)
			}
			if (tc.usecase.gotCursor != nil) != tc.wantCursor {
				t.Errorf("cursor present = %v, want %v", tc.usecase.gotCursor != nil, tc.wantCursor)
			}
			if tc.wantCursor && tc.usecase.gotCursor != nil {
				if !tc.usecase.gotCursor.At.Equal(cursorAt) || tc.usecase.gotCursor.ID != cursorID {
					t.Errorf("forwarded cursor = %+v, want {%v %s}",
						tc.usecase.gotCursor, cursorAt, cursorID)
				}
			}
		})
	}
}

func TestArticleHandlerListArticlesEnvelope(t *testing.T) {
	t.Parallel()

	uc := &fakeArticleUsecase{listAll: []*model.Article{{ID: uuid.New()}}}
	c, rec := newEchoContext(t, http.MethodGet, "/api/v1/articles", "")
	h := NewArticleHandler(uc, quietLogger())

	if err := h.ListArticles(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env apiresp.Envelope[articleListData]
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if len(env.Data.Articles) != 1 {
		t.Errorf("data.articles = %d, want 1", len(env.Data.Articles))
	}
	if env.Meta.Pagination == nil {
		t.Fatal("meta.pagination is nil, want present")
	}
	// Single page: no further results, so the cursor must be null.
	if env.Meta.Pagination.HasMore {
		t.Error("has_more = true, want false")
	}
	if env.Meta.Pagination.NextCursor != nil {
		t.Errorf("next_cursor = %q, want null", *env.Meta.Pagination.NextCursor)
	}
}

func TestArticleHandlerListArticles(t *testing.T) {
	t.Parallel()

	articles := []*model.Article{{ID: uuid.New()}}
	cursorID := uuid.New()
	cursorAt := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	validToken := *encodeCursor(&model.PageCursor{At: cursorAt, ID: cursorID})

	tests := []struct {
		name       string
		query      string
		usecase    *fakeArticleUsecase
		wantStatus int
		wantErr    bool
		wantCode   apperror.Code
		wantLimit  int
		wantCursor bool
	}{
		{
			name:       "success defaults limit to 10",
			usecase:    &fakeArticleUsecase{listAll: articles},
			wantStatus: http.StatusOK,
			wantLimit:  10,
		},
		{
			name:       "explicit limit is forwarded",
			query:      "?limit=25",
			usecase:    &fakeArticleUsecase{listAll: articles},
			wantStatus: http.StatusOK,
			wantLimit:  25,
		},
		{
			name:       "cursor token builds a cursor",
			query:      "?cursor=" + validToken,
			usecase:    &fakeArticleUsecase{listAll: articles},
			wantStatus: http.StatusOK,
			wantLimit:  10,
			wantCursor: true,
		},
		{
			name:     "malformed cursor is invalid argument",
			query:    "?cursor=@@@",
			usecase:  &fakeArticleUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "limit over max fails validation",
			query:    "?limit=101",
			usecase:  &fakeArticleUsecase{},
			wantErr:  true,
			wantCode: apperror.CodeInvalidArgument,
		},
		{
			name:     "usecase error is propagated",
			usecase:  &fakeArticleUsecase{listAllErr: apperror.NewInternal("uc", "boom", nil)},
			wantErr:  true,
			wantCode: apperror.CodeInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, rec := newEchoContext(t, http.MethodGet, "/api/v1/articles"+tc.query, "")
			h := NewArticleHandler(tc.usecase, quietLogger())

			err := h.ListArticles(c)

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
				if !tc.usecase.gotCursor.At.Equal(cursorAt) || tc.usecase.gotCursor.ID != cursorID {
					t.Errorf("forwarded cursor = %+v, want {%v %s}",
						tc.usecase.gotCursor, cursorAt, cursorID)
				}
			}
		})
	}
}

// TestArticleHandlerListArticlesPagination drives the has-more branch: the
// usecase's structured next cursor must surface as a decodable opaque token.
func TestArticleHandlerListArticlesPagination(t *testing.T) {
	t.Parallel()

	next := &model.PageCursor{At: time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC), ID: uuid.New()}
	uc := &fakeArticleUsecase{
		listAll:        []*model.Article{{ID: uuid.New()}},
		listAllNext:    next,
		listAllHasMore: true,
	}
	c, rec := newEchoContext(t, http.MethodGet, "/api/v1/articles", "")
	h := NewArticleHandler(uc, quietLogger())

	if err := h.ListArticles(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env apiresp.Envelope[articleListData]
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.Meta.Pagination == nil {
		t.Fatal("meta.pagination is nil, want present")
	}
	if !env.Meta.Pagination.HasMore {
		t.Error("has_more = false, want true")
	}
	if env.Meta.Pagination.NextCursor == nil {
		t.Fatal("next_cursor is null, want an opaque token")
	}
	got, err := decodeCursor(*env.Meta.Pagination.NextCursor)
	if err != nil {
		t.Fatalf("decode next_cursor: %v", err)
	}
	if !got.At.Equal(next.At) || got.ID != next.ID {
		t.Errorf("next_cursor = %+v, want %+v", got, next)
	}
}

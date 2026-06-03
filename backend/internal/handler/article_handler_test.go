package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestArticleHandlerListArticlesByFeedID(t *testing.T) {
	t.Parallel()

	feedID := uuid.New()
	articles := []*model.Article{{ID: uuid.New()}}

	tests := []struct {
		name       string
		paramID    string
		query      string
		usecase    *fakeArticleUsecase
		wantStatus int
		wantErr    bool
		wantCode   apperror.Code
		wantLimit  int
	}{
		{
			name:       "success defaults limit to 10",
			paramID:    feedID.String(),
			usecase:    &fakeArticleUsecase{listByFeed: articles},
			wantStatus: http.StatusOK,
			wantLimit:  10,
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
		})
	}
}

func TestArticleHandlerListArticles(t *testing.T) {
	t.Parallel()

	articles := []*model.Article{{ID: uuid.New()}}

	tests := []struct {
		name       string
		query      string
		usecase    *fakeArticleUsecase
		wantStatus int
		wantErr    bool
		wantCode   apperror.Code
		wantLimit  int
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
		})
	}
}

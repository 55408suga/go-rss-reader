package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestListArticlesByFeedID(t *testing.T) {
	t.Parallel()

	articles := []*model.Article{{ID: uuid.New()}, {ID: uuid.New()}}

	tests := []struct {
		name     string
		list     []*model.Article
		listErr  error
		wantLen  int
		wantErr  bool
		wantCode apperror.Code
	}{
		{name: "returns articles", list: articles, wantLen: 2},
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
			repo := &fakeArticleRepo{listByFeed: tc.list, listByFeedErr: tc.listErr}
			interactor := NewArticleInteractor(repo)
			got, err := interactor.ListArticlesByFeedID(context.Background(), uuid.New(), nil, 10)
			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo.gotListLimit != 11 {
				t.Errorf("repo limit = %d, want 11 (limit+1 over-fetch)", repo.gotListLimit)
			}
			if len(got.Items) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(got.Items), tc.wantLen)
			}
		})
	}
}

func TestArticleListsRejectInvalidLimit(t *testing.T) {
	t.Parallel()

	repo := &fakeArticleRepo{}
	interactor := NewArticleInteractor(repo)

	_, err := interactor.ListArticlesByFeedID(context.Background(), uuid.New(), nil, -1)
	assertAppErrorCode(t, err, apperror.CodeInvalidArgument)

	_, err = interactor.ListArticles(context.Background(), nil, 101)
	assertAppErrorCode(t, err, apperror.CodeInvalidArgument)

	// The guard must fire before any repository work happens.
	if repo.gotListLimit != 0 {
		t.Errorf("repo called with limit %d, want no repo call", repo.gotListLimit)
	}
}

func TestListArticles(t *testing.T) {
	t.Parallel()

	articles := []*model.Article{{ID: uuid.New()}}

	tests := []struct {
		name     string
		list     []*model.Article
		listErr  error
		wantLen  int
		wantErr  bool
		wantCode apperror.Code
	}{
		{name: "returns articles", list: articles, wantLen: 1},
		{
			name:     "error preserved",
			listErr:  apperror.NewNotFound("repo", "missing", nil),
			wantErr:  true,
			wantCode: apperror.CodeNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := &fakeArticleRepo{listAll: tc.list, listAllErr: tc.listErr}
			interactor := NewArticleInteractor(repo)
			got, err := interactor.ListArticles(context.Background(), nil, 10)
			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo.gotListLimit != 11 {
				t.Errorf("repo limit = %d, want 11 (limit+1 over-fetch)", repo.gotListLimit)
			}
			if len(got.Items) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(got.Items), tc.wantLen)
			}
		})
	}
}

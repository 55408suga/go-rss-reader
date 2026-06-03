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
			interactor := NewArticleInteractor(
				&fakeArticleRepo{listByFeed: tc.list, listByFeedErr: tc.listErr},
			)
			got, err := interactor.ListArticlesByFeedID(context.Background(), uuid.New(), nil, 10)
			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(got), tc.wantLen)
			}
		})
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
			interactor := NewArticleInteractor(
				&fakeArticleRepo{listAll: tc.list, listAllErr: tc.listErr},
			)
			got, err := interactor.ListArticles(context.Background(), nil, 10)
			if tc.wantErr {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}

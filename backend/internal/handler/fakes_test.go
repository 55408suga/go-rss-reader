package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/usecase"
)

// Compile-time guarantees that the fakes stay in sync with the real ports.
var (
	_ usecase.FeedUsecase    = (*fakeFeedUsecase)(nil)
	_ usecase.ArticleUsecase = (*fakeArticleUsecase)(nil)
)

type fakeFeedUsecase struct {
	registerFeed     *model.Feed
	registerArticles []*model.Article
	registerErr      error

	discoverFeed       *model.Feed
	discoverArticles   []*model.Article
	discoverCandidates []model.FeedCandidate
	discoverErr        error

	getFeed *model.Feed
	getErr  error

	listResult  []*model.Feed
	listNext    *model.PageCursor
	listHasMore bool
	listErr     error

	refreshErr error
	deleteErr  error

	// captured arguments from the most recent call
	gotURL    string
	gotID     uuid.UUID
	gotCursor *model.PageCursor
	gotLimit  int
}

func (f *fakeFeedUsecase) RegisterFeed(
	_ context.Context, feedURL string,
) (*model.Feed, []*model.Article, error) {
	f.gotURL = feedURL
	return f.registerFeed, f.registerArticles, f.registerErr
}

func (f *fakeFeedUsecase) DiscoverAndRegisterFeed(
	_ context.Context, websiteURL string,
) (*model.Feed, []*model.Article, []model.FeedCandidate, error) {
	f.gotURL = websiteURL
	return f.discoverFeed, f.discoverArticles, f.discoverCandidates, f.discoverErr
}

func (f *fakeFeedUsecase) GetFeedByID(_ context.Context, feedID uuid.UUID) (*model.Feed, error) {
	f.gotID = feedID
	return f.getFeed, f.getErr
}

func (f *fakeFeedUsecase) ListFeeds(
	_ context.Context, cursor *model.PageCursor, limit int,
) (*model.Page[*model.Feed], error) {
	f.gotCursor = cursor
	f.gotLimit = limit
	if f.listErr != nil {
		return nil, f.listErr
	}
	return &model.Page[*model.Feed]{
		Items:      f.listResult,
		NextCursor: f.listNext,
		HasMore:    f.listHasMore,
	}, nil
}

func (f *fakeFeedUsecase) RefreshFeed(_ context.Context, feedID uuid.UUID) error {
	f.gotID = feedID
	return f.refreshErr
}

func (f *fakeFeedUsecase) DeleteFeed(_ context.Context, feedID uuid.UUID) error {
	f.gotID = feedID
	return f.deleteErr
}

type fakeArticleUsecase struct {
	listByFeed        []*model.Article
	listByFeedNext    *model.PageCursor
	listByFeedHasMore bool
	listByFeedErr     error
	listAll           []*model.Article
	listAllNext       *model.PageCursor
	listAllHasMore    bool
	listAllErr        error

	gotFeedID uuid.UUID
	gotCursor *model.PageCursor
	gotLimit  int
}

func (f *fakeArticleUsecase) ListArticlesByFeedID(
	_ context.Context, feedID uuid.UUID, cursor *model.PageCursor, limit int,
) (*model.Page[*model.Article], error) {
	f.gotFeedID = feedID
	f.gotCursor = cursor
	f.gotLimit = limit
	if f.listByFeedErr != nil {
		return nil, f.listByFeedErr
	}
	return &model.Page[*model.Article]{
		Items:      f.listByFeed,
		NextCursor: f.listByFeedNext,
		HasMore:    f.listByFeedHasMore,
	}, nil
}

func (f *fakeArticleUsecase) ListArticles(
	_ context.Context, cursor *model.PageCursor, limit int,
) (*model.Page[*model.Article], error) {
	f.gotCursor = cursor
	f.gotLimit = limit
	if f.listAllErr != nil {
		return nil, f.listAllErr
	}
	return &model.Page[*model.Article]{
		Items:      f.listAll,
		NextCursor: f.listAllNext,
		HasMore:    f.listAllHasMore,
	}, nil
}

// quietLogger discards log output so handler warning logs do not clutter tests.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newEchoContext builds an *echo.Context backed by httptest for direct handler calls.
// A JSON content-type is set whenever body is non-empty.
func newEchoContext(t *testing.T, method, target, body string) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// setPathParam injects a single path parameter; echo v5 reads params from PathValues.
func setPathParam(c *echo.Context, name, value string) {
	c.SetPathValues(echo.PathValues{{Name: name, Value: value}})
}

// assertAppErrorCode fails unless err is a non-nil *apperror.AppError carrying want.
func assertAppErrorCode(t *testing.T, err error, want apperror.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", want)
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T: %v", err, err)
	}
	if appErr.Code != want {
		t.Errorf("error code = %q, want %q", appErr.Code, want)
	}
}

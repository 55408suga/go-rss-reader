package router_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"rss_reader/internal/apperror"
	"rss_reader/internal/di"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/handler"
	appmw "rss_reader/internal/infra/middleware"
	"rss_reader/internal/infra/router"
	"rss_reader/internal/usecase"
)

// Compile-time guarantees that the stubs satisfy the real input ports.
var (
	_ usecase.FeedUsecase    = (*stubFeedUsecase)(nil)
	_ usecase.ArticleUsecase = (*stubArticleUsecase)(nil)
)

type stubFeedUsecase struct {
	feed     *model.Feed
	articles []*model.Article
	feeds    []*model.Feed
	err      error
}

func (s *stubFeedUsecase) RegisterFeed(
	context.Context, string,
) (*model.Feed, []*model.Article, error) {
	return s.feed, s.articles, s.err
}

func (s *stubFeedUsecase) GetFeedByID(context.Context, uuid.UUID) (*model.Feed, error) {
	return s.feed, s.err
}

func (s *stubFeedUsecase) ListFeeds(
	context.Context, *model.PageCursor, int,
) ([]*model.Feed, error) {
	return s.feeds, s.err
}

func (s *stubFeedUsecase) RefreshFeed(context.Context, uuid.UUID) error { return s.err }
func (s *stubFeedUsecase) DeleteFeed(context.Context, uuid.UUID) error  { return s.err }

type stubArticleUsecase struct {
	articles []*model.Article
	err      error
}

func (s *stubArticleUsecase) ListArticlesByFeedID(
	context.Context, uuid.UUID, *model.PageCursor, int,
) ([]*model.Article, error) {
	return s.articles, s.err
}

func (s *stubArticleUsecase) ListArticles(
	context.Context, *model.PageCursor, int,
) ([]*model.Article, error) {
	return s.articles, s.err
}

// newServer wires the production HTTP stack (error handler + request-id
// middleware + real routes) around mocked usecases, mirroring cmd/main.go.
func newServer(feedUC usecase.FeedUsecase, articleUC usecase.ArticleUsecase) *echo.Echo {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	e := echo.NewWithConfig(echo.Config{
		HTTPErrorHandler: appmw.NewGlobalErrorHandler(logger),
	})
	e.Use(echomw.RequestID())
	e.Use(appmw.RequestIDContext())
	e.Use(appmw.RequestLogger(logger))
	e.Use(echomw.Recover())

	components := &di.ApplicationComponents{
		FeedHandler:    handler.NewFeedHandler(feedUC, logger),
		ArticleHandler: handler.NewArticleHandler(articleUC, logger),
	}
	router.SetupRoutes(e, components)
	return e
}

func TestRouterStatusMatrix(t *testing.T) {
	t.Parallel()

	feedID := uuid.New()

	tests := []struct {
		name       string
		method     string
		target     string
		body       string
		feedUC     *stubFeedUsecase
		articleUC  *stubArticleUsecase
		wantStatus int
	}{
		{
			name:       "register feed returns 201",
			method:     http.MethodPost,
			target:     "/api/v1/feeds",
			body:       `{"feed_url":"https://example.com/feed.xml"}`,
			feedUC:     &stubFeedUsecase{feed: &model.Feed{ID: feedID}, articles: []*model.Article{}},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "list feeds returns 200",
			method:     http.MethodGet,
			target:     "/api/v1/feeds",
			feedUC:     &stubFeedUsecase{feeds: []*model.Feed{}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "get feed with bad uuid returns 400",
			method:     http.MethodGet,
			target:     "/api/v1/feeds/not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "refresh feed returns 204",
			method:     http.MethodPost,
			target:     "/api/v1/feeds/" + feedID.String() + "/refresh",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "delete feed returns 204",
			method:     http.MethodDelete,
			target:     "/api/v1/feeds/" + feedID.String(),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "list articles by feed returns 200",
			method:     http.MethodGet,
			target:     "/api/v1/feeds/" + feedID.String() + "/articles",
			articleUC:  &stubArticleUsecase{articles: []*model.Article{}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "list articles returns 200",
			method:     http.MethodGet,
			target:     "/api/v1/articles",
			articleUC:  &stubArticleUsecase{articles: []*model.Article{}},
			wantStatus: http.StatusOK,
		},
		{
			// Regression guard: Echo raises its built-in echo.ErrNotFound for an
			// unmatched path. The global error handler must surface that as 404,
			// not the generic 500 it returned before the echo.StatusCode fix.
			name:       "unknown route returns 404",
			method:     http.MethodGet,
			target:     "/api/v1/does-not-exist",
			wantStatus: http.StatusNotFound,
		},
		{
			// /api/v1/feeds is registered for GET and POST; PATCH makes Echo raise
			// its built-in echo.ErrMethodNotAllowed, which must surface as 405.
			name:       "wrong method on a known path returns 405",
			method:     http.MethodPatch,
			target:     "/api/v1/feeds",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			feedUC := tc.feedUC
			if feedUC == nil {
				feedUC = &stubFeedUsecase{}
			}
			articleUC := tc.articleUC
			if articleUC == nil {
				articleUC = &stubArticleUsecase{}
			}
			e := newServer(feedUC, articleUC)

			var body io.Reader = http.NoBody
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.target, body)
			if tc.body != "" {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			}
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestRouterErrorResponseShape(t *testing.T) {
	t.Parallel()

	e := newServer(
		&stubFeedUsecase{err: apperror.NewNotFound("uc", "missing", nil)},
		&stubArticleUsecase{},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/feeds/"+uuid.New().String(), http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Error struct {
			Code    apperror.Code `json:"code"`
			Message string        `json:"message"`
		} `json:"error"`
		Meta struct {
			RequestID string `json:"request_id"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != apperror.CodeNotFound {
		t.Errorf("error.code = %q, want %q", body.Error.Code, apperror.CodeNotFound)
	}
	// echo's RequestID middleware must populate meta.request_id end to end.
	if body.Meta.RequestID == "" {
		t.Error("meta.request_id is empty; the RequestID middleware did not propagate")
	}
}

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
	feed       *model.Feed
	articles   []*model.Article
	feeds      []*model.Feed
	candidates []model.FeedCandidate
	err        error
}

func (s *stubFeedUsecase) RegisterFeed(
	context.Context, string,
) (*model.Feed, []*model.Article, error) {
	return s.feed, s.articles, s.err
}

func (s *stubFeedUsecase) DiscoverAndRegisterFeed(
	context.Context, string,
) (*model.Feed, []*model.Article, []model.FeedCandidate, error) {
	return s.feed, s.articles, s.candidates, s.err
}

func (s *stubFeedUsecase) GetFeedByID(context.Context, uuid.UUID) (*model.Feed, error) {
	return s.feed, s.err
}

func (s *stubFeedUsecase) ListFeeds(
	context.Context, *model.PageCursor, int,
) (*model.Page[*model.Feed], error) {
	if s.err != nil {
		return nil, s.err
	}
	return &model.Page[*model.Feed]{Items: s.feeds}, nil
}

func (s *stubFeedUsecase) RefreshFeed(context.Context, uuid.UUID) error { return s.err }
func (s *stubFeedUsecase) DeleteFeed(context.Context, uuid.UUID) error  { return s.err }

type stubArticleUsecase struct {
	articles []*model.Article
	err      error
}

func (s *stubArticleUsecase) ListArticlesByFeedID(
	context.Context, uuid.UUID, *model.PageCursor, int,
) (*model.Page[*model.Article], error) {
	if s.err != nil {
		return nil, s.err
	}
	return &model.Page[*model.Article]{Items: s.articles}, nil
}

func (s *stubArticleUsecase) ListArticles(
	context.Context, *model.PageCursor, int,
) (*model.Page[*model.Article], error) {
	if s.err != nil {
		return nil, s.err
	}
	return &model.Page[*model.Article]{Items: s.articles}, nil
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

// TestCORSPreflightAndHeaders mirrors cmd/main.go's middleware order
// (request-id -> CORS -> routes) to verify the CORS wiring: preflight from an
// allowed origin is short-circuited with 204 + allow headers, a disallowed
// origin gets no allow-origin header, and an actual request echoes the origin.
func TestCORSPreflightAndHeaders(t *testing.T) {
	t.Parallel()

	const allowed = "http://localhost:3000"
	newCORSServer := func() *echo.Echo {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		e := echo.NewWithConfig(echo.Config{HTTPErrorHandler: appmw.NewGlobalErrorHandler(logger)})
		e.Use(echomw.RequestID())
		e.Use(appmw.RequestIDContext())
		e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
			AllowOrigins:     []string{allowed},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
			AllowCredentials: false,
		}))
		components := &di.ApplicationComponents{
			FeedHandler:    handler.NewFeedHandler(&stubFeedUsecase{feeds: []*model.Feed{}}, logger),
			ArticleHandler: handler.NewArticleHandler(&stubArticleUsecase{}, logger),
		}
		router.SetupRoutes(e, components)
		return e
	}

	t.Run("preflight from allowed origin returns 204 with allow headers", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/feeds", http.NoBody)
		req.Header.Set(echo.HeaderOrigin, allowed)
		req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
		rec := httptest.NewRecorder()
		newCORSServer().ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("preflight status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		if got := rec.Header().Get(echo.HeaderAccessControlAllowOrigin); got != allowed {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, allowed)
		}
		if got := rec.Header().Get(echo.HeaderAccessControlAllowCredentials); got != "" {
			t.Errorf("Access-Control-Allow-Credentials = %q, want empty", got)
		}
	})

	t.Run("preflight from disallowed origin has no allow-origin", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/feeds", http.NoBody)
		req.Header.Set(echo.HeaderOrigin, "http://evil.example.com")
		req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
		rec := httptest.NewRecorder()
		newCORSServer().ServeHTTP(rec, req)

		if got := rec.Header().Get(echo.HeaderAccessControlAllowOrigin); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want empty for a disallowed origin", got)
		}
	})

	t.Run("actual request from allowed origin echoes the origin", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/feeds", http.NoBody)
		req.Header.Set(echo.HeaderOrigin, allowed)
		rec := httptest.NewRecorder()
		newCORSServer().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get(echo.HeaderAccessControlAllowOrigin); got != allowed {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, allowed)
		}
	})
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

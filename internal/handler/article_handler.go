// Package handler provides HTTP handlers for API endpoints.
package handler

import (
	"log/slog"
	"net/http"
	applogger "rss_reader/internal/applog"
	"rss_reader/internal/apperror"
	"rss_reader/internal/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// ArticleHandler handles article-related HTTP requests.
type ArticleHandler struct {
	articleUsecase usecase.ArticleUsecase
	logger         *slog.Logger
}

// NewArticleHandler creates an ArticleHandler.
func NewArticleHandler(articleUsecase usecase.ArticleUsecase, logger *slog.Logger) *ArticleHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &ArticleHandler{articleUsecase: articleUsecase, logger: logger}
}

// ListArticlesByFeedID returns articles for a feed.
// Accepts optional query params cursor_at (RFC3339) and cursor_id (UUID) for pagination.
func (ah *ArticleHandler) ListArticlesByFeedID(c *echo.Context) error {
	const op = "ArticleHandler.ListArticlesByFeedID"

	feedID, err := uuid.Parse(c.Param("feed_id"))
	if err != nil {
		applogger.WithContext(c.Request().Context(), ah.logger).WarnContext(c.Request().Context(),
			"invalid feed id",
			"error", err,
		)
		return apperror.NewInvalidArgument(op, "invalid feed id", err)
	}

	cursor := parseCursorFromQuery(c)
	articles, err := ah.articleUsecase.ListArticlesByFeedID(c.Request().Context(), feedID, cursor, 50)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, articles)
}

// ListArticles returns articles.
// Accepts optional query params cursor_at (RFC3339) and cursor_id (UUID) for pagination.
func (ah *ArticleHandler) ListArticles(c *echo.Context) error {
	const op = "ArticleHandler.ListArticles"

	cursor := parseCursorFromQuery(c)
	articles, err := ah.articleUsecase.ListArticles(c.Request().Context(), cursor, 50)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, articles)
}


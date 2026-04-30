// Package handler provides HTTP handlers for API endpoints.
package handler

import (
	"log/slog"
	"net/http"
	"rss_reader/internal/apperror"
	applogger "rss_reader/internal/applog"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/usecase"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// ArticleHandler handles article-related HTTP requests.
type ArticleHandler struct {
	articleUsecase usecase.ArticleUsecase
	logger         *slog.Logger
}

// ListArticlesRequest is the query params for listing articles with pagination.
type ListArticlesRequest struct {
	CursorAt *time.Time `query:"cursor_at"` // RFC3339 timestamp
	CursorID *uuid.UUID `query:"cursor_id"` // UUID
	Limit    int        `query:"limit" validate:"gte=1,lte=100"`
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
func (h *ArticleHandler) ListArticlesByFeedID(c *echo.Context) error {
	const op = "ArticleHandler.ListArticlesByFeedID"
	ctx := c.Request().Context()
	logger := applogger.WithContext(ctx, h.logger)

	feedID, err := uuid.Parse(c.Param("feed_id"))
	if err != nil {
		logger.WarnContext(ctx, "invalid feed id", "error", err)
		return apperror.NewInvalidArgument(op, "invalid feed id", err)
	}

	var req ListArticlesRequest
	if err := c.Bind(&req); err != nil {
		logger.WarnContext(ctx, "invalid query params", "error", err)
		return apperror.NewInvalidArgument(op, "invalid query params", err)
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	if err := requestValidator.Struct(req); err != nil {
		logger.WarnContext(ctx, "request validation failed", "error", err)
		return apperror.NewInvalidArgument(op, "validation failed", err)
	}

	var cursor *model.PageCursor
	if req.CursorAt != nil && req.CursorID != nil {
		cursor = &model.PageCursor{At: *req.CursorAt, ID: *req.CursorID}
	}

	articles, err := h.articleUsecase.ListArticlesByFeedID(ctx, feedID, cursor, req.Limit)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, articles)
}

// ListArticles returns articles.
// Accepts optional query params cursor_at (RFC3339) and cursor_id (UUID) for pagination.
func (h *ArticleHandler) ListArticles(c *echo.Context) error {
	const op = "ArticleHandler.ListArticles"
	ctx := c.Request().Context()
	logger := applogger.WithContext(ctx, h.logger)

	var req ListArticlesRequest
	if err := c.Bind(&req); err != nil {
		logger.WarnContext(ctx, "invalid query params", "error", err)
		return apperror.NewInvalidArgument(op, "invalid query params", err)
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	if err := requestValidator.Struct(req); err != nil {
		logger.WarnContext(ctx, "request validation failed", "error", err)
		return apperror.NewInvalidArgument(op, "validation failed", err)
	}
	var cursor *model.PageCursor
	if req.CursorAt != nil && req.CursorID != nil {
		cursor = &model.PageCursor{At: *req.CursorAt, ID: *req.CursorID}
	}

	articles, err := h.articleUsecase.ListArticles(ctx, cursor, req.Limit)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, articles)
}

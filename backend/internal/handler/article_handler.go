// Package handler provides HTTP handlers for API endpoints.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"rss_reader/internal/apperror"
	applogger "rss_reader/internal/applog"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/usecase"
)

// ArticleHandler handles article-related HTTP requests.
type ArticleHandler struct {
	articleUsecase usecase.ArticleUsecase
	logger         *slog.Logger
}

// ListArticlesRequest is the query params for listing articles with pagination.
type ListArticlesRequest struct {
	Cursor string `query:"cursor"` // opaque pagination token (empty = first page)
	Limit  int    `query:"limit" validate:"gte=1,lte=100"`
}

// articleListData is the data payload for a paginated article list.
type articleListData struct {
	Articles []*model.Article `json:"articles"`
}

// NewArticleHandler creates an ArticleHandler.
func NewArticleHandler(articleUsecase usecase.ArticleUsecase, logger *slog.Logger) *ArticleHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &ArticleHandler{articleUsecase: articleUsecase, logger: logger}
}

// ListArticlesByFeedID returns one page of articles for a feed.
// Accepts optional query params cursor (opaque pagination token) and limit.
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
		return apperror.NewInvalidArgument(op, "validation failed", err).
			WithDetails(validationDetails(err))
	}

	cursor, err := decodeCursor(req.Cursor)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	page, err := h.articleUsecase.ListArticlesByFeedID(ctx, feedID, cursor, req.Limit)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	articles := page.Items
	if articles == nil {
		articles = []*model.Article{} // emit data.articles:[] rather than null
	}
	return respondPage(c, http.StatusOK, articleListData{Articles: articles}, page)
}

// ListArticles returns one page of articles.
// Accepts optional query params cursor (opaque pagination token) and limit.
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
		return apperror.NewInvalidArgument(op, "validation failed", err).
			WithDetails(validationDetails(err))
	}
	cursor, err := decodeCursor(req.Cursor)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	page, err := h.articleUsecase.ListArticles(ctx, cursor, req.Limit)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	articles := page.Items
	if articles == nil {
		articles = []*model.Article{} // emit data.articles:[] rather than null
	}
	return respondPage(c, http.StatusOK, articleListData{Articles: articles}, page)
}

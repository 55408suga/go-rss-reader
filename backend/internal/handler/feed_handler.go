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

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// FeedHandler handles feed-related HTTP requests.
type FeedHandler struct {
	feedUsecase usecase.FeedUsecase
	logger      *slog.Logger
}

// RegisterFeedRequest is the payload for registering a feed URL.
type RegisterFeedRequest struct {
	FeedURL string `json:"feed_url" validate:"required,url"`
}

// ListFeedsRequest is the query params for listing feeds with pagination.
type ListFeedsRequest struct {
	CursorAt *time.Time `query:"cursor_at"` // RFC3339 timestamp
	CursorID *uuid.UUID `query:"cursor_id"` // UUID
	Limit    int        `query:"limit" validate:"gte=1,lte=100"`
}

// RegisterFeedResponse is the response payload after feed registration.
type RegisterFeedResponse struct {
	Feed     *model.Feed      `json:"feed"`
	Articles []*model.Article `json:"articles"`
}

var requestValidator = validator.New(validator.WithRequiredStructEnabled())

// NewFeedHandler creates a FeedHandler.
func NewFeedHandler(feedUsecase usecase.FeedUsecase, logger *slog.Logger) *FeedHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &FeedHandler{
		feedUsecase: feedUsecase,
		logger:      logger,
	}
}

// RegisterFeed validates and registers a feed with its current articles.
func (h *FeedHandler) RegisterFeed(c *echo.Context) error {
	const op = "FeedHandler.RegisterFeed"
	ctx := c.Request().Context()
	logger := applogger.WithContext(ctx, h.logger)

	var req RegisterFeedRequest
	if err := c.Bind(&req); err != nil {
		logger.WarnContext(ctx, "invalid request body", "error", err)
		return apperror.NewInvalidArgument(op, "invalid request body", err)
	}

	if err := requestValidator.Struct(req); err != nil {
		logger.WarnContext(ctx, "request validation failed", "error", err)
		return apperror.NewInvalidArgument(op, "validation failed", err)
	}

	feed, articles, err := h.feedUsecase.RegisterFeed(ctx, req.FeedURL)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusCreated, RegisterFeedResponse{
		Feed:     feed,
		Articles: articles,
	})
}

// ListFeeds returns registered feeds.
// Accepts optional query params cursor_at (RFC3339) and cursor_id (UUID) for pagination.
func (h *FeedHandler) ListFeeds(c *echo.Context) error {
	const op = "FeedHandler.ListFeeds"
	ctx := c.Request().Context()
	logger := applogger.WithContext(ctx, h.logger)

	var req ListFeedsRequest
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

	feeds, err := h.feedUsecase.ListFeeds(ctx, cursor, req.Limit)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, feeds)
}

// GetFeedByID returns a single feed by ID.
func (h *FeedHandler) GetFeedByID(c *echo.Context) error {
	const op = "FeedHandler.GetFeedByID"

	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewInvalidArgument(op, "invalid feed id", err)
	}

	feed, err := h.feedUsecase.GetFeedByID(c.Request().Context(), feedID)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, feed)
}

// RefreshFeed refreshes metadata/articles for a single feed.
func (h *FeedHandler) RefreshFeed(c *echo.Context) error {
	const op = "FeedHandler.RefreshFeed"

	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewInvalidArgument(op, "invalid feed id", err)
	}

	if err := h.feedUsecase.RefreshFeed(c.Request().Context(), feedID); err != nil {
		return apperror.Wrap(err, op)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteFeed deletes a feed by ID.
func (h *FeedHandler) DeleteFeed(c *echo.Context) error {
	const op = "FeedHandler.DeleteFeed"

	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewInvalidArgument(op, "invalid feed id", err)
	}

	err = h.feedUsecase.DeleteFeed(c.Request().Context(), feedID)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.NoContent(http.StatusNoContent)
}

// Package handler provides HTTP handlers for API endpoints.
package handler

import (
	"log/slog"
	"net/http"
	"time"
	applogger "rss_reader/internal/applog"
	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/usecase"

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
func (fh *FeedHandler) RegisterFeed(c *echo.Context) error {
	const op = "FeedHandler.RegisterFeed"

	var req RegisterFeedRequest
	if err := c.Bind(&req); err != nil {
		applogger.WithContext(c.Request().Context(), fh.logger).WarnContext(c.Request().Context(),
			"invalid request body",
			"error", err,
		)
		return apperror.NewInvalidArgument(op, "invalid request body", err)
	}

	if err := requestValidator.Struct(req); err != nil {
		applogger.WithContext(c.Request().Context(), fh.logger).WarnContext(c.Request().Context(),
			"request validation failed",
			"error", err,
		)
		return apperror.NewInvalidArgument(op, "validation failed", err)
	}

	feed, articles, err := fh.feedUsecase.RegisterFeed(c.Request().Context(), req.FeedURL)
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
func (fh *FeedHandler) ListFeeds(c *echo.Context) error {
	const op = "FeedHandler.ListFeeds"

	cursor := parseCursorFromQuery(c)
	feeds, err := fh.feedUsecase.ListFeeds(c.Request().Context(), cursor, 50)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, feeds)
}

// parseCursorFromQuery extracts a PageCursor from cursor_at and cursor_id query params.
// Returns nil if either param is absent or invalid.
func parseCursorFromQuery(c *echo.Context) *model.PageCursor {
	rawAt := c.QueryParam("cursor_at")
	rawID := c.QueryParam("cursor_id")
	if rawAt == "" || rawID == "" {
		return nil
	}
	at, err := time.Parse(time.RFC3339, rawAt)
	if err != nil {
		return nil
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil
	}
	return &model.PageCursor{At: at, ID: id}
}

// GetFeedByID returns a single feed by ID.
func (fh *FeedHandler) GetFeedByID(c *echo.Context) error {
	const op = "FeedHandler.GetFeedByID"

	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewInvalidArgument(op, "invalid feed id", err)
	}

	feed, err := fh.feedUsecase.GetFeedByID(c.Request().Context(), feedID)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, feed)
}

// RefreshFeed refreshes metadata/articles for a single feed.
func (fh *FeedHandler) RefreshFeed(c *echo.Context) error {
	const op = "FeedHandler.RefreshFeed"

	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewInvalidArgument(op, "invalid feed id", err)
	}

	if err := fh.feedUsecase.RefreshFeed(c.Request().Context(), feedID); err != nil {
		return apperror.Wrap(err, op)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteFeed deletes a feed by ID.
func (fh *FeedHandler) DeleteFeed(c *echo.Context) error {
	const op = "FeedHandler.DeleteFeed"

	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewInvalidArgument(op, "invalid feed id", err)
	}

	err = fh.feedUsecase.DeleteFeed(c.Request().Context(), feedID)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.NoContent(http.StatusNoContent)
}

// Package handler provides HTTP handlers for API endpoints.
package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"rss_reader/internal/apperror"
	applogger "rss_reader/internal/applog"
	"rss_reader/internal/domain/model"
	"rss_reader/internal/usecase"
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

// DiscoverFeedRequest is the payload for discovering and subscribing a
// website's advertised feed.
type DiscoverFeedRequest struct {
	WebsiteURL string `json:"website_url" validate:"required,url"`
}

// ListFeedsRequest is the query params for listing feeds with pagination.
type ListFeedsRequest struct {
	Cursor string `query:"cursor"` // opaque pagination token (empty = first page)
	Limit  int    `query:"limit" validate:"gte=1,lte=100"`
}

// RegisterFeedResponse is the data payload after feed registration.
type RegisterFeedResponse struct {
	Feed     *model.Feed      `json:"feed"`
	Articles []*model.Article `json:"articles"`
}

// DiscoverFeedResponse is the data payload after autodiscovery registration.
// Candidates holds every feed link found in the page (the registered one
// first) so clients can offer alternatives via POST /feeds.
type DiscoverFeedResponse struct {
	Feed       *model.Feed           `json:"feed"`
	Articles   []*model.Article      `json:"articles"`
	Candidates []model.FeedCandidate `json:"candidates"`
}

// feedListData is the data payload for a paginated feed list.
type feedListData struct {
	Feeds []*model.Feed `json:"feeds"`
}

// feedData is the data payload for a single feed.
type feedData struct {
	Feed *model.Feed `json:"feed"`
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
		return apperror.NewInvalidArgument(op, "validation failed", err).
			WithDetails(validationDetails(err))
	}
	if err := requireHTTPScheme(op, "feed_url", req.FeedURL); err != nil {
		logger.WarnContext(ctx, "request validation failed", "error", err)
		return err
	}

	feed, articles, err := h.feedUsecase.RegisterFeed(ctx, req.FeedURL)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	if articles == nil {
		articles = []*model.Article{} // emit data.articles:[] rather than null
	}
	return respondData(c, http.StatusCreated, RegisterFeedResponse{
		Feed:     feed,
		Articles: articles,
	})
}

// DiscoverAndRegisterFeed validates a website URL, discovers its advertised
// feed via HTML autodiscovery, and subscribes to the first candidate.
func (h *FeedHandler) DiscoverAndRegisterFeed(c *echo.Context) error {
	const op = "FeedHandler.DiscoverAndRegisterFeed"
	ctx := c.Request().Context()
	logger := applogger.WithContext(ctx, h.logger)

	var req DiscoverFeedRequest
	if err := c.Bind(&req); err != nil {
		logger.WarnContext(ctx, "invalid request body", "error", err)
		return apperror.NewInvalidArgument(op, "invalid request body", err)
	}

	if err := requestValidator.Struct(req); err != nil {
		logger.WarnContext(ctx, "request validation failed", "error", err)
		return apperror.NewInvalidArgument(op, "validation failed", err).
			WithDetails(validationDetails(err))
	}
	if err := requireHTTPScheme(op, "website_url", req.WebsiteURL); err != nil {
		logger.WarnContext(ctx, "request validation failed", "error", err)
		return err
	}

	feed, articles, candidates, err := h.feedUsecase.DiscoverAndRegisterFeed(ctx, req.WebsiteURL)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	if articles == nil {
		articles = []*model.Article{} // emit data.articles:[] rather than null
	}
	if candidates == nil {
		candidates = []model.FeedCandidate{} // emit data.candidates:[] rather than null
	}
	return respondData(c, http.StatusCreated, DiscoverFeedResponse{
		Feed:       feed,
		Articles:   articles,
		Candidates: candidates,
	})
}

// requireHTTPScheme rejects URLs whose scheme is not http/https. The url
// validator tag accepts any scheme (ftp, file, ...), but the discovery
// gateway must only ever fetch web pages (SSRF surface), so the allowlist
// is enforced here at the boundary.
func requireHTTPScheme(op, field, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return apperror.NewInvalidArgument(op, "validation failed", err).
			WithDetails([]apperror.FieldViolation{
				{Field: field, Reason: "must be an http or https URL"},
			})
	}
	return nil
}

// ListFeeds returns one page of registered feeds.
// Accepts optional query params cursor (opaque pagination token) and limit.
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
		return apperror.NewInvalidArgument(op, "validation failed", err).
			WithDetails(validationDetails(err))
	}

	cursor, err := decodeCursor(req.Cursor)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	page, err := h.feedUsecase.ListFeeds(ctx, cursor, req.Limit)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	feeds := page.Items
	if feeds == nil {
		feeds = []*model.Feed{} // emit data.feeds:[] rather than null
	}
	return respondPage(c, http.StatusOK, feedListData{Feeds: feeds}, page)
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

	return respondData(c, http.StatusOK, feedData{Feed: feed})
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

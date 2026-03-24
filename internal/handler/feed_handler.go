package handler

import (
	"errors"
	"net/http"
	"rss_reader/internal/usecase"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
)

type FeedHandler struct {
	feedUsecase usecase.FeedUsecase
}

type RegisterFeedRequest struct {
	FeedURL string `json:"feed_url" validate:"required,url"`
}

var requestValidator = validator.New(validator.WithRequiredStructEnabled())

func NewFeedHandler(feedUsecase usecase.FeedUsecase) *FeedHandler {
	return &FeedHandler{
		feedUsecase: feedUsecase,
	}
}

func (fh *FeedHandler) RegisterFeed(c *echo.Context) error {
	var req RegisterFeedRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := requestValidator.Struct(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	feed, err := fh.feedUsecase.RegisterFeed(c.Request().Context(), req.FeedURL)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to register feed")
	}

	return c.JSON(http.StatusCreated, feed)
}

func (fh *FeedHandler) GetAllFeeds(c *echo.Context) error {
	feeds, err := fh.feedUsecase.GetAllFeeds(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch feeds")
	}

	return c.JSON(http.StatusOK, feeds)
}

func (fh *FeedHandler) GetFeedByID(c *echo.Context) error {
	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid feed id")
	}

	feed, err := fh.feedUsecase.GetFeedByID(c.Request().Context(), feedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "feed not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch feed")
	}

	return c.JSON(http.StatusOK, feed)
}

func (fh *FeedHandler) DeleteFeed(c *echo.Context) error {
	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid feed id")
	}

	err = fh.feedUsecase.DeleteFeed(c.Request().Context(), feedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "feed not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete feed")
	}

	return c.NoContent(http.StatusNoContent)
}

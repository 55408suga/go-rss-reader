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

// GetArticlesByFeedID returns articles for a feed.
func (ah *ArticleHandler) GetArticlesByFeedID(c *echo.Context) error {
	const op = "ArticleHandler.GetArticlesByFeedID"

	feedID, err := uuid.Parse(c.Param("feed_id"))
	if err != nil {
		applogger.WithContext(c.Request().Context(), ah.logger).WarnContext(c.Request().Context(),
			"invalid feed id",
			"error", err,
		)
		return apperror.NewInvalidArgument(op, "invalid feed id", err)
	}

	articles, err := ah.articleUsecase.GetArticlesByFeedID(c.Request().Context(), feedID)
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, articles)
}

// GetAllArticles returns all articles.
func (ah *ArticleHandler) GetAllArticles(c *echo.Context) error {
	const op = "ArticleHandler.GetAllArticles"

	articles, err := ah.articleUsecase.GetAllArticles(c.Request().Context())
	if err != nil {
		return apperror.Wrap(err, op)
	}

	return c.JSON(http.StatusOK, articles)
}

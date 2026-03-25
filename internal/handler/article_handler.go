package handler

import (
	"net/http"
	"rss_reader/internal/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type ArticleHandler struct {
	articleUsecase usecase.ArticleUsecase
}

func NewArticleHandler(articleUsecase usecase.ArticleUsecase) *ArticleHandler {
	return &ArticleHandler{articleUsecase: articleUsecase}
}

func (ah *ArticleHandler) GetArticlesByFeedID(c *echo.Context) error {
	feedID, err := uuid.Parse(c.Param("feed_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid feed id")
	}

	articles, err := ah.articleUsecase.GetArticlesByFeedID(c.Request().Context(), feedID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch articles")
	}

	return c.JSON(http.StatusOK, articles)
}

// Package router registers HTTP routes.
package router

import (
	"rss_reader/internal/di"

	"github.com/labstack/echo/v5"
)

// SetupRoutes registers API routes.
func SetupRoutes(e *echo.Echo, components *di.ApplicationComponents) {
	api := e.Group("/api")
	v1 := api.Group("/v1")
	// Feed routes
	v1.POST("/feeds", components.FeedHandler.RegisterFeed)
	v1.GET("/feeds", components.FeedHandler.ListFeeds)
	v1.GET("/feeds/:id", components.FeedHandler.GetFeedByID)
	v1.POST("/feeds/:id/refresh", components.FeedHandler.RefreshFeed)
	v1.POST("/feeds/refresh", components.FeedHandler.RefreshAllFeeds)
	v1.DELETE("/feeds/:id", components.FeedHandler.DeleteFeed)

	// Article routes
	v1.GET("/articles", components.ArticleHandler.ListArticles)
	v1.GET("/feeds/:feed_id/articles", components.ArticleHandler.ListArticlesByFeedID)
}

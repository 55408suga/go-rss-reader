package di

import (
	"context"
	"rss_reader/internal/handler"
	"rss_reader/internal/infra/config"
	"rss_reader/internal/infra/gateway"
	"rss_reader/internal/infra/persistence/postgres"
	"rss_reader/internal/usecase"
)

type ApplicationComponents struct {
	FeedHandler    handler.FeedHandler
	ArticleHandler handler.ArticleHandler
	close          func() error
}

func NewApplicationComponents() *ApplicationComponents {
	ctx := context.Background()
	// ── 1. Config ──
	config := config.NewConfig()

	// ── 2. DB ──
	db, err := postgres.NewDB(ctx, config.DatabaseURL)
	if err != nil {
		panic(err)
	}

	// ── 3. Repository ──
	feedRepo := postgres.NewFeedRepository(db)
	articleRepo := postgres.NewArticleRepository(db)

	// ── 4. Gateway ──
	rssGateway := gateway.NewRSSGateway()

	// ── 5. Usecase ──
	feedUC := usecase.NewFeedInteractor(feedRepo, rssGateway)
	articleUC := usecase.NewArticleInteractor(articleRepo, feedRepo, rssGateway)

	// ── 6. Handler ──
	feedHandler := handler.NewFeedHandler(feedUC)
	articleHandler := handler.NewArticleHandler(articleUC)

	return &ApplicationComponents{
		FeedHandler:    *feedHandler,
		ArticleHandler: *articleHandler,
		close:          func() error { db.Close(); return nil },
	}
}

func (ac *ApplicationComponents) Close() error {
	if ac.close != nil {
		return ac.close()
	}
	return nil
}

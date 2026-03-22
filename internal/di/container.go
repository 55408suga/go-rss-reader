package di

import (
	"context"
	"rss_reader/internal/infra/config"
	"rss_reader/internal/infra/gateway"
	"rss_reader/internal/infra/persistence/postgres"
	"rss_reader/internal/usecase"
)

// ApplicationComponents はアプリケーションの全依存を束ねる構造体。
// main.go からはこの構造体を通じてハンドラにアクセスする。
type ApplicationComponents struct {
	FeedUsecase    usecase.FeedUsecase
	ArticleUsecase usecase.ArticleUsecase
}

//	func main() {
//	    components := di.NewApplicationComponents()
//	    router.Setup(e, components)
//	}
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
	// feedHandler := handler.NewFeedHandler(feedUC)

	return &ApplicationComponents{
		FeedUsecase:    feedUC,
		ArticleUsecase: articleUC,
	}
}

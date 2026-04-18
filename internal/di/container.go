// Package di wires application dependencies.
package di

import (
	"context"
	"fmt"
	"log/slog"
	"rss_reader/internal/handler"
	"rss_reader/internal/infra/config"
	"rss_reader/internal/infra/gateway"
	"rss_reader/internal/infra/persistence/postgres"
	"rss_reader/internal/job"
	"rss_reader/internal/usecase"
	"time"
)

// ApplicationComponents holds fully wired application entry-point dependencies.
type ApplicationComponents struct {
	FeedHandler    handler.FeedHandler
	ArticleHandler handler.ArticleHandler
	Scheduler      *job.Scheduler
	close          func() error
}

// NewApplicationComponents wires application dependencies.
func NewApplicationComponents(cfg *config.Config, logger *slog.Logger) (*ApplicationComponents, error) {
	ctx := context.Background()

	if cfg == nil {
		cfg = config.NewConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	db, err := postgres.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database initialization failed: %w", err)
	}

	feedRepo := postgres.NewFeedRepository(db, logger)
	articleRepo := postgres.NewArticleRepository(db, logger)
	feedStatusRepo := postgres.NewFetchStatusRepository(db, logger)

	rssGateway := gateway.NewRSSGateway(gateway.NewHTTPClient(), logger)

	txManager := postgres.NewPgTransactionManager(db, logger)

	feedUC := usecase.NewFeedInteractor(feedRepo, articleRepo, feedStatusRepo, rssGateway, txManager)
	articleUC := usecase.NewArticleInteractor(articleRepo)
	feedJobUC := usecase.NewFeedJobInteractor(
		rssGateway, articleRepo, feedStatusRepo, txManager, logger,
	)

	feedHandler := handler.NewFeedHandler(feedUC, logger)
	articleHandler := handler.NewArticleHandler(articleUC, logger)

	scheduler := job.NewJobScheduler(logger)
	scheduler.Add(job.Job{
		Name:     "refresh-due-feeds",
		Interval: 10 * time.Minute,
		Timeout:  5 * time.Minute,
		Fnc:      feedJobUC.RefreshDueFeeds,
	})

	return &ApplicationComponents{
		FeedHandler:    *feedHandler,
		ArticleHandler: *articleHandler,
		Scheduler:      scheduler,
		close:          func() error { db.Close(); return nil },
	}, nil
}

// Close releases resources held by components.
func (ac *ApplicationComponents) Close() error {
	if ac.close != nil {
		return ac.close()
	}
	return nil
}

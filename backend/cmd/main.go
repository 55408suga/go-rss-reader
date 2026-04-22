// Package main starts the RSS reader HTTP server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"rss_reader/internal/di"
	"rss_reader/internal/infra/config"
	applogger "rss_reader/internal/infra/logger"
	appmiddleware "rss_reader/internal/infra/middleware"
	"rss_reader/internal/infra/router"
	"rss_reader/internal/job"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	cfg := config.NewConfig()
	logger := applogger.New(cfg)
	slog.SetDefault(logger)

	components, err := di.NewApplicationComponents(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := components.Close(); closeErr != nil {
			logger.Error("failed to close components", "error", closeErr)
		}
	}()

	e := echo.NewWithConfig(echo.Config{
		Logger:           logger,
		HTTPErrorHandler: appmiddleware.NewGlobalErrorHandler(logger),
	})

	e.Use(middleware.RequestID())
	e.Use(appmiddleware.RequestIDContext())
	e.Use(appmiddleware.RequestLogger(logger))
	e.Use(middleware.Recover())
	e.Use(middleware.ContextTimeout(15 * time.Second))

	router.SetupRoutes(e, components)
	sc := echo.StartConfig{
		Address:         ":8080",
		GracefulTimeout: 5 * time.Second,
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	scheduler := job.NewJobScheduler(logger)
	scheduler.Add(job.Job{
		Name:     "refresh-due-feeds",
		Interval: 10 * time.Minute,
		Timeout:  5 * time.Minute,
		Fnc:      components.FeedJobUC.RefreshDueFeeds,
	})
	scheduler.Start(ctx)

	if err := sc.Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("failed to start server", "error", err)
	}

	scheduler.Shutdown()
}

// Package main starts the RSS reader HTTP server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"rss_reader/internal/di"
	"rss_reader/internal/infra/config"
	applogger "rss_reader/internal/infra/logger"
	appmiddleware "rss_reader/internal/infra/middleware"
	"rss_reader/internal/infra/router"
	"rss_reader/internal/job"
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

	ctx, appCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer appCancel()

	scheduler := job.NewJobScheduler(logger)
	scheduler.Add(job.Job{
		Name:     "refresh-due-feeds",
		Interval: 10 * time.Minute,
		Timeout:  5 * time.Minute,
		Func:     components.FeedJobUC.RefreshDueFeeds,
	})
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer shutdownCancel()
		if err := scheduler.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shutdown scheduler", "error", err)
		}
	}()

	e := echo.NewWithConfig(echo.Config{
		Logger:           logger,
		HTTPErrorHandler: appmiddleware.NewGlobalErrorHandler(logger),
	})
	e.Use(middleware.RequestID())
	e.Use(appmiddleware.RequestIDContext())
	// CORS sits right after request-id so even preflight/rejected requests are
	// correlated, but before the logger/recover/timeout and the router: Echo's
	// CORS short-circuits OPTIONS preflight with a 204, so there is no point
	// running those for a no-op preflight.
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowCredentials: false,
	}))
	e.Use(appmiddleware.RequestLogger(logger))
	e.Use(middleware.Recover())
	e.Use(middleware.ContextTimeout(15 * time.Second))
	router.SetupRoutes(e, components)

	sc := echo.StartConfig{
		Address:         ":8080",
		GracefulTimeout: 5 * time.Second,
		OnShutdownError: func(err error) {
			logger.Error("failed to graceful shutdown", "error", err)
		},
		BeforeServeFunc: func(_ *http.Server) error {
			scheduler.Start(ctx)
			return nil
		},
	}

	if err := sc.Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("failed to start server", "error", err)
	}
}

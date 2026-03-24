package main

import (
	"rss_reader/internal/di"
	"rss_reader/internal/infra/router"

	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	components := di.NewApplicationComponents()
	defer components.Close()

	e := echo.New()

	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.ContextTimeout(15 * time.Second))

	router.SetupRoutes(e, components)
	sc := echo.StartConfig{
		Address: ":8080",
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) // start shutdown process on signal
	defer cancel()
	if err := sc.Start(ctx, e); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}

}

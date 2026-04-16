package middleware

import (
	"log/slog"
	"rss_reader/internal/applog"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
)

// RequestLogger writes structured access logs via slog.
func RequestLogger(baseLogger *slog.Logger) echo.MiddlewareFunc {
	if baseLogger == nil {
		baseLogger = slog.Default()
	}

	return echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogValuesFunc: func(c *echo.Context, v echomw.RequestLoggerValues) error {
			ctx := c.Request().Context()
			if v.RequestID != "" && applog.RequestID(ctx) == "" {
				ctx = applog.WithRequestID(ctx, v.RequestID)
				c.SetRequest(c.Request().WithContext(ctx))
			}

			requestLogger := applog.WithContext(ctx, baseLogger)
			attrs := []any{
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency_ms", v.Latency.Milliseconds(),
			}

			if v.Error != nil {
				requestLogger.ErrorContext(ctx, "request completed with error", append(attrs, "error", v.Error)...)
				return nil
			}

			switch {
			case v.Status >= 500:
				requestLogger.ErrorContext(ctx, "request completed", attrs...)
			case v.Status >= 400:
				requestLogger.WarnContext(ctx, "request completed", attrs...)
			default:
				requestLogger.InfoContext(ctx, "request completed", attrs...)
			}
			return nil
		},
	})
}

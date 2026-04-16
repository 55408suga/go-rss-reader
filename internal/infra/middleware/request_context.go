package middleware

import (
	"rss_reader/internal/infra/logger"

	"github.com/labstack/echo/v5"
)

// RequestIDContext adds request_id to context so downstream logs can correlate events.
func RequestIDContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			requestID := requestIDFromEcho(c)
			if requestID != "" {
				req := c.Request()
				ctx := logger.WithRequestID(req.Context(), requestID)
				c.SetRequest(req.WithContext(ctx))
			}
			return next(c)
		}
	}
}

func requestIDFromEcho(c *echo.Context) string {
	requestID := c.Request().Header.Get(echo.HeaderXRequestID)
	if requestID != "" {
		return requestID
	}
	return c.Response().Header().Get(echo.HeaderXRequestID)
}

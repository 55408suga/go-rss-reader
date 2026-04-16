// Package middleware provides HTTP middleware for logging and error handling.
package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"rss_reader/internal/apperror"
	"rss_reader/internal/infra/logger"

	"github.com/labstack/echo/v5"
)

type errorResponse struct {
	Error errorBody `json:"error"`
	Meta  metaBody  `json:"meta"`
}

type errorBody struct {
	Code    apperror.Code `json:"code"`
	Message string        `json:"message"`
}

type metaBody struct {
	RequestID string `json:"request_id"`
}

// NewGlobalErrorHandler converts returned errors into standardized JSON responses.
func NewGlobalErrorHandler(baseLogger *slog.Logger) echo.HTTPErrorHandler {
	if baseLogger == nil {
		baseLogger = slog.Default()
	}

	return func(c *echo.Context, err error) {
		if res, unwrapErr := echo.UnwrapResponse(c.Response()); unwrapErr == nil {
			if res.Committed {
				return
			}
		}

		requestID := requestIDFromEcho(c)
		ctx := c.Request().Context()
		if requestID != "" && logger.RequestID(ctx) == "" {
			ctx = logger.WithRequestID(ctx, requestID)
			c.SetRequest(c.Request().WithContext(ctx))
		}
		log := logger.WithContext(ctx, baseLogger)

		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			status := appErr.Code.HTTPStatus()
			message := appErr.Message
			if status >= http.StatusInternalServerError && message == "" {
				message = "internal server error"
			}
			writeErrorJSON(c, status, appErr.Code, message, requestID)
			if status >= http.StatusInternalServerError {
				log.ErrorContext(ctx, "request failed",
					"code", appErr.Code,
					"op", appErr.Op,
					"error", appErr.Err,
				)
			}
			return
		}

		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			status := httpErr.Code
			code := apperror.CodeFromStatus(status)
			message := httpErr.Message
			if status >= http.StatusInternalServerError {
				message = "internal server error"
			}
			if message == "" {
				message = http.StatusText(status)
			}
			writeErrorJSON(c, status, code, message, requestID)
			if status >= http.StatusInternalServerError {
				log.ErrorContext(ctx, "request failed",
					"code", code,
					"status", status,
					"error", err,
				)
			}
			return
		}

		writeErrorJSON(c, http.StatusInternalServerError, apperror.CodeInternal, "internal server error", requestID)
		log.ErrorContext(ctx, "request failed",
			"code", apperror.CodeInternal,
			"error", err,
		)
	}
}

func writeErrorJSON(c *echo.Context, status int, code apperror.Code, message, requestID string) {
	response := errorResponse{
		Error: errorBody{Code: code, Message: message},
		Meta:  metaBody{RequestID: requestID},
	}
	_ = c.JSON(status, response)
}

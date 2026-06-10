// Package middleware provides HTTP middleware for logging and error handling.
package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"rss_reader/internal/apiresp"
	"rss_reader/internal/apperror"
	"rss_reader/internal/applog"
)

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
		if requestID != "" && applog.RequestID(ctx) == "" {
			ctx = applog.WithRequestID(ctx, requestID)
			c.SetRequest(c.Request().WithContext(ctx))
		}
		log := applog.WithContext(ctx, baseLogger)

		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			status := statusFromCode(appErr.Code)
			message := publicMessage(status, appErr.Message)
			writeErrorJSON(c, status, appErr.Code, message, requestID, appErr.Details)
			if status >= http.StatusInternalServerError {
				log.ErrorContext(ctx, "request failed",
					"code", appErr.Code,
					"op", appErr.Op,
					"error", appErr.Err,
				)
			}
			return
		}

		// Echo reports routing failures (unknown route -> 404, wrong method -> 405)
		// and handler-raised HTTP errors through values that implement
		// echo.HTTPStatusCoder. echo.StatusCode matches BOTH the exported
		// *echo.HTTPError and Echo's built-in sentinels (echo.ErrNotFound,
		// echo.ErrMethodNotAllowed, ...) via errors.As, returning 0 when the error
		// carries no HTTP status. The previous errors.As(&*echo.HTTPError) check
		// missed the sentinels and mislabeled 404/405 as 500.
		if status := echo.StatusCode(err); status != 0 {
			code := codeFromStatus(status)

			// Only a handler-constructed *echo.HTTPError carries a caller-supplied
			// message; Echo's built-in sentinels (echo.ErrNotFound, ...) do not.
			// errors.As reaches it even when wrapped; for the sentinels message stays
			// empty and publicMessage falls back to http.StatusText(status).
			var message string
			var httpErr *echo.HTTPError
			if errors.As(err, &httpErr) {
				message = httpErr.Message
			}

			message = publicMessage(status, message)
			writeErrorJSON(c, status, code, message, requestID, nil)
			if status >= http.StatusInternalServerError {
				log.ErrorContext(ctx, "request failed",
					"code", code,
					"status", status,
					"error", err,
				)
			}
			return
		}

		writeErrorJSON(
			c, http.StatusInternalServerError, apperror.CodeInternal,
			"internal server error", requestID, nil,
		)
		log.ErrorContext(ctx, "request failed",
			"code", apperror.CodeInternal,
			"error", err,
		)
	}
}

func writeErrorJSON(
	c *echo.Context,
	status int,
	code apperror.Code,
	message, requestID string,
	details []apperror.FieldViolation,
) {
	response := apiresp.ErrorEnvelope{
		Error: apiresp.ErrorBody{
			Code:    string(code),
			Message: message,
			Details: toFieldErrors(details),
		},
		Meta: apiresp.Meta{RequestID: requestID},
	}
	_ = c.JSON(status, response)
}

// toFieldErrors maps domain violations onto the transport DTO, returning nil
// (so details is omitted) when there are none.
func toFieldErrors(violations []apperror.FieldViolation) []apiresp.FieldError {
	if len(violations) == 0 {
		return nil
	}
	out := make([]apiresp.FieldError, len(violations))
	for i, v := range violations {
		out[i] = apiresp.FieldError{Field: v.Field, Reason: v.Reason}
	}
	return out
}

func statusFromCode(code apperror.Code) int {
	switch code {
	case apperror.CodeNotFound:
		return http.StatusNotFound
	case apperror.CodeInvalidArgument:
		return http.StatusBadRequest
	case apperror.CodeConflict:
		return http.StatusConflict
	case apperror.CodeExternalUnavailable:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func codeFromStatus(status int) apperror.Code {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return apperror.CodeInvalidArgument
	case http.StatusNotFound:
		return apperror.CodeNotFound
	case http.StatusConflict:
		return apperror.CodeConflict
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return apperror.CodeExternalUnavailable
	default:
		if status >= http.StatusInternalServerError {
			return apperror.CodeInternal
		}
		return apperror.CodeInvalidArgument
	}
}

func publicMessage(status int, message string) string {
	if status >= http.StatusInternalServerError {
		return "internal server error"
	}
	if message == "" {
		return http.StatusText(status)
	}
	return message
}

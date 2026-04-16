// Package applog provides context helpers for correlated structured logging.
package applog

import (
	"context"
	"log/slog"
)

type requestIDKey struct{}

// WithRequestID returns a context carrying request_id for log correlation.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID extracts request_id from context.
func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// WithContext attaches context-derived attributes (request_id) to a logger.
func WithContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	if requestID := RequestID(ctx); requestID != "" {
		return base.With("request_id", requestID)
	}
	return base
}

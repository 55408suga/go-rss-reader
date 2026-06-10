package handler

import (
	"github.com/labstack/echo/v5"

	"rss_reader/internal/apiresp"
	applogger "rss_reader/internal/applog"
	"rss_reader/internal/domain/model"
)

// requestIDOf reads the correlation id attached by the RequestIDContext
// middleware, so success envelopes carry the same meta.request_id as the
// errors produced by the global error handler.
func requestIDOf(c *echo.Context) string {
	return applogger.RequestID(c.Request().Context())
}

// respondData writes a non-paginated success envelope: {data, meta}.
func respondData[T any](c *echo.Context, status int, data T) error {
	return c.JSON(status, apiresp.Envelope[T]{
		Data: data,
		Meta: apiresp.Meta{RequestID: requestIDOf(c)},
	})
}

// respondPage writes a paginated success envelope. The page's structured next
// cursor is wrapped into an opaque token here, at the transport boundary, and
// exposed under meta.pagination. D is the data wrapper (e.g. feedListData); T
// is the page item type, both inferred from the arguments.
func respondPage[D, T any](c *echo.Context, status int, data D, page *model.Page[T]) error {
	return c.JSON(status, apiresp.Envelope[D]{
		Data: data,
		Meta: apiresp.Meta{
			RequestID: requestIDOf(c),
			Pagination: &apiresp.Pagination{
				NextCursor: encodeCursor(page.NextCursor),
				HasMore:    page.HasMore,
			},
		},
	})
}

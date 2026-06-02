package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"rss_reader/internal/applog"
)

func TestRequestIDContextPropagatesHeaderToContext(t *testing.T) {
	t.Parallel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(echo.HeaderXRequestID, "req-42")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var seen string
	next := func(c *echo.Context) error {
		seen = applog.RequestID(c.Request().Context())
		return nil
	}

	if err := RequestIDContext()(next)(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen != "req-42" {
		t.Errorf("propagated request_id = %q, want req-42", seen)
	}
}

func TestRequestIDContextWithoutHeaderLeavesContextEmpty(t *testing.T) {
	t.Parallel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	seen := "sentinel"
	next := func(c *echo.Context) error {
		seen = applog.RequestID(c.Request().Context())
		return nil
	}

	if err := RequestIDContext()(next)(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen != "" {
		t.Errorf("propagated request_id = %q, want empty", seen)
	}
}

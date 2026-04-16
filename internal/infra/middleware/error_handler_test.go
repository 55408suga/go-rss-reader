package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"rss_reader/internal/apperror"
	"testing"

	"github.com/labstack/echo/v5"
)

type errorResponseForTest struct {
	Error struct {
		Code    apperror.Code `json:"code"`
		Message string        `json:"message"`
	} `json:"error"`
	Meta struct {
		RequestID string `json:"request_id"`
	} `json:"meta"`
}

func TestGlobalErrorHandlerAppError(t *testing.T) {
	e := echo.New()
	handler := NewGlobalErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderXRequestID, "req-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler(c, apperror.NewNotFound("test.op", "feed not found", errors.New("no rows")))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	var body errorResponseForTest
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body.Error.Code != apperror.CodeNotFound {
		t.Fatalf("expected code %q, got %q", apperror.CodeNotFound, body.Error.Code)
	}

	if body.Error.Message != "feed not found" {
		t.Fatalf("unexpected message: %s", body.Error.Message)
	}

	if body.Meta.RequestID != "req-1" {
		t.Fatalf("unexpected request_id: %s", body.Meta.RequestID)
	}
}

func TestGlobalErrorHandlerEchoHTTPError(t *testing.T) {
	e := echo.New()
	handler := NewGlobalErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler(c, echo.NewHTTPError(http.StatusBadRequest, "bad request"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var body errorResponseForTest
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body.Error.Code != apperror.CodeInvalidArgument {
		t.Fatalf("expected code %q, got %q", apperror.CodeInvalidArgument, body.Error.Code)
	}
}

func TestGlobalErrorHandlerUnknownError(t *testing.T) {
	e := echo.New()
	handler := NewGlobalErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler(c, errors.New("unexpected"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var body errorResponseForTest
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body.Error.Code != apperror.CodeInternal {
		t.Fatalf("expected code %q, got %q", apperror.CodeInternal, body.Error.Code)
	}

	if body.Error.Message != "internal server error" {
		t.Fatalf("unexpected message: %s", body.Error.Message)
	}
}

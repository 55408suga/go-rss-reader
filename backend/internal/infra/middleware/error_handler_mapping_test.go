package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"rss_reader/internal/apperror"
)

func TestStatusFromCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code apperror.Code
		want int
	}{
		{"not found", apperror.CodeNotFound, http.StatusNotFound},
		{"invalid argument", apperror.CodeInvalidArgument, http.StatusBadRequest},
		{"conflict", apperror.CodeConflict, http.StatusConflict},
		{"external unavailable", apperror.CodeExternalUnavailable, http.StatusBadGateway},
		{"internal", apperror.CodeInternal, http.StatusInternalServerError},
		{"unknown code defaults to 500", apperror.Code("weird"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := statusFromCode(tc.code); got != tc.want {
				t.Errorf("statusFromCode(%q) = %d, want %d", tc.code, got, tc.want)
			}
		})
	}
}

func TestCodeFromStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		want   apperror.Code
	}{
		{"400 bad request", http.StatusBadRequest, apperror.CodeInvalidArgument},
		{"422 unprocessable", http.StatusUnprocessableEntity, apperror.CodeInvalidArgument},
		{"404 not found", http.StatusNotFound, apperror.CodeNotFound},
		{"409 conflict", http.StatusConflict, apperror.CodeConflict},
		{"502 bad gateway", http.StatusBadGateway, apperror.CodeExternalUnavailable},
		{"503 unavailable", http.StatusServiceUnavailable, apperror.CodeExternalUnavailable},
		{"504 gateway timeout", http.StatusGatewayTimeout, apperror.CodeExternalUnavailable},
		{"500 internal", http.StatusInternalServerError, apperror.CodeInternal},
		{"other 4xx defaults to invalid argument", http.StatusTeapot, apperror.CodeInvalidArgument},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := codeFromStatus(tc.status); got != tc.want {
				t.Errorf("codeFromStatus(%d) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestPublicMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		message string
		want    string
	}{
		{"5xx scrubs the message", http.StatusInternalServerError, "db password leaked", "internal server error"},
		{"4xx keeps the message", http.StatusBadRequest, "invalid feed url", "invalid feed url"},
		{"4xx empty falls back to status text", http.StatusNotFound, "", http.StatusText(http.StatusNotFound)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := publicMessage(tc.status, tc.message); got != tc.want {
				t.Errorf("publicMessage(%d, %q) = %q, want %q", tc.status, tc.message, got, tc.want)
			}
		})
	}
}

func TestGlobalErrorHandlerAppErrorStatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         *apperror.AppError
		wantStatus  int
		wantCode    apperror.Code
		wantMessage string
	}{
		{
			name:        "conflict maps to 409",
			err:         apperror.NewConflict("op", "already exists", nil),
			wantStatus:  http.StatusConflict,
			wantCode:    apperror.CodeConflict,
			wantMessage: "already exists",
		},
		{
			name:        "invalid argument maps to 400",
			err:         apperror.NewInvalidArgument("op", "bad input", nil),
			wantStatus:  http.StatusBadRequest,
			wantCode:    apperror.CodeInvalidArgument,
			wantMessage: "bad input",
		},
		{
			// 502 is a 5xx status, so the human-readable message is scrubbed even
			// though the machine-readable code stays exposed.
			name:        "external unavailable maps to 502 and scrubs the message",
			err:         apperror.NewExternalUnavailable("op", "upstream down", nil),
			wantStatus:  http.StatusBadGateway,
			wantCode:    apperror.CodeExternalUnavailable,
			wantMessage: "internal server error",
		},
		{
			name:        "internal maps to 500 and scrubs the message",
			err:         apperror.NewInternal("op", "stack trace with secrets", nil),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    apperror.CodeInternal,
			wantMessage: "internal server error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			handler := NewGlobalErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler(c, tc.err)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			var body errorResponseForTest
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Error.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tc.wantCode)
			}
			if body.Error.Message != tc.wantMessage {
				t.Errorf("message = %q, want %q", body.Error.Message, tc.wantMessage)
			}
		})
	}
}

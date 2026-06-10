package handler

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestCursorRoundTrip(t *testing.T) {
	t.Parallel()

	original := &model.PageCursor{
		At: time.Date(2026, 6, 3, 12, 30, 45, 123456789, time.UTC),
		ID: uuid.New(),
	}

	token := encodeCursor(original)
	if token == nil {
		t.Fatal("encodeCursor returned nil for a non-nil cursor")
	}

	got, err := decodeCursor(*token)
	if err != nil {
		t.Fatalf("decodeCursor: %v", err)
	}
	if !got.At.Equal(original.At) || got.ID != original.ID {
		t.Errorf("round trip = %+v, want %+v", got, original)
	}
}

func TestEncodeCursorNilReturnsNil(t *testing.T) {
	t.Parallel()

	if got := encodeCursor(nil); got != nil {
		t.Errorf("encodeCursor(nil) = %q, want nil", *got)
	}
}

func TestDecodeCursorEmptyReturnsNil(t *testing.T) {
	t.Parallel()

	got, err := decodeCursor("")
	if err != nil {
		t.Fatalf("decodeCursor(\"\") err = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("decodeCursor(\"\") = %+v, want nil", got)
	}
}

func TestDecodeCursorInvalidIsInvalidArgument(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"not base64":            "!!! not base64 !!!",
		"valid base64 not json": base64.RawURLEncoding.EncodeToString([]byte("foobar")),
		// Decodable JSON whose shape is not a usable keyset position.
		"empty object is zero-value cursor": base64.RawURLEncoding.EncodeToString([]byte(`{}`)),
		"missing ID":                        base64.RawURLEncoding.EncodeToString([]byte(`{"At":"2026-06-03T12:00:00Z"}`)),
		"missing At": base64.RawURLEncoding.EncodeToString(
			[]byte(`{"ID":"` + uuid.New().String() + `"}`)),
	}

	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeCursor(token)

			var appErr *apperror.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("error = %v (%T), want *apperror.AppError", err, err)
			}
			if appErr.Code != apperror.CodeInvalidArgument {
				t.Errorf("code = %q, want %q", appErr.Code, apperror.CodeInvalidArgument)
			}
		})
	}
}

package apperror

import (
	"errors"
	"testing"
)

func TestWrapPreservesAppErrorMetadata(t *testing.T) {
	root := errors.New("root cause")
	base := NewNotFound("FeedRepository.GetFeedByID", "resource not found", root)

	wrapped := Wrap(base, "FeedInteractor.GetFeedByID")

	var appErr *AppError
	if !errors.As(wrapped, &appErr) {
		t.Fatalf("expected AppError, got %T", wrapped)
	}

	if appErr.Code != CodeNotFound {
		t.Fatalf("expected code %q, got %q", CodeNotFound, appErr.Code)
	}

	if appErr.Op != "FeedInteractor.GetFeedByID: FeedRepository.GetFeedByID" {
		t.Fatalf("unexpected op chain: %s", appErr.Op)
	}

	if !errors.Is(appErr, root) {
		t.Fatalf("expected wrapped error to contain root cause")
	}
}

func TestWrapCreatesInternalForUnknownError(t *testing.T) {
	err := errors.New("boom")
	wrapped := Wrap(err, "Any.Operation")

	var appErr *AppError
	if !errors.As(wrapped, &appErr) {
		t.Fatalf("expected AppError, got %T", wrapped)
	}

	if appErr.Code != CodeInternal {
		t.Fatalf("expected code %q, got %q", CodeInternal, appErr.Code)
	}

	if appErr.Message != "internal server error" {
		t.Fatalf("unexpected message: %s", appErr.Message)
	}
}

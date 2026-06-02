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

func TestConstructorsSetCode(t *testing.T) {
	t.Parallel()

	cause := errors.New("cause")

	tests := []struct {
		name      string
		construct func(op, message string, err error) *AppError
		wantCode  Code
	}{
		{"not found", NewNotFound, CodeNotFound},
		{"invalid argument", NewInvalidArgument, CodeInvalidArgument},
		{"conflict", NewConflict, CodeConflict},
		{"external unavailable", NewExternalUnavailable, CodeExternalUnavailable},
		{"internal", NewInternal, CodeInternal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.construct("Pkg.Method", "boom", cause)
			if err.Code != tc.wantCode {
				t.Errorf("Code = %q, want %q", err.Code, tc.wantCode)
			}
			if err.Op != "Pkg.Method" {
				t.Errorf("Op = %q, want %q", err.Op, "Pkg.Method")
			}
			if err.Message != "boom" {
				t.Errorf("Message = %q, want %q", err.Message, "boom")
			}
			if !errors.Is(err, cause) {
				t.Error("errors.Is(err, cause) = false, want true")
			}
		})
	}
}

func TestWrapEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns nil", func(t *testing.T) {
		t.Parallel()
		if got := Wrap(nil, "Op"); got != nil {
			t.Errorf("Wrap(nil) = %v, want nil", got)
		}
	})

	t.Run("empty op keeps the existing op and code", func(t *testing.T) {
		t.Parallel()
		base := NewConflict("Repo.Save", "duplicate", nil)
		var appErr *AppError
		if !errors.As(Wrap(base, ""), &appErr) {
			t.Fatal("expected AppError")
		}
		if appErr.Op != "Repo.Save" {
			t.Errorf("Op = %q, want %q", appErr.Op, "Repo.Save")
		}
		if appErr.Code != CodeConflict {
			t.Errorf("Code = %q, want %q", appErr.Code, CodeConflict)
		}
	})

	t.Run("empty existing op is replaced by op", func(t *testing.T) {
		t.Parallel()
		base := &AppError{Code: CodeInvalidArgument, Message: "bad", Op: ""}
		var appErr *AppError
		if !errors.As(Wrap(base, "Handler.Bind"), &appErr) {
			t.Fatal("expected AppError")
		}
		if appErr.Op != "Handler.Bind" {
			t.Errorf("Op = %q, want %q", appErr.Op, "Handler.Bind")
		}
	})

	t.Run("does not mutate the original", func(t *testing.T) {
		t.Parallel()
		base := NewNotFound("Repo.Get", "missing", nil)
		_ = Wrap(base, "Interactor.Get")
		if base.Op != "Repo.Get" {
			t.Errorf("original Op mutated to %q, want %q", base.Op, "Repo.Get")
		}
	})
}

func TestAppErrorErrorString(t *testing.T) {
	t.Parallel()

	cause := errors.New("disk full")

	tests := []struct {
		name string
		err  *AppError
		want string
	}{
		{
			name: "with cause appends the cause",
			err:  &AppError{Op: "Repo.Save", Message: "failed", Err: cause},
			want: "Repo.Save: failed: disk full",
		},
		{
			name: "without cause omits the trailing segment",
			err:  &AppError{Op: "Repo.Save", Message: "failed"},
			want: "Repo.Save: failed",
		},
		{
			name: "nil receiver is the empty string",
			err:  nil,
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("cause")
	if got := errors.Unwrap(NewInternal("Op", "msg", cause)); !errors.Is(got, cause) {
		t.Errorf("Unwrap() = %v, want %v", got, cause)
	}
	if got := errors.Unwrap(NewInternal("Op", "msg", nil)); got != nil {
		t.Errorf("Unwrap() with no cause = %v, want nil", got)
	}
}

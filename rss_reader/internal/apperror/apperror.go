// Package apperror defines application-wide structured error primitives.
package apperror

import "errors"

// Code represents a stable application-level error code.
type Code string

const (
	// CodeNotFound indicates the requested resource does not exist.
	CodeNotFound Code = "not_found"
	// CodeInvalidArgument indicates request input is invalid.
	CodeInvalidArgument Code = "invalid_argument"
	// CodeConflict indicates a state conflict such as unique constraint violation.
	CodeConflict Code = "conflict"
	// CodeExternalUnavailable indicates an upstream dependency is unavailable.
	CodeExternalUnavailable Code = "external_unavailable"
	// CodeInternal indicates an unexpected internal failure.
	CodeInternal Code = "internal"
)

// AppError is the canonical error type propagated across layers.
type AppError struct {
	Code    Code
	Message string
	Op      string
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Op + ": " + e.Message
	}
	return e.Op + ": " + e.Message + ": " + e.Err.Error()
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// New constructs an AppError.
func New(code Code, op, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Op:      op,
		Err:     err,
	}
}

// NewNotFound constructs a not-found AppError.
func NewNotFound(op, message string, err error) *AppError {
	return New(CodeNotFound, op, message, err)
}

// NewInvalidArgument constructs an invalid-argument AppError.
func NewInvalidArgument(op, message string, err error) *AppError {
	return New(CodeInvalidArgument, op, message, err)
}

// NewConflict constructs a conflict AppError.
func NewConflict(op, message string, err error) *AppError {
	return New(CodeConflict, op, message, err)
}

// NewExternalUnavailable constructs an external-unavailable AppError.
func NewExternalUnavailable(op, message string, err error) *AppError {
	return New(CodeExternalUnavailable, op, message, err)
}

// NewInternal constructs an internal AppError.
func NewInternal(op, message string, err error) *AppError {
	return New(CodeInternal, op, message, err)
}

// Wrap keeps existing AppError metadata and prefixes operation context.
func Wrap(err error, op string) error {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		cloned := *appErr
		switch {
		case op == "":
			// keep existing op
		case cloned.Op == "":
			cloned.Op = op
		default:
			cloned.Op = op + ": " + cloned.Op
		}
		return &cloned
	}

	return NewInternal(op, "internal server error", err)
}

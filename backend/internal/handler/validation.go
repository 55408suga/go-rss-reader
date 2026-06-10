package handler

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"rss_reader/internal/apperror"
)

// init makes the validator report the wire field name (e.g. "feed_url" for a
// body field, "limit" for a query param) instead of the Go struct field, so
// error.details[].field matches exactly what the client sent. Body requests tag
// with json, list requests with query, so both are consulted.
func init() {
	requestValidator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		for _, tag := range []string{"json", "query"} {
			if name, _, _ := strings.Cut(fld.Tag.Get(tag), ","); name != "" && name != "-" {
				return name
			}
		}
		return fld.Name
	})
}

// validationDetails converts go-playground validation errors into transport-
// agnostic field violations. It returns nil for non-validation errors so the
// caller can attach details unconditionally.
func validationDetails(err error) []apperror.FieldViolation {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return nil
	}

	violations := make([]apperror.FieldViolation, 0, len(verrs))
	for _, fe := range verrs {
		violations = append(violations, apperror.FieldViolation{
			Field:  fe.Field(),
			Reason: humanizeReason(fe),
		})
	}
	return violations
}

// humanizeReason turns a single validation failure into a user-facing reason
// shown under error.details[].reason. fe.Param() carries the constraint value
// (e.g. the "100" of lte=100) for the bound-based rules.
func humanizeReason(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "url":
		return "must be a valid URL"
	case "gte":
		return "must be " + fe.Param() + " or greater"
	case "lte":
		return "must be " + fe.Param() + " or less"
	default:
		return "is invalid"
	}
}

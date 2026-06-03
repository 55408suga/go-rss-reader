package handler

import (
	"testing"

	"rss_reader/internal/apperror"
)

func TestValidationDetailsMapsJSONFieldNames(t *testing.T) {
	t.Parallel()

	// FeedURL is `required,url` with json tag "feed_url"; an empty struct fails it.
	err := requestValidator.Struct(RegisterFeedRequest{})
	if err == nil {
		t.Fatal("expected a validation error for an empty RegisterFeedRequest")
	}

	details := validationDetails(err)
	if len(details) == 0 {
		t.Fatal("validationDetails returned no violations")
	}
	// The tag-name func must surface the json name, not the Go field "FeedURL".
	if details[0].Field != "feed_url" {
		t.Errorf("field = %q, want %q", details[0].Field, "feed_url")
	}
	if details[0].Reason == "" {
		t.Error("reason is empty, want a non-empty explanation")
	}
}

func TestValidationDetailsReasons(t *testing.T) {
	t.Parallel()

	// Each struct is crafted so exactly one field fails, so details[0] is it.
	tests := []struct {
		name       string
		req        any
		wantField  string
		wantReason string
	}{
		{"required", RegisterFeedRequest{}, "feed_url", "this field is required"},
		{"url", RegisterFeedRequest{FeedURL: "not a url"}, "feed_url", "must be a valid URL"},
		{"lte", ListFeedsRequest{Limit: 101}, "limit", "must be 100 or less"},
		{"gte", ListFeedsRequest{Limit: -1}, "limit", "must be 1 or greater"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := requestValidator.Struct(tc.req)
			if err == nil {
				t.Fatalf("expected a validation error for %+v", tc.req)
			}
			details := validationDetails(err)
			if len(details) != 1 {
				t.Fatalf("details = %d, want 1: %+v", len(details), details)
			}
			if details[0].Field != tc.wantField {
				t.Errorf("field = %q, want %q", details[0].Field, tc.wantField)
			}
			if details[0].Reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", details[0].Reason, tc.wantReason)
			}
		})
	}
}

func TestValidationDetailsNonValidationErrorReturnsNil(t *testing.T) {
	t.Parallel()

	if got := validationDetails(apperror.NewInternal("op", "boom", nil)); got != nil {
		t.Errorf("validationDetails(non-validation error) = %v, want nil", got)
	}
}

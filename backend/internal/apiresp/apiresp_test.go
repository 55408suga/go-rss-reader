package apiresp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvelopeMarshalsDataAndMeta(t *testing.T) {
	t.Parallel()

	env := Envelope[map[string]int]{
		Data: map[string]int{"n": 1},
		Meta: Meta{RequestID: "req-1"},
	}
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"data":{"n":1},"meta":{"request_id":"req-1"}}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestMetaOmitsPaginationWhenNil(t *testing.T) {
	t.Parallel()

	got, _ := json.Marshal(Meta{RequestID: "r"})
	if strings.Contains(string(got), "pagination") {
		t.Errorf("pagination should be omitted, got %s", got)
	}
}

func TestPaginationNextCursorNullOnLastPage(t *testing.T) {
	t.Parallel()

	// next_cursor has no omitempty: clients can always read the key on a list
	// response, and null unambiguously means "no further page".
	got, _ := json.Marshal(Pagination{NextCursor: nil, HasMore: false})
	want := `{"next_cursor":null,"has_more":false}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestPaginationNextCursorString(t *testing.T) {
	t.Parallel()

	tok := "abc"
	got, _ := json.Marshal(Pagination{NextCursor: &tok, HasMore: true})
	want := `{"next_cursor":"abc","has_more":true}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestErrorBodyOmitsDetailsWhenEmpty(t *testing.T) {
	t.Parallel()

	got, _ := json.Marshal(ErrorBody{Code: "not_found", Message: "missing"})
	if strings.Contains(string(got), "details") {
		t.Errorf("details should be omitted, got %s", got)
	}
}

func TestErrorBodyIncludesDetails(t *testing.T) {
	t.Parallel()

	got, _ := json.Marshal(ErrorBody{
		Code:    "invalid_argument",
		Message: "validation failed",
		Details: []FieldError{{Field: "feed_url", Reason: "must be a valid URL"}},
	})
	want := `{"code":"invalid_argument","message":"validation failed",` +
		`"details":[{"field":"feed_url","reason":"must be a valid URL"}]}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

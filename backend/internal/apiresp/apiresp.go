// Package apiresp defines the HTTP response envelope shared by the API.
//
// Every /api/v1 response is one of two shapes: a success Envelope ({data,
// meta}) or an ErrorEnvelope ({error, meta}). Keeping these transport DTOs in
// one dependency-free leaf package lets both the handlers and the global error
// handler emit a single, consistent contract.
package apiresp

// Meta is the metadata block carried by both success and error responses.
// RequestID mirrors the X-Request-ID header for log correlation.
type Meta struct {
	RequestID  string      `json:"request_id"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Pagination describes keyset cursor state for list responses.
//
// NextCursor is an opaque, client-treated-as-blob token. It deliberately has
// no omitempty: on a list response the key is always present, and a null value
// unambiguously signals the last page.
type Pagination struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// Envelope is the standard success response: {data, meta}.
type Envelope[T any] struct {
	Data T    `json:"data"`
	Meta Meta `json:"meta"`
}

// ErrorEnvelope is the standard error response: {error, meta}.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}

// ErrorBody describes a failure. Code is a stable, machine-readable string
// (mirrors apperror.Code). Details is omitted unless field-level violations
// are present.
type ErrorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

// FieldError is a single field-level validation problem.
type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

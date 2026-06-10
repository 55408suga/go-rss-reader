package handler

import (
	"encoding/base64"
	"encoding/json"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

// Cursor pagination uses an opaque token at the HTTP boundary: the structured
// model.PageCursor is JSON-encoded then base64url-wrapped so clients treat it
// as a blob and pass it straight back. base64.RawURLEncoding is URL-safe and
// unpadded, so the token drops cleanly into a ?cursor= query parameter.

// encodeCursor turns a structured cursor into an opaque token. It returns nil
// for a nil cursor (e.g. the last page), so it maps directly to a nullable
// next_cursor field.
func encodeCursor(c *model.PageCursor) *string {
	if c == nil {
		return nil
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return nil // PageCursor always marshals; defensively emit no token.
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return &token
}

// decodeCursor parses an opaque token back into a structured cursor. An empty
// token means "first page" and yields (nil, nil); a malformed token is a
// client error surfaced as invalid_argument.
func decodeCursor(token string) (*model.PageCursor, error) {
	const op = "handler.decodeCursor"

	if token == "" {
		return nil, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, apperror.NewInvalidArgument(op, "invalid cursor", err)
	}

	var cursor model.PageCursor
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return nil, apperror.NewInvalidArgument(op, "invalid cursor", err)
	}
	// Syntactically valid JSON like {} still isn't a usable keyset position:
	// both components must be present for (At, ID) comparisons to make sense.
	if cursor.At.IsZero() || cursor.ID == uuid.Nil {
		return nil, apperror.NewInvalidArgument(op, "invalid cursor", nil)
	}
	return &cursor, nil
}

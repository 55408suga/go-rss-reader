package model

import (
	"time"

	"github.com/google/uuid"
)

// PageCursor identifies a position in a keyset-paginated result set.
type PageCursor struct {
	At time.Time
	ID uuid.UUID
}

// Page is a single keyset-paginated slice of results.
//
// NextCursor is the position to resume after; it is non-nil only when
// HasMore is true (i.e. another page exists). It stays structured here —
// encoding it into an opaque token is a transport-layer concern.
type Page[T any] struct {
	Items      []T
	NextCursor *PageCursor
	HasMore    bool
}

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

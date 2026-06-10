package usecase

import (
	"fmt"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

// maxListLimit mirrors the HTTP layer's lte=100 validation so the usecase
// enforces the same ceiling for callers that do not come through a handler.
const maxListLimit = 100

// validateLimit guards the list usecases against limits the paging math
// cannot handle. The HTTP handlers already validate limit (1..100, default
// 10), but the usecase is a package boundary that other callers (e.g. the
// job scheduler or future gRPC handlers) can reach directly: limit <= 0
// would break the limit+1 over-fetch (limit 0 reports has_more for a
// non-empty table; a negative limit panics on items[:limit]).
//
// The two failure messages are deliberately distinct: a non-positive limit
// is impossible as a page size (almost certainly a caller bug), while an
// over-the-ceiling limit is a plausible request that exceeds the allowed
// range.
func validateLimit(op string, limit int) error {
	if limit <= 0 {
		return apperror.NewInvalidArgument(
			op, fmt.Sprintf("limit must be a positive integer, got %d", limit), nil,
		)
	}
	if limit > maxListLimit {
		return apperror.NewInvalidArgument(
			op, fmt.Sprintf("limit %d is out of range: must be %d or less", limit, maxListLimit), nil,
		)
	}
	return nil
}

// paginate turns an over-fetched slice into a model.Page.
//
// List repositories are called with limit+1, so a result longer than limit
// proves a further page exists. We trim the extra row, set HasMore, and take
// the next cursor from the last KEPT item (cursorOf maps an item to its
// keyset position, e.g. (RegisteredAt, ID) for feeds). NextCursor is left nil
// when there is no further page.
func paginate[T any](items []T, limit int, cursorOf func(T) model.PageCursor) *model.Page[T] {
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var next *model.PageCursor
	if hasMore && len(items) > 0 {
		cursor := cursorOf(items[len(items)-1])
		next = &cursor
	}

	return &model.Page[T]{
		Items:      items,
		NextCursor: next,
		HasMore:    hasMore,
	}
}

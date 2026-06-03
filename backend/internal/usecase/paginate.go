package usecase

import "rss_reader/internal/domain/model"

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

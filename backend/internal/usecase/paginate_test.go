package usecase

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rss_reader/internal/domain/model"
)

func TestPaginate(t *testing.T) {
	t.Parallel()

	type pageItem struct {
		at time.Time
		id uuid.UUID
	}
	cursorOf := func(it pageItem) model.PageCursor {
		return model.PageCursor{At: it.at, ID: it.id}
	}

	base := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	mk := func(n int) pageItem {
		return pageItem{at: base.Add(-time.Duration(n) * time.Minute), id: uuid.New()}
	}

	t.Run("over-fetched: trims to limit, reports next cursor", func(t *testing.T) {
		t.Parallel()
		// limit 3, repo returned limit+1 (4) -> a next page exists.
		items := []pageItem{mk(0), mk(1), mk(2), mk(3)}
		page := paginate(items, 3, cursorOf)

		if len(page.Items) != 3 {
			t.Fatalf("Items length = %d, want 3 (trimmed)", len(page.Items))
		}
		if !page.HasMore {
			t.Error("HasMore = false, want true")
		}
		if page.NextCursor == nil {
			t.Fatal("NextCursor = nil, want the 3rd item's cursor")
		}
		want := cursorOf(items[2]) // last KEPT item, not the over-fetched 4th
		if *page.NextCursor != want {
			t.Errorf("NextCursor = %v, want %v", *page.NextCursor, want)
		}
	})

	t.Run("exactly limit: no next page", func(t *testing.T) {
		t.Parallel()
		items := []pageItem{mk(0), mk(1), mk(2)}
		page := paginate(items, 3, cursorOf)

		if len(page.Items) != 3 {
			t.Fatalf("Items length = %d, want 3", len(page.Items))
		}
		if page.HasMore {
			t.Error("HasMore = true, want false")
		}
		if page.NextCursor != nil {
			t.Errorf("NextCursor = %v, want nil", *page.NextCursor)
		}
	})

	t.Run("fewer than limit: no next page", func(t *testing.T) {
		t.Parallel()
		items := []pageItem{mk(0)}
		page := paginate(items, 3, cursorOf)

		if len(page.Items) != 1 || page.HasMore || page.NextCursor != nil {
			t.Errorf("got Items=%d HasMore=%v NextCursor=%v, want 1/false/nil",
				len(page.Items), page.HasMore, page.NextCursor)
		}
	})

	t.Run("empty: no next page", func(t *testing.T) {
		t.Parallel()
		page := paginate([]pageItem{}, 3, cursorOf)

		if len(page.Items) != 0 || page.HasMore || page.NextCursor != nil {
			t.Errorf("got Items=%d HasMore=%v NextCursor=%v, want 0/false/nil",
				len(page.Items), page.HasMore, page.NextCursor)
		}
	})
}

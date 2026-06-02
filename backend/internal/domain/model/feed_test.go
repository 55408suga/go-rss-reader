package model

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

func TestNewFeed(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name        string
		title       string
		feedURL     string
		websiteURL  string
		description string
		language    string
		updatedAt   time.Time
		want        *Feed // every field except the generated ID / RegisteredAt
	}{
		{
			name:        "typical feed",
			title:       "Example Feed",
			feedURL:     "https://example.com/feed.xml",
			websiteURL:  "https://example.com",
			description: "An example feed",
			language:    "en",
			updatedAt:   updatedAt,
			want: &Feed{
				Title:       "Example Feed",
				FeedURL:     "https://example.com/feed.xml",
				WebsiteURL:  "https://example.com",
				Description: "An example feed",
				Language:    "en",
				UpdatedAt:   updatedAt,
			},
		},
		{
			name:      "empty optional fields",
			feedURL:   "https://example.com/feed.xml",
			updatedAt: updatedAt,
			want: &Feed{
				FeedURL:   "https://example.com/feed.xml",
				UpdatedAt: updatedAt,
			},
		},
		{
			name:        "unicode metadata",
			title:       "日本語フィード",
			feedURL:     "https://example.jp/feed.xml",
			description: "説明テキスト",
			language:    "ja",
			updatedAt:   updatedAt,
			want: &Feed{
				Title:       "日本語フィード",
				FeedURL:     "https://example.jp/feed.xml",
				Description: "説明テキスト",
				Language:    "ja",
				UpdatedAt:   updatedAt,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			before := time.Now().UTC()
			got, err := NewFeed(tc.title, tc.feedURL, tc.websiteURL, tc.description, tc.language, tc.updatedAt)
			after := time.Now().UTC()
			if err != nil {
				t.Fatalf("NewFeed() unexpected error: %v", err)
			}

			assertGeneratedUUIDv7(t, got.ID)
			assertWithinUTCWindow(t, "RegisteredAt", got.RegisteredAt, before, after)

			// Compare caller-supplied fields exactly; generated fields are checked above.
			ignore := cmpopts.IgnoreFields(Feed{}, "ID", "RegisteredAt")
			if diff := cmp.Diff(tc.want, got, ignore); diff != "" {
				t.Errorf("NewFeed() field mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewFeedGeneratesUniqueIDs(t *testing.T) {
	t.Parallel()

	const n = 100
	seen := make(map[uuid.UUID]struct{}, n)
	for range n {
		feed, err := NewFeed("t", "https://example.com/feed.xml", "w", "d", "en", time.Now())
		if err != nil {
			t.Fatalf("NewFeed() unexpected error: %v", err)
		}
		if _, dup := seen[feed.ID]; dup {
			t.Fatalf("duplicate ID generated: %s", feed.ID)
		}
		seen[feed.ID] = struct{}{}
	}
}

package model

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

func TestNewArticle(t *testing.T) {
	t.Parallel()

	feedID := uuid.New()
	publishedAt := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	tests := []struct {
		name        string
		title       string
		description string
		content     string
		websiteURL  string
		publishedAt time.Time
		feedID      uuid.UUID
		externalID  string
		want        *Article // every field except the generated ID
	}{
		{
			name:        "typical article with guid external id",
			title:       "Hello World",
			description: "A short description",
			content:     "<p>body</p>",
			websiteURL:  "https://example.com/posts/1",
			publishedAt: publishedAt,
			feedID:      feedID,
			externalID:  "guid-123",
			want: &Article{
				Title:       "Hello World",
				Description: "A short description",
				Content:     "<p>body</p>",
				WebsiteURL:  "https://example.com/posts/1",
				PublishedAt: publishedAt,
				FeedID:      feedID,
				ExternalID:  "guid-123",
			},
		},
		{
			name:        "minimal article with hashed external id",
			websiteURL:  "https://example.com/posts/2",
			publishedAt: publishedAt,
			feedID:      feedID,
			externalID:  "0f9e8d7c6b5a",
			want: &Article{
				WebsiteURL:  "https://example.com/posts/2",
				PublishedAt: publishedAt,
				FeedID:      feedID,
				ExternalID:  "0f9e8d7c6b5a",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewArticle(
				tc.title, tc.description, tc.content, tc.websiteURL,
				tc.publishedAt, tc.feedID, tc.externalID,
			)
			if err != nil {
				t.Fatalf("NewArticle() unexpected error: %v", err)
			}

			assertGeneratedUUIDv7(t, got.ID)

			// FeedID and ExternalID are caller-supplied; comparing them guards the
			// idempotency key the persistence layer relies on (UNIQUE(feed_id, external_id)).
			ignore := cmpopts.IgnoreFields(Article{}, "ID")
			if diff := cmp.Diff(tc.want, got, ignore); diff != "" {
				t.Errorf("NewArticle() field mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

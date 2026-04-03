// Package gateway provides implementations of gateways for fetching RSS feed data.
package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"rss_reader/internal/domain/model"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
)

type RSSGateway struct {
	parser *gofeed.Parser
}

func NewRSSGateway() *RSSGateway {
	return &RSSGateway{
		parser: gofeed.NewParser(),
	}
}

// FetchFeedWithArticles fetches and parses an RSS feed URL, returning both
// the feed metadata and all articles in a single HTTP request.
// Article FeedIDs are set to uuid.Nil; callers must set them after feed is persisted.
func (rg *RSSGateway) FetchFeedWithArticles(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, error) {
	feedData, err := rg.parser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, nil, err
	}

	// Build model.Feed
	updatedAt := time.Now().UTC()
	if feedData.UpdatedParsed != nil {
		updatedAt = feedData.UpdatedParsed.UTC()
	}
	feed, err := model.NewFeed(
		feedData.Title,
		feedURL,
		feedData.Link,
		feedData.Description,
		feedData.Language,
		updatedAt,
	)
	if err != nil {
		return nil, nil, err
	}

	// Build model.Article list
	articles := make([]*model.Article, 0, len(feedData.Items))
	for _, item := range feedData.Items {
		publishedAt := time.Now().UTC()
		if item.PublishedParsed != nil {
			publishedAt = item.PublishedParsed.UTC()
		} else if item.UpdatedParsed != nil {
			publishedAt = item.UpdatedParsed.UTC()
		}

		externalID := resolveExternalID(item, publishedAt)

		article, err := model.NewArticle(
			item.Title,
			item.Description,
			item.Content,
			item.Link,
			publishedAt,
			uuid.Nil, // FeedID will be set by the caller after feed is saved
			externalID,
		)
		if err != nil {
			return nil, nil, err
		}
		articles = append(articles, article)
	}

	return feed, articles, nil
}

func resolveExternalID(item *gofeed.Item, publishedAt time.Time) string {
	if guid := strings.TrimSpace(item.GUID); guid != "" {
		return guid
	}

	if link := strings.TrimSpace(item.Link); link != "" {
		return link
	}

	fallbackPublishedAt := publishedAt.UTC()
	if item.PublishedParsed == nil && item.UpdatedParsed == nil {
		fallbackPublishedAt = time.Time{}.UTC()
	}

	seed := strings.TrimSpace(item.Title) + "|" + fallbackPublishedAt.Format(time.RFC3339Nano)
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}

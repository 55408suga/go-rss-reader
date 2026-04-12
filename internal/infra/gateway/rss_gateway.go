// Package gateway provides implementations of gateways for fetching RSS feed data.
package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"rss_reader/internal/domain/model"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
)

type RSSGateway struct {
	parser     *gofeed.Parser
	httpClient http.Client
}

func NewRSSGateway() *RSSGateway {
	return &RSSGateway{
		parser:     gofeed.NewParser(),
		httpClient: *NewHTTPClient(),
	}
}

// FetchFeedWithArticles fetches and parses an RSS feed URL, returning both
// the feed metadata and all articles in a single HTTP request.
// Article FeedIDs are set to uuid.Nil; callers must set them after feed is persisted.
func (rg *RSSGateway) FetchFeedWithArticles(ctx context.Context, feedURL string) (*model.Feed, []*model.Article, *model.FeedCursor, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	resp, err := rg.httpClient.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()
	// 後で詳細に分岐を詰める
	if resp.StatusCode != 200 {
		return nil, nil, nil, err
	}
	feedData, err := rg.parser.Parse(resp.Body)
	if err != nil {
		return nil, nil, nil, err
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
		return nil, nil, nil, err
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
			return nil, nil, nil, err
		}
		articles = append(articles, article)
	}

	// build fetch status
	etag := toOptionalString(strings.TrimSpace(resp.Header.Get("ETag")))
	lastModified := parseHTTPTime(strings.TrimSpace(resp.Header.Get("Last-Modified")))
	feedCursor := &model.FeedCursor{
		ETag:         etag,
		LastModified: lastModified,
	}
	return feed, articles, feedCursor, nil
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

// 以下util後でまとめを検討
func toOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func parseHTTPTime(value string) *time.Time {
	if value == "" {
		return nil
	}

	parsed, err := http.ParseTime(value)
	if err != nil {
		return nil
	}

	utc := parsed.UTC()
	return &utc
}

// Package gateway provides implementations of gateways for fetching RSS feed data.
package gateway

import (
	"context"
	"rss_reader/internal/domain/model"
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

func (rg *RSSGateway) FetchFeed(ctx context.Context, feedURL string) (*model.Feed, error) {
	feedData, err := rg.parser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, err
	}

	updatedAt := time.Now().UTC()
	if feedData.UpdatedParsed != nil {
		updatedAt = feedData.UpdatedParsed.UTC()
	}
	// Convert gofeed.Feed to model.Feed
	feed, err := model.NewFeed(
		feedData.Title,
		feedURL,
		feedData.Link,
		feedData.Description,
		feedData.Language,
		updatedAt,
	)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

func (rg *RSSGateway) FetchArticles(ctx context.Context, feedID uuid.UUID, feedURL string) ([]*model.Article, error) {
	feedData, err := rg.parser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, err
	}
	articles := make([]*model.Article, 0, len(feedData.Items))
	// convert gofeed.Item to model.Article
	for _, item := range feedData.Items {
		publishedAt := time.Now().UTC()
		if item.PublishedParsed != nil {
			publishedAt = item.PublishedParsed.UTC()
		}

		article, err := model.NewArticle(
			item.Title,
			item.Description,
			item.Content,
			item.Link,
			publishedAt,
			feedID,
		)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	return articles, nil
}

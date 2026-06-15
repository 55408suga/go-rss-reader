package model

// FeedCandidate is a feed URL discovered from a website's HTML head via RSS
// autodiscovery (<link rel="alternate" type="application/rss+xml" href=...>).
// Like FeedCursor it is a value object: no identity, never persisted.
type FeedCandidate struct {
	FeedURL  string `json:"feed_url"`
	Title    string `json:"title"`     // <link title="..."> if present
	MIMEType string `json:"mime_type"` // e.g. application/rss+xml
}

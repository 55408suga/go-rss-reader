package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"rss_reader/internal/apperror"
	applogger "rss_reader/internal/applog"
	"rss_reader/internal/domain/model"
)

// maxDiscoveryHTMLBytes caps how much of the page is read while scanning for
// autodiscovery links. Feed links live in <head>, so 1MiB is generous; the
// cap bounds memory/time on hostile or broken pages (SSRF hardening).
const maxDiscoveryHTMLBytes = 1 << 20

// DiscoveryGateway discovers feed URLs from a website's HTML head following
// the RSS Board feed-autodiscovery convention. It implements
// usecase.FeedDiscoverer. Distinct from RSSGateway on purpose: this type
// parses HTML pages, RSSGateway parses feed XML.
type DiscoveryGateway struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewDiscoveryGateway constructs a gateway for discovering feed URLs.
func NewDiscoveryGateway(httpClient *http.Client, logger *slog.Logger) *DiscoveryGateway {
	if httpClient == nil {
		httpClient = NewHTTPClient()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &DiscoveryGateway{
		httpClient: httpClient,
		logger:     logger,
	}
}

// DiscoverFeedURLs fetches websiteURL and returns the feed candidates
// declared in its HTML head, in document order (the page's own priority).
// Classification: unreachable/non-2xx -> external_unavailable; non-HTML
// response or zero candidates -> not_found.
func (dg *DiscoveryGateway) DiscoverFeedURLs(
	ctx context.Context,
	websiteURL string,
) ([]model.FeedCandidate, error) {
	const op = "DiscoveryGateway.DiscoverFeedURLs"
	startedAt := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, websiteURL, http.NoBody)
	if err != nil {
		return nil, apperror.NewInvalidArgument(op, "invalid website url", err)
	}
	req.Header.Set("User-Agent", "Go RSS Reader/1.0")

	resp, err := dg.httpClient.Do(req)
	if err != nil {
		applogger.WithContext(ctx, dg.logger).WarnContext(ctx,
			"website fetch failed",
			"website_url", websiteURL,
			"error", err,
		)
		return nil, apperror.NewExternalUnavailable(op, "failed to fetch website", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			applogger.WithContext(ctx, dg.logger).WarnContext(ctx,
				"failed to close website response body",
				"website_url", websiteURL,
				"error", closeErr,
			)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		applogger.WithContext(ctx, dg.logger).WarnContext(ctx,
			"website fetch returned non-success status",
			"website_url", websiteURL,
			"status", resp.StatusCode,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
		return nil, apperror.NewExternalUnavailable(
			op,
			fmt.Sprintf("website returned status %d", resp.StatusCode),
			nil,
		)
	}

	if !isHTMLContentType(resp.Header.Get("Content-Type")) {
		// A non-HTML document cannot carry autodiscovery links. (Treating a
		// direct feed URL as subscribable here is a Phase 2 extension.)
		return nil, apperror.NewNotFound(op, "website did not return an html document", nil)
	}

	// Relative hrefs resolve against the final URL after redirects, not the
	// URL the caller passed in.
	baseURL := resp.Request.URL

	limitedBody := io.LimitReader(resp.Body, maxDiscoveryHTMLBytes)
	candidates, scanErr := scanFeedLinks(limitedBody, baseURL)
	if scanErr != nil {
		// A truncated/failed read is an external problem, not "no feed here".
		applogger.WithContext(ctx, dg.logger).WarnContext(ctx,
			"website html read failed during scan",
			"website_url", websiteURL,
			"error", scanErr,
		)
		return nil, apperror.NewExternalUnavailable(op, "failed to read website html", scanErr)
	}
	// Drain the (limit-capped) remainder so the shared transport can return
	// the connection to its keep-alive pool; the scanner stops at </head>.
	_, _ = io.Copy(io.Discard, limitedBody)

	if len(candidates) == 0 {
		return nil, apperror.NewNotFound(op, "no rss/atom feed found at this website", nil)
	}

	applogger.WithContext(ctx, dg.logger).InfoContext(ctx,
		"feed autodiscovery succeeded",
		"website_url", websiteURL,
		"candidates", len(candidates),
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return candidates, nil
}

// isHTMLContentType reports whether the response Content-Type can carry
// autodiscovery <link> elements.
func isHTMLContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "text/html" || mediaType == "application/xhtml+xml"
}

// scanFeedLinks tokenizes htmlBody and collects feed candidates from <link>
// elements, stopping at </head>, <body>, or end of (possibly truncated)
// input. The tokenizer never builds a DOM, so a huge page costs only the
// bytes actually read.
func scanFeedLinks(htmlBody io.Reader, baseURL *url.URL) ([]model.FeedCandidate, error) {
	tokenizer := html.NewTokenizer(htmlBody)
	var candidates []model.FeedCandidate

	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			// The tokenizer stops here for reasons that must NOT be treated
			// alike: a clean EOF and the 1MiB LimitReader cap (which also
			// surfaces as io.EOF) mean "we read all the head we were given",
			// but any other error is a truncated/failed read that the caller
			// must classify as external_unavailable instead of not_found.
			if err := tokenizer.Err(); err != nil && !errors.Is(err, io.EOF) {
				return candidates, err
			}
			return candidates, nil
		case html.EndTagToken:
			if name, _ := tokenizer.TagName(); string(name) == "head" {
				return candidates, nil
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := tokenizer.TagName()
			if string(name) == "body" {
				return candidates, nil
			}
			if string(name) != "link" || !hasAttr {
				continue
			}
			if candidate, ok := feedCandidateFromLink(tokenizer, baseURL); ok {
				candidates = append(candidates, candidate)
			}
		default:
			// text/comment/doctype tokens carry no <link> elements; skip.
		}
	}
}

// feedCandidateFromLink reads the current <link> tag's attributes and builds
// a candidate when the element is a feed autodiscovery link.
func feedCandidateFromLink(
	tokenizer *html.Tokenizer,
	baseURL *url.URL,
) (model.FeedCandidate, bool) {
	var rel, typ, href, title string
	for {
		key, value, more := tokenizer.TagAttr()
		switch string(key) { // attribute keys are lowercased by the tokenizer
		case "rel":
			rel = string(value)
		case "type":
			typ = string(value)
		case "href":
			href = string(value)
		case "title":
			title = string(value)
		}
		if !more {
			break
		}
	}

	mimeType, ok := feedMIMEType(rel, typ)
	if !ok {
		return model.FeedCandidate{}, false
	}

	href = strings.TrimSpace(href)
	if href == "" {
		return model.FeedCandidate{}, false
	}
	hrefURL, err := url.Parse(href)
	if err != nil {
		return model.FeedCandidate{}, false
	}
	resolved := baseURL.ResolveReference(hrefURL)
	// The href comes from the fetched page, i.e. it is attacker-controlled.
	// Candidates feed straight into a second server-side fetch (RegisterFeed),
	// so anything outside http/https is dropped here, not at Do() time.
	// (resolved.Scheme is already lowercased: url.Parse normalizes it, so an
	// uppercase "HTTPS" href compares equal here without EqualFold.)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return model.FeedCandidate{}, false
	}

	return model.FeedCandidate{
		FeedURL:  resolved.String(),
		Title:    title,
		MIMEType: mimeType,
	}, true
}

// feedMIMEType decides whether a <link> element's raw rel/typ attribute
// values declare a feed autodiscovery link, returning the canonical
// (lowercased, trimmed) MIME type and true when they do.
//
// rel is a space-separated, ASCII case-insensitive token list (e.g.
// "alternate", "ALTERNATE", "alternate stylesheet"). typ is the raw type
// attribute value. Feed types are application/rss+xml, application/atom+xml,
// and application/feed+json.
func feedMIMEType(rel, typ string) (string, bool) {
	hasAlternate := false
	for token := range strings.FieldsSeq(rel) {
		switch {
		case strings.EqualFold(token, "stylesheet"):
			// rel="alternate stylesheet" declares an alternate stylesheet,
			// not a feed.
			return "", false
		case strings.EqualFold(token, "alternate"):
			hasAlternate = true
		}
	}
	if !hasAlternate {
		return "", false
	}

	mimeType := strings.ToLower(strings.TrimSpace(typ))
	switch mimeType {
	case "application/rss+xml", "application/atom+xml", "application/feed+json":
		return mimeType, true
	}
	return "", false
}

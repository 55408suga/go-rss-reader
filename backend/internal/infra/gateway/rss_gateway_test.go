package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

const rssXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Example Feed</title>
    <link>https://example.com</link>
    <description>An example feed</description>
    <language>en</language>
    <item>
      <title>First Post</title>
      <link>https://example.com/posts/1</link>
      <description>first</description>
      <guid>guid-1</guid>
      <pubDate>Tue, 10 Mar 2026 12:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Second Post</title>
      <link>https://example.com/posts/2</link>
      <description>second</description>
      <guid>guid-2</guid>
      <pubDate>Wed, 11 Mar 2026 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func strptr(s string) *string { return &s }

func assertAppErrorCode(t *testing.T, err error, want apperror.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", want)
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T: %v", err, err)
	}
	if appErr.Code != want {
		t.Errorf("error code = %q, want %q", appErr.Code, want)
	}
}

func TestResolveExternalID(t *testing.T) {
	t.Parallel()

	published := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	hash := func(title string, ts time.Time) string {
		seed := title + "|" + ts.Format(time.RFC3339Nano)
		sum := sha256.Sum256([]byte(seed))
		return hex.EncodeToString(sum[:])
	}

	tests := []struct {
		name        string
		item        *gofeed.Item
		publishedAt time.Time
		want        string
	}{
		{
			name:        "prefers guid over everything",
			item:        &gofeed.Item{GUID: "guid-1", Link: "https://e/1", Title: "t"},
			publishedAt: published,
			want:        "guid-1",
		},
		{
			name:        "trims whitespace around guid",
			item:        &gofeed.Item{GUID: "  guid-2  ", Title: "t"},
			publishedAt: published,
			want:        "guid-2",
		},
		{
			name:        "falls back to link when guid is empty",
			item:        &gofeed.Item{Link: "https://e/3", Title: "t"},
			publishedAt: published,
			want:        "https://e/3",
		},
		{
			name:        "hashes title and published time when a parsed time exists",
			item:        &gofeed.Item{Title: "Hello", PublishedParsed: &published},
			publishedAt: published,
			want:        hash("Hello", published),
		},
		{
			name:        "hashes title and the zero time when no parsed times exist",
			item:        &gofeed.Item{Title: "Hello"},
			publishedAt: published,
			want:        hash("Hello", time.Time{}.UTC()),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveExternalID(tc.item, tc.publishedAt); got != tc.want {
				t.Errorf("resolveExternalID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestToOptionalString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  *string
	}{
		{name: "empty becomes nil", value: "", want: nil},
		{name: "non-empty becomes a pointer", value: "etag", want: strptr("etag")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := toOptionalString(tc.value)
			switch {
			case tc.want == nil:
				if got != nil {
					t.Errorf("got %q, want nil", *got)
				}
			case got == nil:
				t.Errorf("got nil, want %q", *tc.want)
			case *got != *tc.want:
				t.Errorf("got %q, want %q", *got, *tc.want)
			}
		})
	}
}

func TestParseHTTPTime(t *testing.T) {
	t.Parallel()

	known := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value string
		want  *time.Time
	}{
		{name: "empty is nil", value: "", want: nil},
		{name: "invalid is nil", value: "not a date", want: nil},
		{name: "valid http date parses to UTC", value: known.Format(http.TimeFormat), want: &known},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseHTTPTime(tc.value)
			switch {
			case tc.want == nil:
				if got != nil {
					t.Errorf("parseHTTPTime(%q) = %v, want nil", tc.value, *got)
				}
			case got == nil:
				t.Errorf("parseHTTPTime(%q) = nil, want %v", tc.value, *tc.want)
			case !got.Equal(*tc.want):
				t.Errorf("parseHTTPTime(%q) = %v, want %v", tc.value, *got, *tc.want)
			}
		})
	}
}

func TestFetchFeedWithCursorSuccess(t *testing.T) {
	t.Parallel()

	lastMod := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Last-Modified", lastMod.Format(http.TimeFormat))
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(rssXML))
	}))
	defer srv.Close()

	gw := NewRSSGateway(srv.Client(), quietLogger())
	feed, articles, cursor, err := gw.FetchNewFeed(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if feed.Title != "Example Feed" {
		t.Errorf("feed.Title = %q, want %q", feed.Title, "Example Feed")
	}
	if feed.WebsiteURL != "https://example.com" {
		t.Errorf("feed.WebsiteURL = %q, want %q", feed.WebsiteURL, "https://example.com")
	}
	if feed.FeedURL != srv.URL {
		t.Errorf("feed.FeedURL = %q, want %q", feed.FeedURL, srv.URL)
	}
	if feed.Language != "en" {
		t.Errorf("feed.Language = %q, want %q", feed.Language, "en")
	}

	if len(articles) != 2 {
		t.Fatalf("len(articles) = %d, want 2", len(articles))
	}
	if articles[0].ExternalID != "guid-1" {
		t.Errorf("articles[0].ExternalID = %q, want guid-1", articles[0].ExternalID)
	}
	if articles[0].WebsiteURL != "https://example.com/posts/1" {
		t.Errorf("articles[0].WebsiteURL = %q, want the item link", articles[0].WebsiteURL)
	}
	for i, a := range articles {
		if a.FeedID != feed.ID {
			t.Errorf("articles[%d].FeedID = %s, want feed ID %s", i, a.FeedID, feed.ID)
		}
	}

	if cursor.ETag == nil || *cursor.ETag != `"abc123"` {
		t.Errorf("cursor.ETag = %v, want %q", cursor.ETag, `"abc123"`)
	}
	if cursor.LastModified == nil || !cursor.LastModified.Equal(lastMod) {
		t.Errorf("cursor.LastModified = %v, want %v", cursor.LastModified, lastMod)
	}
}

func TestFetchFeedWithCursorNotModified(t *testing.T) {
	t.Parallel()

	var gotINM, gotIMS string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotINM = r.Header.Get("If-None-Match")
		gotIMS = r.Header.Get("If-Modified-Since")
		w.WriteHeader(http.StatusNotModified)
	}))
	defer srv.Close()

	lastMod := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	etag := `"abc123"`
	inCursor := &model.FeedCursor{ETag: &etag, LastModified: &lastMod}

	gw := NewRSSGateway(srv.Client(), quietLogger())
	feed, articles, cursor, err := gw.FetchFeedWithCursor(context.Background(), srv.URL, inCursor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if feed != nil || articles != nil {
		t.Errorf("expected nil feed/articles on 304, got feed=%v articles=%v", feed, articles)
	}
	if cursor != inCursor {
		t.Error("expected the input cursor to be returned unchanged on 304")
	}
	if gotINM != etag {
		t.Errorf("If-None-Match = %q, want %q", gotINM, etag)
	}
	if want := lastMod.Format(http.TimeFormat); gotIMS != want {
		t.Errorf("If-Modified-Since = %q, want %q", gotIMS, want)
	}
}

func TestFetchFeedWithCursorErrors(t *testing.T) {
	t.Parallel()

	t.Run("non-2xx status is external unavailable", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		gw := NewRSSGateway(srv.Client(), quietLogger())
		_, _, _, err := gw.FetchNewFeed(context.Background(), srv.URL)
		assertAppErrorCode(t, err, apperror.CodeExternalUnavailable)
	})

	t.Run("unparseable body is external unavailable", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("this is definitely not rss"))
		}))
		defer srv.Close()
		gw := NewRSSGateway(srv.Client(), quietLogger())
		_, _, _, err := gw.FetchNewFeed(context.Background(), srv.URL)
		assertAppErrorCode(t, err, apperror.CodeExternalUnavailable)
	})

	t.Run("malformed url is invalid argument", func(t *testing.T) {
		t.Parallel()
		gw := NewRSSGateway(nil, quietLogger())
		_, _, _, err := gw.FetchNewFeed(context.Background(), "://bad-url")
		assertAppErrorCode(t, err, apperror.CodeInvalidArgument)
	})

	t.Run("connection failure is external unavailable", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := srv.URL
		srv.Close() // close so the next request is refused
		gw := NewRSSGateway(nil, quietLogger())
		_, _, _, err := gw.FetchNewFeed(context.Background(), url)
		assertAppErrorCode(t, err, apperror.CodeExternalUnavailable)
	})
}

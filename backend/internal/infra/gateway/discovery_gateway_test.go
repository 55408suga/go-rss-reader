package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

func TestDiscoveryGatewayDiscoverFeedURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		reqPath     string // path requested on the test server ("/" if empty)
		html        string
		contentType string // "text/html; charset=utf-8" if empty
		status      int    // 200 if zero
		// want candidates; "{base}" in FeedURL is replaced with the test
		// server URL because httptest binds an ephemeral port.
		want     []model.FeedCandidate
		wantCode apperror.Code
	}{
		{
			name: "detects rss atom and json feed links in document order",
			html: `<!DOCTYPE html><html><head>
<link rel="alternate" type="application/rss+xml" title="Example RSS" href="/feed.xml">
<link rel="alternate" type="application/atom+xml" title="Example Atom" href="/atom.xml">
<link rel="alternate" type="application/feed+json" href="/feed.json">
</head><body></body></html>`,
			want: []model.FeedCandidate{
				{FeedURL: "{base}/feed.xml", Title: "Example RSS", MIMEType: "application/rss+xml"},
				{FeedURL: "{base}/atom.xml", Title: "Example Atom", MIMEType: "application/atom+xml"},
				{FeedURL: "{base}/feed.json", MIMEType: "application/feed+json"},
			},
		},
		{
			name:    "resolves relative href against the page URL",
			reqPath: "/blog/",
			html: `<html><head>
<link rel="alternate" type="application/rss+xml" href="feed.xml">
</head><body></body></html>`,
			want: []model.FeedCandidate{
				{FeedURL: "{base}/blog/feed.xml", MIMEType: "application/rss+xml"},
			},
		},
		{
			name: "excludes alternate stylesheet links",
			html: `<html><head>
<link rel="alternate stylesheet" type="application/rss+xml" href="/not-a-feed.xml">
<link rel="alternate" type="application/rss+xml" href="/real.xml">
</head><body></body></html>`,
			want: []model.FeedCandidate{
				{FeedURL: "{base}/real.xml", MIMEType: "application/rss+xml"},
			},
		},
		{
			name: "matches rel and type case-insensitively",
			html: `<html><head>
<LINK REL="ALTERNATE" TYPE="APPLICATION/RSS+XML" href="/up.xml">
</head><body></body></html>`,
			want: []model.FeedCandidate{
				{FeedURL: "{base}/up.xml", MIMEType: "application/rss+xml"},
			},
		},
		{
			name:     "no feed link yields not_found",
			html:     `<html><head><title>plain</title></head><body></body></html>`,
			wantCode: apperror.CodeNotFound,
		},
		{
			name:        "non-html content type yields not_found",
			html:        `{"not": "html"}`,
			contentType: "application/json",
			wantCode:    apperror.CodeNotFound,
		},
		{
			name:     "non-2xx status yields external_unavailable",
			html:     "oops",
			status:   http.StatusInternalServerError,
			wantCode: apperror.CodeExternalUnavailable,
		},
		{
			name: "ignores links outside head",
			html: `<html><head><title>t</title></head><body>
<link rel="alternate" type="application/rss+xml" href="/late.xml">
</body></html>`,
			wantCode: apperror.CodeNotFound,
		},
		{
			name: "stops reading after 1MiB",
			html: "<html><head><!--" + strings.Repeat("x", 1<<20) + `-->
<link rel="alternate" type="application/rss+xml" href="/beyond-limit.xml">
</head><body></body></html>`,
			wantCode: apperror.CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				contentType := tt.contentType
				if contentType == "" {
					contentType = "text/html; charset=utf-8"
				}
				w.Header().Set("Content-Type", contentType)
				status := tt.status
				if status == 0 {
					status = http.StatusOK
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte(tt.html))
			}))
			defer server.Close()

			gateway := NewDiscoveryGateway(server.Client(), quietLogger())
			reqPath := tt.reqPath
			if reqPath == "" {
				reqPath = "/"
			}
			got, err := gateway.DiscoverFeedURLs(context.Background(), server.URL+reqPath)

			if tt.wantCode != "" {
				assertAppErrorCode(t, err, tt.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("DiscoverFeedURLs: %v", err)
			}
			want := make([]model.FeedCandidate, len(tt.want))
			for i, candidate := range tt.want {
				candidate.FeedURL = strings.Replace(candidate.FeedURL, "{base}", server.URL, 1)
				want[i] = candidate
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("candidates mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// Relative hrefs must resolve against the final URL after redirects, not the
// URL the user typed.
func TestDiscoveryGatewayResolvesAgainstRedirectedURL(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/blog/", http.StatusFound)
	})
	mux.HandleFunc("/blog/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head>
<link rel="alternate" type="application/rss+xml" href="feed.xml">
</head><body></body></html>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	gateway := NewDiscoveryGateway(server.Client(), quietLogger())
	got, err := gateway.DiscoverFeedURLs(context.Background(), server.URL+"/start")
	if err != nil {
		t.Fatalf("DiscoverFeedURLs: %v", err)
	}
	want := []model.FeedCandidate{
		{FeedURL: server.URL + "/blog/feed.xml", MIMEType: "application/rss+xml"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("candidates mismatch (-want +got):\n%s", diff)
	}
}

func TestDiscoveryGatewayUnreachableServer(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close()

	gateway := NewDiscoveryGateway(nil, quietLogger())
	_, err := gateway.DiscoverFeedURLs(context.Background(), url)
	assertAppErrorCode(t, err, apperror.CodeExternalUnavailable)
}

func TestDiscoveryGatewayInvalidURL(t *testing.T) {
	t.Parallel()

	gateway := NewDiscoveryGateway(nil, quietLogger())
	_, err := gateway.DiscoverFeedURLs(context.Background(), "://missing-scheme")
	assertAppErrorCode(t, err, apperror.CodeInvalidArgument)
}

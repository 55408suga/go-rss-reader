package gateway

import (
	"net/http"
	"time"
)

// NewHTTPClient creates an HTTP client tuned for RSS fetch workloads.
//
// TODO(phase-2): this client fetches user-supplied URLs (POST /feeds and
// POST /feeds/discover) with no dial restrictions, so loopback/private-range
// targets (127.0.0.1, 10/8, 172.16/12, 192.168/16, 169.254/16, ::1) are
// reachable. Acceptable while the app is single-user on localhost; add a
// net.Dialer.Control blocklist here before any multi-user or cloud deploy.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

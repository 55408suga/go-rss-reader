package gateway

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// NewHTTPClient creates an HTTP client tuned for RSS fetch workloads.
//
// This client fetches user-supplied URLs (POST /feeds and POST /feeds/discover),
// which makes it an SSRF surface. The dialer's Control hook (blockInternalDial)
// runs after DNS resolution and before the TCP connect on every dial — including
// the dials triggered by HTTP redirects — so loopback/private/link-local targets
// (127.0.0.1, 10/8, 172.16/12, 192.168/16, 169.254/16, ::1, fc00::/7, fe80::/10)
// can never be reached, regardless of how the hostname resolves.
func NewHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   blockInternalDial,
	}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// blockInternalDial is the net.Dialer.Control hook used as the SSRF gate.
//
// It fires after the destination has been resolved to a concrete IP and just
// before the socket connects, so address is the real "host:port" we are about
// to reach (not the original hostname). Validating here — rather than parsing
// the request URL up front — defeats DNS rebinding: the IP we check is exactly
// the IP we connect to. Returning a non-nil error aborts the dial.
func blockInternalDial(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("ssrf guard: cannot parse dial address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		// Control always receives a resolved IP literal, never a hostname, so
		// a parse failure means something unexpected — fail closed.
		return fmt.Errorf("ssrf guard: dial address %q is not an IP literal", host)
	}
	if isBlockedIP(ip) {
		return fmt.Errorf("ssrf guard: refusing to connect to internal address %s", ip)
	}
	return nil
}

// isBlockedIP reports whether ip points at an internal or otherwise
// non-routable destination that a user-supplied URL must never reach.
//
// It fails closed: anything loopback (127.0.0.0/8, ::1), private
// (10/8, 172.16/12, 192.168/16, fc00::/7), link-local (169.254/16 incl. the
// 169.254.169.254 cloud-metadata endpoint, fe80::/10), or unspecified
// (0.0.0.0, :: — which can route back to loopback) is blocked. Ordinary public
// unicast addresses pass.
func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return true
	}
	return false
}

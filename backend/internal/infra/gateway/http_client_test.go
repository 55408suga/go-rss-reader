package gateway

import (
	"net"
	"testing"
)

// TestIsBlockedIP is the spec for the SSRF dial gate: user-supplied URLs must
// never let the server reach loopback, private, link-local, or unspecified
// addresses, while ordinary public addresses stay reachable.
func TestIsBlockedIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		// Blocked: loopback (server's own admin services).
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv4 loopback non-1 octet", "127.0.0.5", true},
		{"IPv6 loopback", "::1", true},

		// Blocked: private ranges (internal network recon).
		{"private 10/8", "10.0.0.5", true},
		{"private 172.16/12", "172.16.0.1", true},
		{"private 192.168/16", "192.168.1.1", true},
		{"IPv6 unique local fc00::/7", "fc00::1", true},
		{"IPv6 unique local fd00::/8", "fd12:3456:789a::1", true},

		// Blocked: link-local — 169.254.169.254 is the cloud metadata endpoint
		// (AWS/GCP/Azure IMDS) and the canonical SSRF credential-theft target.
		{"cloud metadata", "169.254.169.254", true},
		{"IPv4 link-local", "169.254.1.1", true},
		{"IPv6 link-local fe80::/10", "fe80::1", true},

		// Blocked: unspecified (0.0.0.0 / :: can route to local services).
		{"IPv4 unspecified", "0.0.0.0", true},
		{"IPv6 unspecified", "::", true},

		// Allowed: ordinary public addresses must still work.
		{"public DNS 8.8.8.8", "8.8.8.8", false},
		{"public DNS 1.1.1.1", "1.1.1.1", false},
		{"public host", "93.184.216.34", false},
		{"public IPv6", "2606:2800:220:1:248:1893:25c8:1946", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("test setup: %q is not a valid IP", tt.ip)
			}
			if got := isBlockedIP(ip); got != tt.want {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

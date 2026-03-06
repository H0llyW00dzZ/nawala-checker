// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

// Result represents the outcome of checking a single domain
// against a Nawala DNS server.
//
// Callers must always check [Result.Error] before reading [Result.Blocked].
// When Error is non-nil the Blocked value is unspecified and must be
// ignored — it may default to false even though the domain's actual
// blocking status is unknown.
type Result struct {
	// Domain is the domain name that was checked.
	Domain string

	// Blocked indicates whether the domain is blocked by Nawala.
	//
	// This field is only meaningful when [Result.Error] is nil.
	// If Error is non-nil, Blocked may be false regardless of the
	// domain's actual status. Always check Error first.
	Blocked bool

	// Server is the DNS server IP that was used for the check.
	Server string

	// Error is non-nil if the check encountered an error
	// (e.g., DNS timeout, invalid domain, NXDOMAIN).
	// When set, the [Result.Blocked] field is unreliable and must be ignored.
	Error error
}

// ServerStatus represents the health status of a single DNS server.
//
// Callers must always check [ServerStatus.Error] before reading [ServerStatus.Online].
// When Error is non-nil the Online value is unspecified and must be
// ignored — it may default to false even though the server's actual
// reachability is unknown.
type ServerStatus struct {
	// Server is the DNS server address that was checked,
	// in the same format as [DNSServer.Address].
	Server string

	// Online indicates whether the server is responding to queries.
	//
	// This field is only meaningful when [ServerStatus.Error] is nil.
	// If Error is non-nil, Online may be false regardless of the
	// server's actual reachability. Always check Error first.
	Online bool

	// LatencyMs is the round-trip time in milliseconds.
	//
	// Only meaningful when [ServerStatus.Online] is true.
	LatencyMs int64

	// Error is non-nil if the health check encountered an error.
	// When set, the [ServerStatus.Online] field is unreliable and must be ignored.
	Error error
}

// DNSServer represents a single DNS server configuration used for
// checking whether domains are blocked by Nawala.
//
// The [DNSServer.Address] field accepts any of the following formats:
//
//   - Plain IPv4 address:              "180.131.144.144"
//   - IPv4 address with port:          "180.131.144.144:5353"
//   - Plain IPv6 address:              "2001:db8::1"
//   - IPv6 address with port:          "[2001:db8::1]:5353"
//   - Hostname (domain name):          "dns.example.com"
//   - Hostname with port:              "dns.example.com:5353"
//
// When no port is specified, the default port is determined by the transport
// protocol: port 53 for UDP and TCP, and port 853 for DNS-over-TLS (tcp-tls).
type DNSServer struct {
	// Address is the DNS server to query.
	//
	// Accepted formats: plain IP ("8.8.8.8"), IP with port ("8.8.8.8:5353"),
	// hostname ("dns.example.com"), or hostname with port ("dns.example.com:5353").
	// IPv6 addresses with a port must be bracketed: "[::1]:5353".
	//
	// If no port is given, port 53 is used for UDP/TCP and port 853 for tcp-tls.
	Address string

	// Keyword is the substring to search for in DNS responses
	// that indicates a domain is blocked.
	Keyword string

	// QueryType is the DNS record type to query.
	// Use the dns query type constants (e.g., "ANY", "TXT", "A").
	QueryType string
}

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
type ServerStatus struct {
	// Server is the DNS server IP address.
	Server string

	// Online indicates whether the server is responding to queries.
	Online bool

	// LatencyMs is the round-trip time in milliseconds.
	// Only meaningful when Online is true.
	LatencyMs int64

	// Error is non-nil if the health check failed.
	Error error
}

// DNSServer represents a single DNS server configuration used for
// checking whether domains are blocked by Nawala.
type DNSServer struct {
	// Address is the IP address of the DNS server.
	Address string

	// Keyword is the substring to search for in DNS responses
	// that indicates a domain is blocked.
	Keyword string

	// QueryType is the DNS record type to query.
	// Use the dns query type constants (e.g., "ANY", "TXT", "A").
	QueryType string
}

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import "errors"

// Sentinel errors for the nawala checker package.
var (
	// ErrNoDNSServers is returned when no DNS servers are configured.
	ErrNoDNSServers = errors.New("nawala: no DNS servers configured")

	// ErrAllDNSFailed is returned when all configured DNS servers
	// fail to respond to queries.
	ErrAllDNSFailed = errors.New("nawala: all DNS servers failed to respond")

	// ErrInvalidDomain is returned when a domain name fails validation.
	ErrInvalidDomain = errors.New("nawala: invalid domain name")

	// ErrDNSTimeout is returned when a DNS query exceeds the configured timeout.
	ErrDNSTimeout = errors.New("nawala: DNS query timed out")

	// ErrInternalPanic is returned when an internal panic is recovered during execution.
	ErrInternalPanic = errors.New("nawala: internal panic recovered")

	// ErrNXDOMAIN is returned when the DNS server responds with NXDOMAIN (domain does not exist).
	ErrNXDOMAIN = errors.New("nawala: nxdomain")
)

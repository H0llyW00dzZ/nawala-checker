// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/miekg/dns"
)

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

	// ErrQueryRejected is returned when a DNS server explicitly rejects a query
	// (e.g., Format Error, Refused, Not Implemented).
	ErrQueryRejected = errors.New("nawala: query rejected by server")
)

// isConnError reports whether err indicates a broken or stale connection that
// warrants a transparent redial. It returns false for application-level errors
// (e.g. context cancellation, deadlines) so those are surfaced to the caller.
//
// Sources of stale-connection errors according to miekg/dns:
//   - [dns.ErrConnEmpty]   — underlying net.Conn is nil (conn was closed and
//     its internal Conn field set to nil).
//   - [io.EOF] / [io.ErrUnexpectedEOF] — server closed the TCP connection.
//   - *[net.OpError] wrapping an OS-level string such as
//     "use of closed network connection" (internal/poll.ErrNetClosing) or
//     "broken pipe" (syscall.EPIPE); both occur when writing to a locally
//     closed socket.
func isConnError(err error) bool {
	if err == nil {
		return false
	}
	// dns.ErrConnEmpty: the *dns.Conn had its net.Conn set to nil.
	if errors.Is(err, dns.ErrConnEmpty) {
		return true
	}
	// Raw EOF variants returned by ReadMsg on a server-closed connection.
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// *net.OpError wraps OS-level errors when the local end was closed or the
	// remote peer reset the connection.
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Err != nil {
		msg := opErr.Err.Error()
		if strings.Contains(msg, "use of closed network connection") ||
			strings.Contains(msg, "broken pipe") ||
			strings.Contains(msg, "connection reset by peer") {
			return true
		}
	}
	return false
}

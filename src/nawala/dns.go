// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// parseQueryType converts a string query type (e.g., "ANY", "TXT", "A")
// to the corresponding dns library constant.
func parseQueryType(qtype string) uint16 {
	switch strings.ToUpper(strings.TrimSpace(qtype)) {
	case "A":
		return dns.TypeA
	case "AAAA":
		return dns.TypeAAAA
	case "CNAME":
		return dns.TypeCNAME
	case "MX":
		return dns.TypeMX
	case "NS":
		return dns.TypeNS
	case "TXT":
		return dns.TypeTXT
	case "SOA":
		return dns.TypeSOA
	case "SRV":
		return dns.TypeSRV
	case "ANY":
		return dns.TypeANY
	default:
		return dns.TypeA
	}
}

// dnsQuery bundles the parameters for a single DNS query.
type dnsQuery struct {
	client    *dns.Client
	domain    string
	server    string
	qtype     uint16
	edns0Size uint16
}

// queryDNS sends a DNS query for the given domain to the specified server.
// It respects context cancellation and the configured timeout.
//
// EDNS0 is enabled by default ([RFC 6891]) to allow the server to return
// Extended DNS Errors ([RFC 8914]), such as EDE 15 (Blocked) used by Komdigi.
//
// [RFC 6891]: https://datatracker.ietf.org/doc/html/rfc6891
// [RFC 8914]: https://datatracker.ietf.org/doc/html/rfc8914
func queryDNS(ctx context.Context, q dnsQuery) (*dns.Msg, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(q.domain), q.qtype)
	msg.RecursionDesired = true
	msg.SetEdns0(q.edns0Size, false)

	// Ensure server has port.
	server := q.server
	if _, _, err := net.SplitHostPort(server); err != nil {
		// If it's an IPv6 address already enclosed in brackets but without a port, strip brackets first
		// so JoinHostPort can correctly re-add them along with the port.
		server = strings.TrimPrefix(server, "[")
		server = strings.TrimSuffix(server, "]")
		server = net.JoinHostPort(server, "53")
	}

	resp, _, err := q.client.ExchangeContext(ctx, msg, server)
	if err != nil {
		// 1. Did the context specifically exceed its deadline (timeout)?
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", ErrDNSTimeout, ctx.Err())
		}

		// 2. Did the underlying dns.Client hit a network timeout?
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, fmt.Errorf("%w: %v", ErrDNSTimeout, err)
		}

		// 3. For everything else (including context.Canceled), return the raw error
		return nil, err
	}

	return resp, nil
}

// containsKeyword scans all resource records in a DNS response message
// for the presence of a keyword (case-insensitive). This mirrors the
// parseDNSResponse function from the JavaScript implementation.
//
// It checks the Answer, Ns (authority), and Extra (additional) sections.
func containsKeyword(msg *dns.Msg, keyword string) bool {
	if msg == nil {
		return false
	}

	keyword = strings.ToLower(keyword)

	// Check all sections: Answer, Authority (Ns), Additional (Extra).
	sections := [][]dns.RR{msg.Answer, msg.Ns, msg.Extra}
	for _, section := range sections {
		for _, rr := range section {
			// Convert the entire record to its string representation
			// and check for the keyword. This is a broad match that
			// covers all record types (TXT data, CNAME targets, etc.).
			if strings.Contains(strings.ToLower(rr.String()), keyword) {
				return true
			}
		}
	}

	return false
}

// checkDNSHealth performs a health check on a single DNS server by
// resolving "google.com" and measuring the latency.
func checkDNSHealth(ctx context.Context, q dnsQuery) ServerStatus {
	// Override domain and qtype for the health check probe.
	q.domain = "google.com"
	q.qtype = dns.TypeA

	start := time.Now()

	resp, err := queryDNS(ctx, q)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return ServerStatus{
			Server: q.server,
			Online: false,
			Error:  err,
		}
	}

	if resp == nil || resp.Rcode != dns.RcodeSuccess {
		return ServerStatus{
			Server: q.server,
			Online: false,
			Error:  fmt.Errorf("unexpected response code: %d", resp.Rcode),
		}
	}

	return ServerStatus{
		Server:    q.server,
		Online:    true,
		LatencyMs: latency,
	}
}

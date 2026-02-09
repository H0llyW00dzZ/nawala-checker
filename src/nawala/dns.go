// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"fmt"
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

// queryDNS sends a DNS query for the given domain to the specified server.
// It respects context cancellation and the configured timeout.
func queryDNS(ctx context.Context, client *dns.Client, domain, server string, qtype uint16) (*dns.Msg, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), qtype)
	msg.RecursionDesired = true

	// Ensure server has port.
	if !strings.Contains(server, ":") {
		server = server + ":53"
	}

	// Create a channel to receive the result so we can
	// respect context cancellation.
	type dnsResult struct {
		msg *dns.Msg
		err error
	}
	ch := make(chan dnsResult, 1)

	go func() {
		resp, _, err := client.ExchangeContext(ctx, msg, server)
		ch <- dnsResult{msg: resp, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %v", ErrDNSTimeout, ctx.Err())
	case result := <-ch:
		if result.err != nil {
			return nil, result.err
		}
		return result.msg, nil
	}
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
func checkDNSHealth(ctx context.Context, client *dns.Client, server string) ServerStatus {
	start := time.Now()

	resp, err := queryDNS(ctx, client, "google.com", server, dns.TypeA)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return ServerStatus{
			Server: server,
			Online: false,
			Error:  err,
		}
	}

	if resp == nil || resp.Rcode != dns.RcodeSuccess {
		return ServerStatus{
			Server: server,
			Online: false,
			Error:  fmt.Errorf("unexpected response code: %d", resp.Rcode),
		}
	}

	return ServerStatus{
		Server:    server,
		Online:    true,
		LatencyMs: latency,
	}
}

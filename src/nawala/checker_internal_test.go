// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// startBlockingDNSServer starts a local DNS server that responds with a CNAME
// to "internetpositif.id." to simulate Nawala blocking behavior.
func startBlockingDNSServer(t *testing.T) (string, func()) {
	t.Helper()

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeCNAME,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			Target: "internetpositif.id.",
		})
		_ = w.WriteMsg(m)
	})

	return startTestDNSServer(t, handler)
}

// startNormalDNSServer starts a local DNS server that responds normally (not blocked).
func startNormalDNSServer(t *testing.T) (string, func()) {
	t.Helper()

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP("93.184.216.34"),
		})
		_ = w.WriteMsg(m)
	})

	return startTestDNSServer(t, handler)
}

func TestCheckConcurrent(t *testing.T) {
	addr, cleanup := startNormalDNSServer(t)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithTimeout(5*time.Second),
	)

	ctx := context.Background()
	domains := []string{"example.com", "google.com", "github.com"}

	results, err := c.Check(ctx, domains...)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(results) != len(domains) {
		t.Fatalf("expected %d results, got %d", len(domains), len(results))
	}

	for i, r := range results {
		if r.Domain != domains[i] {
			t.Errorf("result[%d].Domain = %q, want %q", i, r.Domain, domains[i])
		}
		if r.Error != nil {
			t.Errorf("result[%d] unexpected error: %v", i, r.Error)
		}
		if r.Blocked {
			t.Errorf("result[%d] unexpectedly blocked", i)
		}
	}
}

func TestCheckBlocked(t *testing.T) {
	addr, cleanup := startBlockingDNSServer(t)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
	)

	ctx := context.Background()
	result, err := c.CheckOne(ctx, "example.com")
	if err != nil {
		t.Fatalf("CheckOne failed: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("unexpected result error: %v", result.Error)
	}
	if !result.Blocked {
		t.Error("expected domain to be blocked")
	}
}

func TestCheckOneWithCaching(t *testing.T) {
	queryCount := 0
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		queryCount++
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP("1.2.3.4"),
		})
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startTestDNSServer(t, handler)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithCacheTTL(5*time.Minute),
	)

	ctx := context.Background()

	// First call — hits DNS.
	r1, err := c.CheckOne(ctx, "example.com")
	if err != nil {
		t.Fatalf("first CheckOne: %v", err)
	}
	if r1.Error != nil {
		t.Fatalf("first check result error: %v", r1.Error)
	}
	if queryCount != 1 {
		t.Errorf("expected 1 DNS query, got %d", queryCount)
	}

	// Second call — should hit cache, no new DNS query.
	r2, err := c.CheckOne(ctx, "example.com")
	if err != nil {
		t.Fatalf("second CheckOne: %v", err)
	}
	if r2.Error != nil {
		t.Fatalf("second check result error: %v", r2.Error)
	}
	if queryCount != 1 {
		t.Errorf("expected 1 DNS query after cache hit, got %d", queryCount)
	}
}

func TestDNSStatusWithLocalServer(t *testing.T) {
	addr, cleanup := startNormalDNSServer(t)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "test", QueryType: "A"},
		}),
	)

	ctx := context.Background()
	statuses, err := c.DNSStatus(ctx)
	if err != nil {
		t.Fatalf("DNSStatus: %v", err)
	}

	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}

	s := statuses[0]
	if !s.Online {
		t.Errorf("expected Online=true, got false (error: %v)", s.Error)
	}
	if s.LatencyMs < 0 {
		t.Errorf("expected non-negative latency, got %d", s.LatencyMs)
	}
}

func TestFailover(t *testing.T) {
	// First server is unreachable, second is fine.
	goodAddr, cleanup := startNormalDNSServer(t)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: "127.0.0.1:19998", Keyword: "internetpositif", QueryType: "A"}, // unreachable
			{Address: goodAddr, Keyword: "internetpositif", QueryType: "A"},          // working
		}),
		WithTimeout(500*time.Millisecond),
		WithMaxRetries(0), // No retries to speed up the test.
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.CheckOne(ctx, "example.com")
	if err != nil {
		t.Fatalf("CheckOne: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("unexpected result error: %v", result.Error)
	}
	// The result should come from the second server.
	if result.Server != goodAddr {
		t.Errorf("expected server=%q, got %q", goodAddr, result.Server)
	}
}

func TestAllServersFail(t *testing.T) {
	c := New(
		WithServers([]DNSServer{
			{Address: "127.0.0.1:19998", Keyword: "test", QueryType: "A"},
			{Address: "127.0.0.1:19999", Keyword: "test", QueryType: "A"},
		}),
		WithTimeout(300*time.Millisecond),
		WithMaxRetries(0),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := c.CheckOne(ctx, "example.com")
	if err != nil {
		t.Fatalf("CheckOne: %v", err)
	}
	if !errors.Is(result.Error, ErrAllDNSFailed) {
		t.Errorf("expected ErrAllDNSFailed, got %v", result.Error)
	}
}

func TestQueryWithRetriesSuccess(t *testing.T) {
	attempts := 0
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		attempts++
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP("1.2.3.4"),
		})
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startTestDNSServer(t, handler)
	defer cleanup()

	c := New(
		WithTimeout(5*time.Second),
		WithMaxRetries(2),
	)

	ctx := context.Background()
	srv := DNSServer{Address: addr, Keyword: "test", QueryType: "A"}
	result, err := c.queryWithRetries(ctx, "example.com", srv, dns.TypeA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", result.Domain, "example.com")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (success on first try), got %d", attempts)
	}
}

func TestQueryWithRetriesRetry(t *testing.T) {
	attempts := 0
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		attempts++
		if attempts < 3 {
			// First two attempts: don't respond (let timeout trigger retry).
			return
		}
		// Third attempt: respond successfully.
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP("1.2.3.4"),
		})
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startTestDNSServer(t, handler)
	defer cleanup()

	c := New(
		WithTimeout(300*time.Millisecond),
		WithMaxRetries(2),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	srv := DNSServer{Address: addr, Keyword: "test", QueryType: "A"}
	result, err := c.queryWithRetries(ctx, "example.com", srv, dns.TypeA)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if result.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", result.Domain, "example.com")
	}
}

func TestQueryWithRetriesContextCancel(t *testing.T) {
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		// Never respond.
		time.Sleep(10 * time.Second)
	})

	addr, cleanup := startTestDNSServer(t, handler)
	defer cleanup()

	c := New(
		WithTimeout(5*time.Second),
		WithMaxRetries(3),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	srv := DNSServer{Address: addr, Keyword: "test", QueryType: "A"}
	_, err := c.queryWithRetries(ctx, "example.com", srv, dns.TypeA)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestCheckDomainNormalization(t *testing.T) {
	addr, cleanup := startNormalDNSServer(t)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
	)

	ctx := context.Background()

	// Uppercase and whitespace should be normalized.
	result, err := c.CheckOne(ctx, "  EXAMPLE.COM  ")
	if err != nil {
		t.Fatalf("CheckOne: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("unexpected result error: %v", result.Error)
	}
	if result.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", result.Domain, "example.com")
	}
}

func TestCheckWithNilCache(t *testing.T) {
	addr, cleanup := startNormalDNSServer(t)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithCache(nil), // Disable caching.
	)

	ctx := context.Background()
	result, err := c.CheckOne(ctx, "example.com")
	if err != nil {
		t.Fatalf("CheckOne: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("unexpected result error: %v", result.Error)
	}
}

func TestFlushCacheNilSafe(t *testing.T) {
	c := New(WithCache(nil))
	// Should not panic.
	c.FlushCache()
}

func TestCheckOneNoServers(t *testing.T) {
	c := New(WithServers(nil))
	ctx := context.Background()

	_, err := c.CheckOne(ctx, "example.com")
	if !errors.Is(err, ErrNoDNSServers) {
		t.Errorf("expected ErrNoDNSServers, got %v", err)
	}
}

func TestDNSStatusNoServers(t *testing.T) {
	c := New(WithServers(nil))
	ctx := context.Background()

	_, err := c.DNSStatus(ctx)
	if !errors.Is(err, ErrNoDNSServers) {
		t.Errorf("expected ErrNoDNSServers, got %v", err)
	}
}

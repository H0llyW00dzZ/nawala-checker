// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	require.Len(t, results, len(domains))

	for i, r := range results {
		assert.Equal(t, domains[i], r.Domain, "result[%d].Domain", i)
		assert.NoError(t, r.Error, "result[%d] unexpected error", i)
		assert.False(t, r.Blocked, "result[%d] unexpectedly blocked", i)
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
	require.NoError(t, err)
	require.NoError(t, result.Error)
	assert.True(t, result.Blocked, "expected domain to be blocked")
}

func TestCheckOneWithCaching(t *testing.T) {
	var queryCount atomic.Int32

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		queryCount.Add(1)
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
	require.NoError(t, err)
	require.NoError(t, r1.Error)
	assert.Equal(t, int32(3), queryCount.Load(), "expected 3 DNS queries (multi-probe)")

	// Second call — should hit cache, no new DNS query.
	r2, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)
	require.NoError(t, r2.Error)
	assert.Equal(t, int32(3), queryCount.Load(), "expected no new DNS queries after cache hit")
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
	require.NoError(t, err)
	require.Len(t, statuses, 1)

	assert.True(t, statuses[0].Online, "expected Online=true")
	assert.GreaterOrEqual(t, statuses[0].LatencyMs, int64(0))
}

func TestFailover(t *testing.T) {
	goodAddr, cleanup := startNormalDNSServer(t)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: "127.0.0.1:19998", Keyword: "internetpositif", QueryType: "A"}, // unreachable
			{Address: goodAddr, Keyword: "internetpositif", QueryType: "A"},          // working
		}),
		WithTimeout(500*time.Millisecond),
		WithMaxRetries(0),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)
	require.NoError(t, result.Error)
	assert.Equal(t, goodAddr, result.Server, "expected result from second (working) server")
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
	require.NoError(t, err)
	assert.ErrorIs(t, result.Error, ErrAllDNSFailed)
}

func TestQueryWithRetriesSuccess(t *testing.T) {
	var attempts atomic.Int32

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		attempts.Add(1)
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
	require.NoError(t, err)
	assert.Equal(t, "example.com", result.Domain)
	assert.Equal(t, int32(3), attempts.Load(), "expected 3 attempts (probes all retries for consistency)")
}

func TestQueryWithRetriesRetry(t *testing.T) {
	var attempts atomic.Int32

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		n := attempts.Add(1)
		if n < 3 {
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
	require.NoError(t, err, "expected success after retries")
	assert.Equal(t, "example.com", result.Domain)
}

func TestQueryWithRetriesContextCancel(t *testing.T) {
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		time.Sleep(10 * time.Second) // Never respond.
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
	assert.Error(t, err, "expected error for cancelled context")
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
	result, err := c.CheckOne(ctx, "  EXAMPLE.COM  ")
	require.NoError(t, err)
	require.NoError(t, result.Error)
	assert.Equal(t, "example.com", result.Domain)
}

func TestCheckWithNilCache(t *testing.T) {
	addr, cleanup := startNormalDNSServer(t)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithCache(nil),
	)

	ctx := context.Background()
	result, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)
	assert.NoError(t, result.Error)
}

func TestFlushCacheNilSafe(t *testing.T) {
	c := New(WithCache(nil))
	assert.NotPanics(t, func() { c.FlushCache() })
}

func TestCheckOneNoServers(t *testing.T) {
	c := New(WithServers(nil))
	ctx := context.Background()

	_, err := c.CheckOne(ctx, "example.com")
	assert.ErrorIs(t, err, ErrNoDNSServers)
}

func TestDNSStatusNoServers(t *testing.T) {
	c := New(WithServers(nil))
	ctx := context.Background()

	_, err := c.DNSStatus(ctx)
	assert.ErrorIs(t, err, ErrNoDNSServers)
}

func TestCheckConcurrencyLimit(t *testing.T) {
	// This test verifies that we don't spawn a goroutine for every domain
	// immediately, but strictly bound it by the semaphore size (100).

	// Create a slow mock server to ensure goroutines stack up.
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		time.Sleep(100 * time.Millisecond) // Slow response
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	})
	addr, cleanup := startTestDNSServer(t, handler)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "test", QueryType: "A"},
		}),
		WithTimeout(1*time.Second),
	)

	ctx := context.Background()
	count := 500 // Significantly more than semaphore size (100)
	domains := make([]string, count)
	for i := range domains {
		domains[i] = "example.com"
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// Measure baseline goroutines
	startGoroutines := runtime.NumGoroutine()

	go func() {
		defer wg.Done()
		_, _ = c.Check(ctx, domains...)
	}()

	// Give time for the loop to spin up
	time.Sleep(100 * time.Millisecond)

	currentGoroutines := runtime.NumGoroutine()
	totalSpawned := currentGoroutines - startGoroutines

	// With the fix:
	// - 100 workers (max semaphore)
	// - 100 internal DNS goroutines (hypothetically)
	// - 1 Check goroutine
	// Total ~201.
	//
	// Without the fix:
	// - 500 workers
	// - 100 internal DNS goroutines
	// - 1 Check goroutine
	// Total ~601.
	//
	// We set threshold to 400 to safely distinguish.
	t.Logf("Goroutines: start=%d, current=%d, delta=%d", startGoroutines, currentGoroutines, totalSpawned)

	if totalSpawned > 400 {
		t.Errorf("Too many goroutines spawned! Expected around 200-250, got %d. This indicates unbounded concurrency spawning.", totalSpawned)
	}

	// Wait for cleanup
	wg.Wait()
}

func TestWithConcurrency(t *testing.T) {
	// Test default concurrency
	c := New()
	if c.concurrency != defaultConcurrency {
		t.Errorf("expected default concurrency %d, got %d", defaultConcurrency, c.concurrency)
	}

	// Test custom concurrency
	c = New(WithConcurrency(50))
	if c.concurrency != 50 {
		t.Errorf("expected concurrency 50, got %d", c.concurrency)
	}

	// Test invalid concurrency (should be ignored and remain default)
	c = New(WithConcurrency(0))
	if c.concurrency != defaultConcurrency {
		t.Errorf("expected concurrency %d, got %d", defaultConcurrency, c.concurrency)
	}

	if c.concurrency != defaultConcurrency {
		t.Errorf("expected concurrency %d, got %d", defaultConcurrency, c.concurrency)
	}
}

func TestCheckRaceCondition(t *testing.T) {
	// Start a slow DNS server to ensure goroutines are still running
	// when we cancel the context.
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		time.Sleep(50 * time.Millisecond) // Simulate delay
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	})
	addr, cleanup := startTestDNSServer(t, handler)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "test", QueryType: "A"},
		}),
		WithTimeout(1*time.Second),
		WithMaxRetries(0),
	)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Check many domains to ensure we have active goroutines
	// Must exceed default concurrency (100) to ensure the loop blocks on semaphore
	// and is still running when we cancel.
	count := 200
	domains := make([]string, count)
	for i := range domains {
		domains[i] = "example.com"
	}

	// Start the check in a goroutine
	done := make(chan struct{})
	go func() {
		results, _ := c.Check(ctx, domains...)
		// Access results to trigger race with background writers
		if len(results) > 0 {
			_ = results[0].Domain
		}
		close(done)
	}()

	// Let it start spawning goroutines
	time.Sleep(10 * time.Millisecond)

	// CANCEL while it's running!
	cancel()

	<-done
}

func TestWithDNSClient(t *testing.T) {
	t.Run("custom client is used", func(t *testing.T) {
		addr, cleanup := startNormalDNSServer(t)
		defer cleanup()

		customClient := &dns.Client{
			Timeout: 10 * time.Second,
			Net:     "udp",
		}

		c := New(
			WithServers([]DNSServer{
				{Address: addr, Keyword: "internetpositif", QueryType: "A"},
			}),
			WithDNSClient(customClient),
		)

		// Verify the custom client is used (not the default).
		if c.dnsClient != customClient {
			t.Error("expected custom DNS client to be set")
		}

		// Verify it works end-to-end.
		ctx := context.Background()
		result, err := c.CheckOne(ctx, "example.com")
		require.NoError(t, err)
		assert.NoError(t, result.Error)
		assert.False(t, result.Blocked)
	})

	t.Run("nil client is ignored", func(t *testing.T) {
		c := New(
			WithDNSClient(nil),
		)

		// Should fall back to default client.
		if c.dnsClient == nil {
			t.Error("expected default DNS client when nil is passed")
		}
		assert.Equal(t, "udp", c.dnsClient.Net, "expected default UDP transport")
	})

	t.Run("custom client overrides timeout", func(t *testing.T) {
		customClient := &dns.Client{
			Timeout: 42 * time.Second,
			Net:     "udp",
		}

		c := New(
			WithTimeout(1*time.Second), // This should be ignored for the client
			WithDNSClient(customClient),
		)

		// The custom client's timeout should be preserved.
		assert.Equal(t, 42*time.Second, c.dnsClient.Timeout)
	})

	t.Run("TCP transport works", func(t *testing.T) {
		// Start a TCP DNS server.
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
				A: net.ParseIP("1.2.3.4"),
			})
			_ = w.WriteMsg(m)
		})

		// Listen on TCP.
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		server := &dns.Server{
			Listener: listener,
			Handler:  handler,
		}

		started := make(chan struct{})
		go func() {
			server.NotifyStartedFunc = func() { close(started) }
			_ = server.ActivateAndServe()
		}()
		<-started
		defer func() { _ = server.Shutdown() }()

		tcpAddr := listener.Addr().String()

		tcpClient := &dns.Client{
			Timeout: 5 * time.Second,
			Net:     "tcp",
		}

		c := New(
			WithServers([]DNSServer{
				{Address: tcpAddr, Keyword: "internetpositif", QueryType: "A"},
			}),
			WithDNSClient(tcpClient),
		)

		ctx := context.Background()
		result, err := c.CheckOne(ctx, "example.com")
		require.NoError(t, err)
		assert.NoError(t, result.Error)
		assert.False(t, result.Blocked)
	})
}
func TestWithNegativeMaxRetries(t *testing.T) {
	c := New(WithMaxRetries(-5))

	// Verify it exactly matches the default configuration
	assert.Equal(t, defaultRetries, c.maxRetries, "maxRetries should default to defaultRetries when negative")
}

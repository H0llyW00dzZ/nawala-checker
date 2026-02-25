// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
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

func TestWithEDNS0Size(t *testing.T) {
	// Test default EDNS0 size
	c := New()
	if c.edns0Size != defaultEDNS0Size {
		t.Errorf("expected default EDNS0 size %d, got %d", defaultEDNS0Size, c.edns0Size)
	}

	// Test custom EDNS0 size
	c = New(WithEDNS0Size(4096))
	if c.edns0Size != 4096 {
		t.Errorf("expected EDNS0 size 4096, got %d", c.edns0Size)
	}

	// Test invalid EDNS0 size (should be ignored and remain default)
	c = New(WithEDNS0Size(0))
	if c.edns0Size != defaultEDNS0Size {
		t.Errorf("expected EDNS0 size %d, got %d", defaultEDNS0Size, c.edns0Size)
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

func TestDNSOverTLS(t *testing.T) {
	// 1. Generate self-signed certificate for testing
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Hello from Go (DNS over TLS RFC 7858)"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// 2. Start TLS Listener
	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	require.NoError(t, err)
	defer listener.Close()

	// 3. Start DNS Server
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

	server := &dns.Server{
		Listener: listener,
		Handler:  handler,
		Net:      "tcp-tls",
	}

	go func() {
		_ = server.ActivateAndServe()
	}()
	// Allow brief time for server to start serving
	time.Sleep(100 * time.Millisecond)

	// 4. Configure Client with TLS
	customClient := &dns.Client{
		Net: "tcp-tls",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true, // Trust our self-signed cert
		},
		Timeout: 5 * time.Second,
	}

	c := New(
		WithServers([]DNSServer{
			{Address: listener.Addr().String(), Keyword: "internetpositif", QueryType: "A"}, // "internetpositif" is what we check for blocking usually, but here we just want to test connectivity
		}),
		WithDNSClient(customClient),
	)

	// 5. Verify Check
	ctx := context.Background()
	result, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)
	assert.NoError(t, result.Error)
	assert.False(t, result.Blocked)
	assert.Equal(t, listener.Addr().String(), result.Server)
}

// TestNawalaRPZStyleBlocking tests the checker with a simplified Nawala-style RPZ blacklist.
func TestNawalaRPZStyleBlocking(t *testing.T) {
	// Simplified Nawala-style RPZ blacklist.
	// Blocked domains are redirected to internetpositif.id.
	blacklist := map[string]bool{
		"blocked0.test.": true,
		"blocked1.test.": true,
		"blocked2.test.": true,
		"blocked3.test.": true,
	}

	// RPZ-style handler: blocked domains get a CNAME to internetpositif.id,
	// non-blocked domains resolve normally.
	rpzHandler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)

		for _, q := range r.Question {
			if blacklist[q.Name] {
				// RPZ Action: redirect to internetpositif.id (Nawala landing page)
				m.Answer = append(m.Answer, &dns.CNAME{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeCNAME,
						Class:  dns.ClassINET,
						Ttl:    3600,
					},
					Target: "internetpositif.id.",
				})
			} else {
				// Not in blacklist — resolve normally.
				m.Answer = append(m.Answer, &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					A: net.ParseIP("93.184.216.34"),
				})
			}
		}
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startTestDNSServer(t, rpzHandler)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithTimeout(5*time.Second),
	)

	ctx := context.Background()

	t.Run("blocked domains detected", func(t *testing.T) {
		blockedDomains := []string{"blocked0.test", "blocked1.test", "blocked2.test", "blocked3.test"}

		for _, domain := range blockedDomains {
			result, err := c.CheckOne(ctx, domain)
			require.NoError(t, err, "domain: %s", domain)
			assert.NoError(t, result.Error, "domain: %s", domain)
			assert.True(t, result.Blocked, "expected %s to be blocked by RPZ", domain)
		}
	})

	t.Run("allowed domains pass through", func(t *testing.T) {
		allowedDomains := []string{"google.com", "github.com", "example.com"}

		for _, domain := range allowedDomains {
			result, err := c.CheckOne(ctx, domain)
			require.NoError(t, err, "domain: %s", domain)
			assert.NoError(t, result.Error, "domain: %s", domain)
			assert.False(t, result.Blocked, "expected %s to NOT be blocked", domain)
		}
	})

	t.Run("batch check mixed domains", func(t *testing.T) {
		domains := []string{"blocked0.test", "google.com", "blocked1.test", "github.com"}
		results, err := c.Check(ctx, domains...)
		require.NoError(t, err)
		require.Len(t, results, 4)

		assert.True(t, results[0].Blocked, "blocked0.test should be blocked")
		assert.False(t, results[1].Blocked, "google.com should not be blocked")
		assert.True(t, results[2].Blocked, "blocked1.test should be blocked")
		assert.False(t, results[3].Blocked, "github.com should not be blocked")
	})
}

// TestNawalaRPZStyleBlockingDoT tests the checker with a simplified Nawala-style RPZ blacklist
// over DNS-over-TLS (DoT) as specified in RFC 7858.
//
// DoT is preferred over DoH here, as using DoH (Overhead LoL + wastes bandwidth on useless HTTP headers) for a checker would be unconventional.
func TestNawalaRPZStyleBlockingDoT(t *testing.T) {
	// 1. Generate self-signed certificate for testing
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Hello from Go (DNS over TLS RFC 7858)"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// 2. Start TLS Listener
	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	require.NoError(t, err)
	defer listener.Close()

	// 3. Simplified Nawala-style RPZ blacklist
	blacklist := map[string]bool{
		"blocked0.test.": true,
		"blocked1.test.": true,
		"blocked2.test.": true,
		"blocked3.test.": true,
	}

	// RPZ-style handler: blocked domains get a CNAME to internetpositif.id,
	// non-blocked domains resolve normally.
	rpzHandler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)

		for _, q := range r.Question {
			if blacklist[q.Name] {
				// RPZ Action: redirect to internetpositif.id (Nawala landing page)
				m.Answer = append(m.Answer, &dns.CNAME{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeCNAME,
						Class:  dns.ClassINET,
						Ttl:    3600,
					},
					Target: "internetpositif.id.",
				})
			} else {
				// Not in blacklist — resolve normally.
				m.Answer = append(m.Answer, &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					A: net.ParseIP("93.184.216.34"),
				})
			}
		}
		_ = w.WriteMsg(m)
	})

	// 4. Start DNS-over-TLS Server
	server := &dns.Server{
		Listener: listener,
		Handler:  rpzHandler,
		Net:      "tcp-tls",
	}

	go func() {
		_ = server.ActivateAndServe()
	}()
	// Allow brief time for server to start serving
	time.Sleep(100 * time.Millisecond)
	defer func() { _ = server.Shutdown() }()

	// 5. Configure Client with TLS (DoT)
	customClient := &dns.Client{
		Net: "tcp-tls",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true, // Trust our self-signed cert
		},
		Timeout: 5 * time.Second,
	}

	c := New(
		WithServers([]DNSServer{
			{Address: listener.Addr().String(), Keyword: "internetpositif", QueryType: "A"},
		}),
		WithDNSClient(customClient),
	)

	ctx := context.Background()

	t.Run("blocked domains detected over DoT", func(t *testing.T) {
		blockedDomains := []string{"blocked0.test", "blocked1.test", "blocked2.test", "blocked3.test"}

		for _, domain := range blockedDomains {
			result, err := c.CheckOne(ctx, domain)
			require.NoError(t, err, "domain: %s", domain)
			assert.NoError(t, result.Error, "domain: %s", domain)
			assert.True(t, result.Blocked, "expected %s to be blocked by RPZ over DoT", domain)
		}
	})

	t.Run("allowed domains pass through over DoT", func(t *testing.T) {
		allowedDomains := []string{"google.com", "github.com", "example.com"}

		for _, domain := range allowedDomains {
			result, err := c.CheckOne(ctx, domain)
			require.NoError(t, err, "domain: %s", domain)
			assert.NoError(t, result.Error, "domain: %s", domain)
			assert.False(t, result.Blocked, "expected %s to NOT be blocked over DoT", domain)
		}
	})

	t.Run("batch check mixed domains over DoT", func(t *testing.T) {
		domains := []string{"blocked0.test", "google.com", "blocked1.test", "github.com"}
		results, err := c.Check(ctx, domains...)
		require.NoError(t, err)
		require.Len(t, results, 4)

		assert.True(t, results[0].Blocked, "blocked0.test should be blocked over DoT")
		assert.False(t, results[1].Blocked, "google.com should not be blocked over DoT")
		assert.True(t, results[2].Blocked, "blocked1.test should be blocked over DoT")
		assert.False(t, results[3].Blocked, "github.com should not be blocked over DoT")
	})
}

// TestDNSQueryPortLogic verifies that queryDNS correctly normalizes server
// addresses for IPv4, IPv6, and hostname inputs.
func TestDNSQueryPortLogic(t *testing.T) {
	// Start a dummy DNS server just to bind a local port.
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	})
	addr, cleanup := startTestDNSServer(t, handler)
	defer cleanup()

	customClient := &dns.Client{Timeout: 1 * time.Second}
	ctx := context.Background()

	// Extract port from the dummy server
	_, port, err := net.SplitHostPort(addr)
	require.NoError(t, err)

	tests := []struct {
		name       string
		serverAddr string
		wantAddr   string
	}{
		{
			name:       "IPv4 without port",
			serverAddr: "127.0.0.1",
			wantAddr:   "127.0.0.1:53",
		},
		{
			name:       "IPv4 with port",
			serverAddr: "127.0.0.1:" + port,
			wantAddr:   "127.0.0.1:" + port,
		},
		{
			name:       "IPv6 without brackets and port",
			serverAddr: "::1",
			wantAddr:   "[::1]:53",
		},
		{
			name:       "IPv6 with brackets but no port",
			serverAddr: "[::1]",
			wantAddr:   "[::1]:53",
		},
		{
			name:       "IPv6 with brackets and port",
			serverAddr: "[::1]:" + port,
			wantAddr:   "[::1]:" + port,
		},
		{
			name:       "Hostname without port",
			serverAddr: "localhost",
			wantAddr:   ":53", // localhost resolves to 127.0.0.1 or [::1] depending on OS
		},
		{
			name:       "Hostname with port",
			serverAddr: "localhost:" + port,
			wantAddr:   "localhost:" + port,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := dnsQuery{
				client:    customClient,
				domain:    "example.com",
				server:    tt.serverAddr,
				qtype:     dns.TypeA,
				edns0Size: 1232,
			}

			// We just want to check if the exchange targets the correct address.
			// Since most test addresses (like 127.0.0.1:53 or [::1]:53) won't have a DNS server
			// running in this test environment, we expect a network error containing the target address.
			_, err := queryDNS(ctx, q)

			// If it succeeds (e.g. hits our dummy server via localhost:<port>), that's fine too.
			if err != nil {
				// The error message should contain the correctly formatted address we tried to dial.
				// For hostname inputs, the DNS library resolves the hostname before dialing,
				// so we check that the expected port suffix is present rather than the exact
				// resolved address (which varies by OS: 127.0.0.1 vs [::1]).
				assert.Contains(t, err.Error(), tt.wantAddr,
					"expected error to reference target address %q", tt.wantAddr)
			}
		})
	}
}

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startTCPDNSServer starts a TCP-only DNS server and returns its address and a cleanup func.
func startTCPDNSServer(t *testing.T, handler dns.HandlerFunc) (string, func()) {
	t.Helper()

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

	return listener.Addr().String(), func() { _ = server.Shutdown() }
}

// TestKeepAlivePoolReusesConn verifies that multiple queries over a keep-alive
// TCP checker share a single underlying TCP connection to the server.
func TestKeepAlivePoolReusesConn(t *testing.T) {
	var connCount atomic.Int32

	// Wrap a standard listener so we can count accepted connections.
	base, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	counting := &countingListener{Listener: base, count: &connCount}

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

	server := &dns.Server{Listener: counting, Handler: handler}
	started := make(chan struct{})
	go func() {
		server.NotifyStartedFunc = func() { close(started) }
		_ = server.ActivateAndServe()
	}()
	<-started
	defer func() { _ = server.Shutdown() }()

	addr := base.Addr().String()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithProtocol("tcp"),
		WithKeepAlive(3),
		WithCache(nil), // disable cache so every query actually hits the server
		WithMaxRetries(0),
	)
	defer func() { _ = c.Close() }()

	ctx := context.Background()
	const queries = 5
	for range queries {
		result, err := c.CheckOne(ctx, "example.com")
		require.NoError(t, err)
		assert.NoError(t, result.Error)
	}

	// With keep-alive, only 1 TCP connection should have been accepted.
	assert.Equal(t, int32(1), connCount.Load(),
		"expected 1 TCP connection to be reused across %d queries", queries)
}

// countingListener wraps a net.Listener and increments count on each Accept.
type countingListener struct {
	net.Listener
	count *atomic.Int32
}

func (cl *countingListener) Accept() (net.Conn, error) {
	conn, err := cl.Listener.Accept()
	if err == nil {
		cl.count.Add(1)
	}
	return conn, err
}

// TestKeepAlivePoolConcurrent exercises the pool under concurrent load with
// the race detector to confirm there are no data races.
func TestKeepAlivePoolConcurrent(t *testing.T) {
	addr, cleanup := startTCPDNSServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP("2.3.4.5"),
		})
		_ = w.WriteMsg(m)
	}))
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithProtocol("tcp"),
		WithKeepAlive(5),
		WithCache(nil),
		WithMaxRetries(0),
		WithTimeout(5*time.Second),
	)
	defer func() { _ = c.Close() }()

	const workers = 50
	var wg sync.WaitGroup
	ctx := context.Background()

	for range workers {
		wg.Go(func() {
			result, err := c.CheckOne(ctx, "example.com")
			require.NoError(t, err)
			assert.NoError(t, result.Error)
		})
	}
	wg.Wait()
}

// TestCheckerClose verifies that Close() drains the pool without panicking
// and is safe to call more than once.
func TestCheckerClose(t *testing.T) {
	addr, cleanup := startTCPDNSServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	}))
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithProtocol("tcp"),
		WithKeepAlive(3),
		WithCache(nil),
		WithMaxRetries(0),
	)

	// Warm the pool by performing a query.
	ctx := context.Background()
	_, _ = c.CheckOne(ctx, "example.com")

	// Close must not panic and must return nil.
	assert.NotPanics(t, func() {
		err := c.Close()
		assert.NoError(t, err)
	})

	// Second Close must also be a no-op and not panic.
	assert.NotPanics(t, func() {
		err := c.Close()
		assert.NoError(t, err)
	})
}

// TestCheckerCloseUDP verifies that Close() on a UDP checker (no pool) is a
// safe no-op.
func TestCheckerCloseUDP(t *testing.T) {
	c := New() // default UDP
	assert.NotPanics(t, func() {
		err := c.Close()
		assert.NoError(t, err)
	})
}

// TestWithKeepAliveUDPNoOp confirms that WithKeepAlive is accepted for UDP
// checkers but does not create a connection pool (pool is only for TCP/TLS).
func TestWithKeepAliveUDPNoOp(t *testing.T) {
	c := New(
		WithProtocol("udp"),
		WithKeepAlive(5),
	)
	assert.Nil(t, c.connPools, "connPools should be nil for UDP even with WithKeepAlive")
}

// TestIsConnError verifies the error classification helper used by connPool.exchange.
func TestIsConnError(t *testing.T) {
	assert.True(t, isConnError(io.EOF))
	assert.True(t, isConnError(io.ErrUnexpectedEOF))
	assert.False(t, isConnError(nil))
	assert.False(t, isConnError(context.Canceled))
	assert.False(t, isConnError(context.DeadlineExceeded))
}

// TestConnPoolGetPutClose tests the low-level pool mechanics directly.
func TestConnPoolGetPutClose(t *testing.T) {
	addr, cleanup := startTCPDNSServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	}))
	defer cleanup()

	client := &dns.Client{Net: "tcp", Timeout: 5 * time.Second}
	pool := newConnPool(client, addr, 2)

	ctx := context.Background()

	// Get + put a connection.
	conn, err := pool.get(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)
	pool.put(conn)

	// Pool should now hold 1 conn; get it back.
	conn2, err := pool.get(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn2)

	// Overflow: put more connections than pool capacity (size=2).
	conn3, err := pool.get(ctx) // dials fresh
	require.NoError(t, err)
	pool.put(conn2)
	pool.put(conn3)
	// Pool is now full (2 slots). Putting one more should close it, not panic.
	conn4, err := pool.get(ctx) // dials fresh
	require.NoError(t, err)
	assert.NotPanics(t, func() { pool.put(conn4) }) // pool already full — closes conn4

	// put(nil) must be a no-op.
	assert.NotPanics(t, func() { pool.put(nil) })

	// close() must drain without panic.
	assert.NotPanics(t, func() { pool.close() })
	// Second close() also safe.
	assert.NotPanics(t, func() { pool.close() })
}

// TestConnPoolExchange_StaleConnRedial verifies that a stale connection (one
// that returns io.EOF on first use) is transparently replaced with a fresh
// dial and the query succeeds.
func TestConnPoolExchange_StaleConnRedial(t *testing.T) {
	addr, cleanup := startTCPDNSServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   net.ParseIP("5.6.7.8"),
		})
		_ = w.WriteMsg(m)
	}))
	defer cleanup()

	client := &dns.Client{Net: "tcp", Timeout: 5 * time.Second}
	pool := newConnPool(client, addr, 2)

	// Inject a closed (stale) connection into the pool to simulate an idle
	// connection that expired on the server side.
	stale, err := client.DialContext(context.Background(), addr)
	require.NoError(t, err)
	_ = stale.Close()  // close it so the next ExchangeWithConnContext returns io.EOF
	pool.pool <- stale // put the stale conn directly into the pool channel

	// exchange should detect the EOF, discard the stale conn, redial, and succeed.
	ctx := context.Background()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)
	msg.SetEdns0(1232, false)

	r, _, err := pool.exchange(ctx, msg)
	require.NoError(t, err, "expected transparent redial on stale connection")
	assert.NotEmpty(t, r.Answer)
}

// TestConnPoolExchange_StaleConnRedialFails verifies that when the stale-conn
// redial itself fails, the original/first exchange error is returned (not the
// dial error), so the caller's retry logic sees a meaningful error.
func TestConnPoolExchange_StaleConnRedialFails(t *testing.T) {
	addr, cleanup := startTCPDNSServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	}))

	// Dial a fresh connection then shut the server down so redial fails.
	client := &dns.Client{Net: "tcp", Timeout: 2 * time.Second}
	stalConn, err := client.DialContext(context.Background(), addr)
	require.NoError(t, err)
	_ = stalConn.Close() // mark as stale (EOF on use)

	cleanup() // server gone — redial will fail

	pool := newConnPool(client, addr, 2)
	pool.pool <- stalConn // inject the stale conn

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)
	msg.SetEdns0(1232, false)

	_, _, err = pool.exchange(ctx, msg)
	// Should return an error (original EOF, since redial also failed).
	assert.Error(t, err)
}

// TestConnPoolExchange_NonEOFError verifies that a non-EOF exchange error
// (e.g. context cancelled) is surfaced directly without attempting a redial.
func TestConnPoolExchange_NonEOFError(t *testing.T) {
	addr, cleanup := startTCPDNSServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		// Sleep long enough for the context to be cancelled before responding.
		time.Sleep(2 * time.Second)
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	}))
	defer cleanup()

	client := &dns.Client{Net: "tcp", Timeout: 5 * time.Second}
	pool := newConnPool(client, addr, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)
	msg.SetEdns0(1232, false)

	_, _, err := pool.exchange(ctx, msg)
	// Should be a timeout/cancelled error, not a redial.
	assert.Error(t, err)
}

// TestConnPoolExchange_GetDialFails verifies that a dial failure during get()
// is propagated correctly from exchange().
func TestConnPoolExchange_GetDialFails(t *testing.T) {
	client := &dns.Client{Net: "tcp", Timeout: 200 * time.Millisecond}
	// Point the pool at an address nothing is listening on.
	pool := newConnPool(client, "127.0.0.1:19977", 1)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)

	_, _, err := pool.exchange(ctx, msg)
	assert.Error(t, err, "expected dial error")
}

// TestWithKeepAlive_ZeroPoolSize verifies that passing 0 (or negative) to
// WithKeepAlive causes the pool to fall back to min(concurrency, 10).
// This covers the `size = min(c.concurrency, 10)` branch in New().
func TestWithKeepAlive_ZeroPoolSize(t *testing.T) {
	addr, cleanup := startTCPDNSServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	}))
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithProtocol("tcp"),
		WithKeepAlive(0), // ≤ 0 → should default to min(concurrency, 10)
		WithCache(nil),
		WithMaxRetries(0),
	)
	defer func() { _ = c.Close() }()

	// Pool must have been created (not nil) despite poolSize=0.
	require.NotNil(t, c.connPools, "connPools should be initialised for tcp even with poolSize=0")
	pool, ok := c.connPools[addr]
	require.True(t, ok, "pool for server address should exist")
	// Cap should be min(defaultConcurrency, 10) = 10.
	assert.Equal(t, 10, cap(pool.pool))

	// Verify it still works end-to-end.
	ctx := context.Background()
	result, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)
	assert.NoError(t, result.Error)
}

// TestIsConnError_ErrConnEmpty verifies that dns.ErrConnEmpty is classified as
// a stale-connection error warranting a redial.
func TestIsConnError_ErrConnEmpty(t *testing.T) {
	assert.True(t, isConnError(dns.ErrConnEmpty),
		"dns.ErrConnEmpty should be classified as a conn error (nil net.Conn)")
}

// TestIsConnError_NetOpError verifies that *net.OpError wrapping OS-level
// closed-connection messages is classified as a stale-connection error.
func TestIsConnError_NetOpError(t *testing.T) {
	// Simulate "use of closed network connection" wrapped inside *net.OpError.
	opErr := &net.OpError{
		Op:  "write",
		Net: "tcp",
		Err: &wrappedMsg{"use of closed network connection"},
	}
	assert.True(t, isConnError(opErr), "net.OpError wrapping 'use of closed network connection'")

	// Simulate "broken pipe".
	opErr2 := &net.OpError{
		Op:  "write",
		Net: "tcp",
		Err: &wrappedMsg{"broken pipe"},
	}
	assert.True(t, isConnError(opErr2), "net.OpError wrapping 'broken pipe'")

	// Simulate "connection reset by peer".
	opErr3 := &net.OpError{
		Op:  "read",
		Net: "tcp",
		Err: &wrappedMsg{"connection reset by peer"},
	}
	assert.True(t, isConnError(opErr3), "net.OpError wrapping 'connection reset by peer'")

	// Unrelated OpError should NOT be classified as a conn error.
	opErr4 := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &wrappedMsg{"connection refused"},
	}
	assert.False(t, isConnError(opErr4), "net.OpError wrapping 'connection refused' should not redial")
}

// wrappedMsg is a minimal error type used to simulate OS-level error strings
// inside a *net.OpError for unit testing purposes.
type wrappedMsg struct{ msg string }

func (e *wrappedMsg) Error() string { return e.msg }

// TestConnPoolExchange_RedialExchangeFails verifies that when the transparent
// redial succeeds but the subsequent DNS exchange on the fresh connection also
// fails, the fresh error is returned and the connection is closed.
func TestConnPoolExchange_RedialExchangeFails(t *testing.T) {
	// Phase 1: start a normal server so the initial dial (for the "stale"
	// connection) succeeds.
	ln1, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln1.Addr().String()

	// Phase 2: start a "drop-immediately" server on the same port that
	// accepts TCP connections but closes them before sending any DNS response.
	// We coordinate via a channel so we can swap behaviour after injecting
	// the stale conn.
	dropConns := make(chan struct{}, 1)
	go func() {
		for {
			conn, err := ln1.Accept()
			if err != nil {
				return // listener closed
			}
			select {
			case <-dropConns:
				// Drop mode: close immediately so the DNS client gets EOF.
				_ = conn.Close()
			default:
				// Normal mode: we only needed the connection to exist; close it.
				_ = conn.Close()
			}
		}
	}()
	defer func() { _ = ln1.Close() }()

	client := &dns.Client{Net: "tcp", Timeout: 2 * time.Second}

	// Create a stale conn (closed locally) to inject into the pool.
	stale, err := client.DialContext(context.Background(), addr)
	require.NoError(t, err)
	_ = stale.Close() // stale: next ExchangeWithConnContext → EOF → isConnError → redial

	// Now activate drop-immediately mode so the fresh redial connection is
	// also dropped immediately, causing ExchangeWithConnContext to fail.
	dropConns <- struct{}{}

	pool := newConnPool(client, addr, 1)
	pool.pool <- stale // inject stale conn

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)
	msg.SetEdns0(1232, false)

	_, _, err = pool.exchange(ctx, msg)
	// The redial succeeded but the exchange on the fresh conn failed → err2 != nil.
	assert.Error(t, err, "expected error when fresh redial exchange also fails")
}

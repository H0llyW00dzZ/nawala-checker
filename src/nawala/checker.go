// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// Default configuration values.
const (
	defaultTimeout     = 5 * time.Second
	defaultRetries     = 2
	defaultCacheTTL    = 5 * time.Minute
	defaultConcurrency = 100
	defaultEDNS0Size   = 1232 // Recommended size to prevent IP fragmentation

	// cacheKeyPrefix is prepended to every cache key to namespace all entries
	// produced by this SDK and avoid collisions with other packages that may
	// share the same cache backend.
	cacheKeyPrefix = "nawala_checker:"
)

// defaultServers are the pre-configured Nawala DNS servers
// with their blocking keywords and query types.
//
// Nawala blocks domains by returning CNAME redirects to known
// block pages: "internetpositif.id" or "internetsehatku.com".
// Komdigi blocks domains by returning an A record with
// EDE 15 (Blocked) containing "trustpositif.komdigi.go.id".
// The keyword is matched against the full DNS record string,
// so these domain names are used as detection keywords.
var defaultServers = []DNSServer{
	{Address: "180.131.144.144", Keyword: "internetpositif", QueryType: "A"},
	{Address: "180.131.145.145", Keyword: "internetpositif", QueryType: "A"},
}

// Checker performs DNS-based domain blocking checks against
// Nawala/Kominfo (now Komdigi) DNS servers.
type Checker struct {
	mu            sync.RWMutex
	servers       []DNSServer
	timeout       time.Duration
	maxRetries    int
	concurrency   int
	cache         Cache
	cacheSet      bool // true when WithCache was called explicitly (even with nil)
	cacheTTL      time.Duration
	edns0Size     uint16
	dnsProtocol   string // dns.Client.Net value: "udp", "tcp", or "tcp-tls"
	tlsServerName string // TLS SNI server name override (tcp-tls only)
	tlsSkipVerify bool   // skip TLS certificate verification (tcp-tls only)
	dnsClient     *dns.Client
	digestHash    func(data string) string // optional; when set, cache keys are digested
	keepAlive     bool                     // true when WithKeepAlive is configured
	poolSize      int                      // max idle conns per server in the pool
	connPools     map[string]*connPool     // keyed by server address; nil when keepAlive is false
}

// New creates a new [Checker] with the default Nawala DNS server
// configuration. Use functional options to customize behavior.
//
//	// Default configuration:
//	c := nawala.New()
//
//	// Custom configuration:
//	c := nawala.New(
//	    nawala.WithTimeout(10 * time.Second),
//	    nawala.WithMaxRetries(3),
//	)
func New(opts ...Option) *Checker {
	c := &Checker{
		servers:     make([]DNSServer, len(defaultServers)),
		timeout:     defaultTimeout,
		maxRetries:  defaultRetries,
		concurrency: defaultConcurrency,
		edns0Size:   defaultEDNS0Size,
		cacheTTL:    defaultCacheTTL,
		dnsProtocol: "udp",
	}
	copy(c.servers, defaultServers)

	for _, opt := range opts {
		opt(c)
	}

	// Initialize cache only when WithCache was not explicitly called.
	// If WithCache(nil) was called, cacheSet is true and cache stays nil (disabled).
	if !c.cacheSet {
		c.cache = newMemoryCache(c.cacheTTL)
	}

	// Initialize shared DNS client if not set by WithDNSClient option.
	if c.dnsClient == nil {
		client := &dns.Client{
			Timeout: c.timeout,
		}

		switch c.dnsProtocol {
		case "tcp-tls":
			// Build TLS config only for tcp-tls and only when explicitly requested,
			// so UDP/TCP paths have zero overhead.
			client.Net = "tcp-tls"
			client.TLSConfig = &tls.Config{
				ServerName:         c.tlsServerName,
				InsecureSkipVerify: c.tlsSkipVerify,
			}
		case "tcp":
			client.Net = "tcp"
		default:
			client.Net = "udp"
		}

		c.dnsClient = client
	}

	// Initialise connection pool for TCP / TCP-TLS when keep-alive is requested.
	// UDP is stateless so pooling is intentionally skipped.
	if c.keepAlive && (c.dnsProtocol == "tcp" || c.dnsProtocol == "tcp-tls") {
		size := c.poolSize
		if size <= 0 {
			size = min(c.concurrency, 10)
		}
		c.connPools = make(map[string]*connPool, len(c.servers))
		for _, srv := range c.servers {
			if _, exists := c.connPools[srv.Address]; !exists {
				c.connPools[srv.Address] = newConnPool(c.dnsClient, srv.Address, size)
			}
		}
	}

	return c
}

// Check checks multiple domains concurrently against the configured
// Nawala DNS servers. It returns a [Result] for each domain.
//
// The checker will automatically select an available DNS server,
// use caching, and retry on failures.
//
// Invalid domains are returned with [ErrInvalidDomain] in the
// Result's Error field.
//
// Domains that do not exist on the internet are returned with
// [ErrNXDOMAIN] in the Result's Error field.
func (c *Checker) Check(ctx context.Context, domains ...string) ([]Result, error) {
	c.mu.RLock()
	n := len(c.servers)
	c.mu.RUnlock()

	if n == 0 {
		return nil, ErrNoDNSServers
	}

	results := make([]Result, len(domains))
	var wg sync.WaitGroup

	// Semaphore to limit concurrency.
	// We use a buffered channel to limit the number
	// of concurrent goroutines.
	sem := make(chan struct{}, c.concurrency)

Loop:
	for i, domain := range domains {
		// Check context before starting new work
		select {
		case <-ctx.Done():
			// Fill remaining results with context error
			for j := i; j < len(domains); j++ {
				results[j] = Result{
					Domain: domains[j],
					Error:  ctx.Err(),
				}
			}
			// Do not return immediately! We must wait for active goroutines.
			// Break the loop to stop spawning new ones.
			break Loop
		default:
		}

		wg.Add(1)

		// Acquire semaphore before spawning goroutine to limit
		// the number of active goroutines.
		sem <- struct{}{}

		go func(idx int, d string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore
			defer func() {
				if r := recover(); r != nil {
					results[idx] = Result{
						Domain: d,
						Error:  fmt.Errorf("%w: %v", ErrInternalPanic, r),
					}
				}
			}()

			results[idx] = c.checkSingle(ctx, d)
		}(i, domain)
	}

	wg.Wait()
	// Check context one last time to return correct error if we broke early
	if ctx.Err() != nil {
		return results, ctx.Err()
	}
	return results, nil
}

// CheckOne checks a single domain against the configured Nawala DNS servers.
// This is a convenience wrapper around [Checker.Check].
func (c *Checker) CheckOne(ctx context.Context, domain string) (Result, error) {
	c.mu.RLock()
	n := len(c.servers)
	c.mu.RUnlock()

	if n == 0 {
		return Result{}, ErrNoDNSServers
	}
	return c.checkSingle(ctx, domain), nil
}

// Stream represents a bidirectional stream of domains and their check results.
type Stream struct {
	In  <-chan string
	Out chan<- Result
}

// CheckStream checks a stream of domains concurrently against the configured
// Nawala DNS servers. It reads domains from the [Stream.In] channel and sends
// the corresponding [Result] to the [Stream.Out] channel.
//
// The checker limits concurrency based on the configured concurrency limit.
// The function blocks until the 'In' channel is closed and all checks
// are complete. It does not close the 'Out' channel, giving callers
// the flexibility to multiplex multiple streams into a single output channel.
//
// If the context is canceled, it returns the context error immediately without
// processing remaining domains in the channel.
func (c *Checker) CheckStream(ctx context.Context, stream Stream) error {
	c.mu.RLock()
	n := len(c.servers)
	c.mu.RUnlock()

	if n == 0 {
		return ErrNoDNSServers
	}

	var wg sync.WaitGroup

	// Semaphore to limit concurrency.
	// We use a buffered channel to limit the number
	// of concurrent goroutines.
	sem := make(chan struct{}, c.concurrency)

Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case domain, ok := <-stream.In:
			if !ok {
				break Loop
			}

			wg.Add(1)

			// Acquire semaphore before spawning goroutine to limit
			// the number of active goroutines.
			sem <- struct{}{}

			go func(d string) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				var res Result
				defer func() {
					if r := recover(); r != nil {
						res = Result{
							Domain: d,
							Error:  fmt.Errorf("%w: %v", ErrInternalPanic, r),
						}
						// Send panic result, respecting context cancellation
						select {
						case <-ctx.Done():
						case stream.Out <- res:
						}
					}
				}()

				res = c.checkSingle(ctx, d)
				// Send result, respecting context cancellation
				select {
				case <-ctx.Done():
				case stream.Out <- res:
				}
			}(domain)
		}
	}

	wg.Wait()
	return ctx.Err()
}

// DNSStatus checks the health of all configured DNS servers.
// It returns the online/offline status and latency for each server.
func (c *Checker) DNSStatus(ctx context.Context) ([]ServerStatus, error) {
	c.mu.RLock()
	servers := make([]DNSServer, len(c.servers))
	copy(servers, c.servers)
	c.mu.RUnlock()

	if len(servers) == 0 {
		return nil, ErrNoDNSServers
	}

	statuses := make([]ServerStatus, len(servers))
	var wg sync.WaitGroup

	// Semaphore to limit concurrency.
	// We use a buffered channel to limit the number
	// of concurrent goroutines.
	sem := make(chan struct{}, c.concurrency)

Loop:
	for i, srv := range servers {
		// Check context before starting new work
		select {
		case <-ctx.Done():
			// Fill remaining results with context error
			for j := i; j < len(servers); j++ {
				statuses[j] = ServerStatus{
					Server: servers[j].Address,
					Error:  ctx.Err(),
				}
			}
			break Loop
		default:
		}

		wg.Add(1)

		// Acquire semaphore before spawning goroutine.
		sem <- struct{}{}

		go func(idx int, server DNSServer) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore
			defer func() {
				if r := recover(); r != nil {
					statuses[idx] = ServerStatus{
						Server: server.Address,
						Error:  fmt.Errorf("%w: %v", ErrInternalPanic, r),
					}
				}
			}()

			statuses[idx] = checkDNSHealth(ctx, dnsQuery{
				client:    c.dnsClient,
				pool:      c.connPools[server.Address],
				server:    server.Address,
				edns0Size: c.edns0Size,
			})
		}(i, srv)
	}

	wg.Wait()
	if ctx.Err() != nil {
		return statuses, ctx.Err()
	}
	return statuses, nil
}

// Close releases resources held by the checker — specifically it drains and
// closes all idle connections in the keep-alive pool, if one was configured
// via [WithKeepAlive].
//
// Callers using the default UDP protocol or without [WithKeepAlive] do not
// need to call Close; it is a no-op in those cases.
func (c *Checker) Close() error {
	for _, p := range c.connPools {
		p.close()
	}
	return nil
}

// FlushCache clears all cached DNS check results.
func (c *Checker) FlushCache() {
	if c.cache != nil {
		c.cache.Flush()
	}
}

// Servers returns a copy of the currently configured DNS servers.
func (c *Checker) Servers() []DNSServer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	servers := make([]DNSServer, len(c.servers))
	copy(servers, c.servers)
	return servers
}

// checkSingle performs the DNS check for a single domain.
// It handles normalization, validation, caching, and failover.
func (c *Checker) checkSingle(ctx context.Context, domain string) Result {
	domain = normalizeDomain(domain)

	if !IsValidDomain(domain) {
		return Result{
			Domain: domain,
			Error:  fmt.Errorf("%w: %s", ErrInvalidDomain, domain),
		}
	}

	// Snapshot the server list under a read lock so that a concurrent
	// SetServers call cannot modify the slice mid-iteration.
	c.mu.RLock()
	servers := make([]DNSServer, len(c.servers))
	copy(servers, c.servers)
	c.mu.RUnlock()

	// Try each server in order (primary with failover).
	for _, srv := range servers {
		qtype := parseQueryType(srv.QueryType)
		// Cache key deliberately includes the server address; different
		// servers may return different blocking verdicts for the same domain
		// (e.g., only one resolver applies a block list). This trades a lower
		// cache hit rate for correctness — a cached "not blocked" from server A
		// must not short-circuit a probe against server B.
		//
		// All keys are prefixed with cacheKeyPrefix to namespace SDK entries
		// from other packages that may share the same cache backend.
		// When WithDigests is configured, the raw components are hashed first
		// and the digest itself becomes the key body (e.g. nawala_checker:<digest>).
		rawKey := fmt.Sprintf("%s:%s:%s:%d", domain, srv.Address, srv.Keyword, qtype)
		var cacheKey string
		if c.digestHash != nil {
			cacheKey = cacheKeyPrefix + c.digestHash(rawKey)
		} else {
			cacheKey = cacheKeyPrefix + rawKey
		}

		// Check cache first.
		if c.cache != nil {
			if cached, ok := c.cache.Get(cacheKey); ok {
				return cached
			}
		}

		// Attempt DNS query with retries.
		result, err := c.queryWithRetries(ctx, domain, srv, qtype)
		if err != nil {
			// If the domain strictly does not exist (NXDOMAIN) or query rejected by server (QueryRejected), return immediately.
			// This is a definitive answer from the DNS server, so we shouldn't failover over it.
			if errors.Is(err, ErrNXDOMAIN) || errors.Is(err, ErrQueryRejected) {
				return Result{
					Domain: domain,
					Server: srv.Address,
					Error:  err,
				}
			}
			// Other errors (timeouts, network issues), try next server.
			continue
		}

		// Cache the result.
		if c.cache != nil {
			c.cache.Set(cacheKey, result)
		}

		return result
	}

	// All servers failed.
	return Result{
		Domain: domain,
		Error:  ErrAllDNSFailed,
	}
}

// queryWithRetries sends a DNS query with retry logic.
//
// Because Nawala/Kominfo (now Komdigi) DNS servers can return inconsistent responses
// (the blocking CNAME may appear intermittently), this function
// probes the server multiple times. If ANY probe detects blocking,
// it returns immediately with Blocked=true. Only after all probes
// return non-blocked does it report the domain as not blocked.
//
// Exponential backoff is applied only after query errors, not
// between successful probes.
func (c *Checker) queryWithRetries(ctx context.Context, domain string, srv DNSServer, qtype uint16) (Result, error) {
	var (
		lastErr    error
		bestResult Result
		responded  bool
	)

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 && lastErr != nil {
			// Exponential backoff only after errors: 1s, 2s, 4s, ...
			backoff := min(
				// Cap backoff to prevent overflow or excessive waits.
				time.Duration(1<<uint(attempt-1))*time.Second, 30*time.Second)

			select {
			case <-ctx.Done():
				return Result{}, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := queryDNS(ctx, dnsQuery{
			client:    c.dnsClient,
			pool:      c.connPools[srv.Address],
			domain:    domain,
			server:    srv.Address,
			qtype:     qtype,
			edns0Size: c.edns0Size,
		})
		if err != nil {
			// If the domain strictly does not exist, or the server explicitly rejected the query, do not retry.
			if errors.Is(err, ErrNXDOMAIN) || errors.Is(err, ErrQueryRejected) {
				return Result{}, err
			}

			lastErr = err
			continue
		}

		// If blocking detected on any probe, return immediately.
		if containsKeyword(resp, srv.Keyword) {
			return Result{
				Domain:  domain,
				Blocked: true,
				Server:  srv.Address,
			}, nil
		}

		// Track first successful non-blocked result.
		if !responded {
			bestResult = Result{
				Domain:  domain,
				Blocked: false,
				Server:  srv.Address,
			}
			responded = true
		}
	}

	// All probes succeeded without detecting blocking.
	if responded {
		return bestResult, nil
	}

	return Result{}, lastErr
}

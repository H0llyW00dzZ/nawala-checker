// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
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
	servers     []DNSServer
	timeout     time.Duration
	maxRetries  int
	concurrency int
	cache       Cache
	cacheTTL    time.Duration
	edns0Size   uint16
	dnsClient   *dns.Client
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
	}
	copy(c.servers, defaultServers)

	for _, opt := range opts {
		opt(c)
	}

	// Initialize cache if not set by option.
	if c.cache == nil {
		c.cache = newMemoryCache(c.cacheTTL)
	}

	// Initialize shared DNS client if not set by WithDNSClient option.
	if c.dnsClient == nil {
		c.dnsClient = &dns.Client{
			Timeout: c.timeout,
			Net:     "udp",
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
func (c *Checker) Check(ctx context.Context, domains ...string) ([]Result, error) {
	if len(c.servers) == 0 {
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
	if len(c.servers) == 0 {
		return Result{}, ErrNoDNSServers
	}
	return c.checkSingle(ctx, domain), nil
}

// DNSStatus checks the health of all configured DNS servers.
// It returns the online/offline status and latency for each server.
func (c *Checker) DNSStatus(ctx context.Context) ([]ServerStatus, error) {
	if len(c.servers) == 0 {
		return nil, ErrNoDNSServers
	}

	statuses := make([]ServerStatus, len(c.servers))
	var wg sync.WaitGroup

	// Semaphore to limit concurrency.
	// We use a buffered channel to limit the number
	// of concurrent goroutines.
	sem := make(chan struct{}, c.concurrency)

Loop:
	for i, srv := range c.servers {
		// Check context before starting new work
		select {
		case <-ctx.Done():
			// Fill remaining results with context error
			for j := i; j < len(c.servers); j++ {
				statuses[j] = ServerStatus{
					Server: c.servers[j].Address,
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

			statuses[idx] = checkDNSHealth(ctx, c.dnsClient, server.Address, c.edns0Size)
		}(i, srv)
	}

	wg.Wait()
	if ctx.Err() != nil {
		return statuses, ctx.Err()
	}
	return statuses, nil
}

// FlushCache clears all cached DNS check results.
func (c *Checker) FlushCache() {
	if c.cache != nil {
		c.cache.Flush()
	}
}

// Servers returns a copy of the currently configured DNS servers.
func (c *Checker) Servers() []DNSServer {
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

	// Try each server in order (primary with failover).
	for _, srv := range c.servers {
		qtype := parseQueryType(srv.QueryType)
		cacheKey := fmt.Sprintf("%s:%s:%s:%d", domain, srv.Address, srv.Keyword, qtype)

		// Check cache first.
		if c.cache != nil {
			if cached, ok := c.cache.Get(cacheKey); ok {
				return cached
			}
		}

		// Attempt DNS query with retries.
		result, err := c.queryWithRetries(ctx, domain, srv, qtype)
		if err != nil {
			// This server failed, try next.
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

		resp, err := queryDNS(ctx, c.dnsClient, domain, srv.Address, qtype, c.edns0Size)
		if err != nil {
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

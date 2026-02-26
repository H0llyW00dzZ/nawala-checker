// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package nawala provides a DNS-based domain blocking checker for
// Indonesian ISP DNS filters (Nawala/Kominfo (now Komdigi)).
//
// It works by querying configurable DNS servers and scanning the responses
// for blocking keywords such as "internetpositif.id" redirects or
// "trustpositif.komdigi.go.id" EDE indicators.
//
// # Network Requirement
//
// This package requires an Indonesian network to function correctly.
// Nawala DNS servers only return blocking responses when queried from
// within Indonesia. Ensure your connection uses a pure Indonesian IP
// with no routing through networks outside Indonesia.
//
// When running on cloud infrastructure (e.g., VPS, microservices, [k8s])
// that is not on an Indonesian network (e.g., a Singapore or US server),
// implement your own DNS server on an Indonesian network, then point this
// SDK at it using [WithServers]. Block indicator behavior depends on the
// DNS server in use; the default Nawala/Komdigi servers only return block
// indicators when queried from an Indonesian source IP.
//
// # General-Purpose Design
//
// Despite rumors that the original Nawala project may cease operations,
// this module remains a general-purpose DNS checking toolkit built from
// the ground up with customizable DNS server and client configurations.
// You can point it at any DNS server, define your own blocking keywords,
// and plug in custom [dns.Client] instances (TCP, DNS-over-TLS, custom
// dialers, etc.). The default Nawala servers are simply pre-configured
// defaults; the SDK itself is fully independent and actively maintained.
//
// # Features
//
//   - Concurrent domain checking — check multiple domains in parallel
//     with a single call
//   - DNS server failover — automatic fallback to secondary servers when
//     the primary fails
//   - Retry with exponential backoff — resilient against transient
//     network errors
//   - Built-in caching — in-memory cache with configurable TTL to avoid
//     redundant queries
//   - Custom cache backends — plug in Redis, memcached, or any backend
//     via the Cache interface
//   - Server health checks — monitor online/offline status and latency
//     of DNS servers
//   - Panic recovery — goroutines are protected from panics with
//     automatic recovery and typed errors
//   - Functional options — clean, [idiomatic Go] configuration pattern
//   - Context-aware — full support for timeouts and cancellation via
//     [context.Context]
//   - Domain validation — automatic normalization and validation of
//     domain names
//   - Typed errors — sentinel errors for [errors.Is] matching
//
// # Quick Start
//
//	c := nawala.New()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	results, err := c.Check(ctx, "google.com", "reddit.com", "github.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, r := range results {
//	    status := "not blocked"
//	    if r.Blocked {
//	        status = "BLOCKED"
//	    }
//	    if r.Error != nil {
//	        status = fmt.Sprintf("error: %v", r.Error)
//	    }
//	    fmt.Printf("%-20s %s (server: %s)\n", r.Domain, status, r.Server)
//	}
//
// # Configuration
//
// Use functional options to customize the checker:
//
//	c := nawala.New(
//	    // Increase timeout for slow networks.
//	    nawala.WithTimeout(15 * time.Second),
//
//	    // Allow more retries (3 retries = 4 total attempts).
//	    nawala.WithMaxRetries(3),
//
//	    // Cache results for 10 minutes.
//	    nawala.WithCacheTTL(10 * time.Minute),
//
//	    // Replace all DNS servers with custom ones.
//	    nawala.WithServers([]nawala.DNSServer{
//	        {Address: "8.8.8.8", Keyword: "blocked", QueryType: "A"},
//	        {Address: "8.8.4.4", Keyword: "blocked", QueryType: "A"},
//	    }),
//
//	    // Limit concurrent checks to 50 goroutines.
//	    nawala.WithConcurrency(50),
//
//	    // Use a custom DNS client (e.g., TCP or DNS-over-TLS).
//	    nawala.WithDNSClient(&dns.Client{
//	        Timeout: 10 * time.Second,
//	        Net:     "tcp-tls",
//	    }),
//
//	    // Set custom EDNS0 size (default is 1232 to prevent fragmentation).
//	    nawala.WithEDNS0Size(4096),
//	)
//
// Available options:
//
//   - [WithTimeout]     — Timeout per DNS query (default: 5s)
//   - [WithMaxRetries]  — Max retry attempts per query, total = n+1 (default: 2)
//   - [WithCacheTTL]    — TTL for the built-in in-memory cache (default: 5m)
//   - [WithCache]       — Custom Cache implementation; pass nil to disable
//   - [WithConcurrency] — Max concurrent DNS checks, semaphore size (default: 100)
//   - [WithEDNS0Size]   — EDNS0 UDP buffer size, prevents fragmentation (default: 1232)
//   - [WithDNSClient]   — Custom client for TCP, TLS, or custom dialer
//   - [WithServer]      — (Deprecated: use [Checker.SetServers] for hot-reloading) Add or replace a single DNS server
//   - [WithServers]     — Replace all DNS servers (default: Nawala servers)
//   - [Checker.SetServers] — Hot-reload: Add or replace servers at runtime safely
//
// # API
//
// Core methods on [Checker]:
//
//	// Check multiple domains concurrently.
//	results, err := c.Check(ctx, "example.com", "another.com")
//
//	// Check a single domain.
//	result, err := c.CheckOne(ctx, "example.com")
//
//	// Check DNS server health and latency.
//	statuses, err := c.DNSStatus(ctx)
//
//	// Clear the result cache.
//	c.FlushCache()
//
//	// Get configured servers.
//	servers := c.Servers()
//
//	// Hot-reload: Add or replace servers at runtime (concurrency-safe).
//	c.SetServers(nawala.DNSServer{
//	    Address:   "203.0.113.1",
//	    Keyword:   "blocked",
//	    QueryType: "A",
//	})
//
// Domain validation:
//
//	ok := nawala.IsValidDomain("example.com") // true
//	ok  = nawala.IsValidDomain("invalid")     // false (single label, no TLD)
//
// # Errors
//
// Sentinel errors for use with [errors.Is]:
//
//	var (
//	    ErrNoDNSServers  // No DNS servers configured
//	    ErrAllDNSFailed  // All DNS servers failed to respond
//	    ErrInvalidDomain // Domain name failed validation
//	    ErrDNSTimeout    // DNS query exceeded the configured timeout
//	    ErrInternalPanic // An internal panic was recovered during execution
//	    ErrNXDOMAIN      // Domain does not exist (NXDOMAIN)
//	    ErrQueryRejected // Query explicitly rejected by server (Format Error, Refused, Not Implemented)
//	)
//
// # Custom Cache
//
// Implement the Cache interface to plug in a custom backend such as
// Redis or memcached:
//
//	type Cache interface {
//	    Get(key string) (Result, bool)
//	    Set(key string, val Result)
//	    Flush()
//	}
//
// Pass a nil value to WithCache to disable caching entirely.
//
// # How Blocking Works
//
// Indonesian ISP DNS filters use two distinct blocking mechanisms.
//
// Nawala — CNAME Redirect:
//
// Nawala intercepts DNS queries for blocked domains and returns a CNAME
// redirect to a landing page instead of the real IP address:
//
//	;; ANSWER SECTION:
//	blocked.example.    3600    IN    CNAME    internetpositif.id.
//
// The checker detects this by scanning all DNS record sections (Answer,
// Authority, Additional) for the keyword "internetpositif" in any
// record's string representation.
//
// Komdigi — EDE 15 (Blocked):
//
// Komdigi uses the Extended DNS Errors mechanism (RFC 8914). The
// response returns an A record pointing to a block page IP, along with
// an EDE option code 15 (Blocked) in the OPT pseudo-section:
//
//	;; OPT PSEUDOSECTION:
//	; EDE: 15 (Blocked): (source=block-list-zone;
//	;   blockListUrl=https://trustpositif.komdigi.go.id/assets/db/domains_isp;
//	;   domain=reddit.com)
//
//	;; ANSWER SECTION:
//	reddit.com.    30    IN    A    103.155.26.29
//
// The checker detects this by scanning the Extra section (which contains
// the OPT record) for the keyword "trustpositif" or "komdigi". To use
// this detection, configure a server with the appropriate keyword:
//
//	nawala.WithServers([]nawala.DNSServer{
//	    {Address: "103.155.26.28", Keyword: "trustpositif", QueryType: "A"},
//	    {Address: "103.155.26.29", Keyword: "komdigi",      QueryType: "A"},
//	})
//
// # Default DNS Servers
//
// The checker comes pre-configured with known Nawala DNS servers:
//
//	Address            Keyword           QueryType
//	180.131.144.144    internetpositif   A
//	180.131.145.145    internetpositif   A
//
// Nawala blocks domains by returning CNAME redirects to known block
// pages (internetpositif.id or internetsehatku.com). Komdigi blocks
// domains by returning an A record with EDE 15 (Blocked) containing
// trustpositif.komdigi.go.id. The keyword is matched against the full
// DNS record string for broad detection.
//
// # Examples
//
// Runnable examples are available in the [examples/] directory:
//
//   - [examples/basic]  — Check multiple domains with default configuration
//   - [examples/custom] — Advanced configuration with custom servers,
//     timeouts, retries, and caching
//   - [examples/status] — Monitor DNS server health and latency
//
// Run an example (requires cloning the repository first):
//
//	git clone https://github.com/H0llyW00dzZ/nawala-checker.git
//	cd nawala-checker
//	go run ./examples/basic
//
// [examples/basic]: https://github.com/H0llyW00dzZ/nawala-checker/blob/master/examples/basic/main.go
// [examples/custom]: https://github.com/H0llyW00dzZ/nawala-checker/blob/master/examples/custom/main.go
// [examples/status]: https://github.com/H0llyW00dzZ/nawala-checker/blob/master/examples/status/main.go
// [examples/]: https://github.com/H0llyW00dzZ/nawala-checker/blob/master/examples
// [k8s]: https://kubernetes.io
// [idiomatic Go]: https://go.dev/doc/effective_go
package nawala

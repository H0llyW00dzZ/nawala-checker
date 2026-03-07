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
// # CLI
//
// A command-line interface is bundled in this repository at cmd/nawala.
// Install it with:
//
//	go install github.com/H0llyW00dzZ/nawala-checker/cmd/nawala@latest
//
// Run nawala --help for usage information.
//
// # Features
//
//   - Concurrent domain checking — check multiple domains in parallel
//     with a single call
//   - Streaming domain checking — process domains through a channel
//     pipeline via [Checker.CheckStream], enabling constant-memory
//     operation even with millions of domains
//   - DNS server failover — automatic fallback to secondary servers when
//     the primary fails
//   - Retry with exponential backoff — resilient against transient
//     network errors
//   - Built-in caching — in-memory cache with configurable TTL to avoid
//     redundant queries
//   - Custom cache backends — plug in Redis, memcached, or any backend
//     via the Cache interface
//   - Cache key namespacing — all keys are prefixed with "nawala_checker:"
//     to prevent collisions when a cache backend is shared between packages
//   - Digest-based cache keys — optional [WithDigests] option to replace
//     the plain cache key body with any hash function, producing keys of
//     the form "nawala_checker:<digest>" (e.g. hex-encoded SHA-256)
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
//   - Keep-alive connection pool — optional [WithKeepAlive] option for
//     persistent TCP/TLS connections, avoiding per-query handshake overhead
//   - Runtime hot-reload — safely add, replace, or remove DNS servers
//     concurrently without restarting
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
//	    // Address accepts: plain IP, IP:port, hostname, or hostname:port.
//	    // Port defaults to 53 (UDP/TCP) or 853 (tcp-tls) when omitted.
//	    nawala.WithServers([]nawala.DNSServer{
//	        {Address: "8.8.8.8",            Keyword: "blocked", QueryType: "A"},
//	        {Address: "8.8.4.4:5353",        Keyword: "blocked", QueryType: "A"},
//	        {Address: "dns.example.com",     Keyword: "blocked", QueryType: "A"},
//	        {Address: "dns.example.com:853", Keyword: "blocked", QueryType: "A"},
//	    }),
//
//	    // Limit concurrent checks to 50 goroutines.
//	    nawala.WithConcurrency(50),
//
//	    // Use a custom DNS client (e.g., custom dialer or advanced TLS).
//	    nawala.WithDNSClient(&dns.Client{
//	        Timeout: 10 * time.Second,
//	        Net:     "tcp-tls",
//	    }),
//
//	    // Or simply switch protocol without replacing the client.
//	    nawala.WithProtocol("tcp-tls"),
//
//	    // Set custom EDNS0 size (default is 1232 to prevent fragmentation).
//	    nawala.WithEDNS0Size(4096),
//
//	    // Digest-based cache keys: replace the plain key body with a hash
//	    // so that long or sensitive values are never stored verbatim.
//	    // The final key is always: "nawala_checker:<digest>".
//	    nawala.WithDigests(func(data string) string {
//	        sum := sha256.Sum256([]byte(data))
//	        return hex.EncodeToString(sum[:])
//	    }),
//	)
//
// Available options:
//
//   - [WithTimeout]           — Timeout per DNS query (default: 5s)
//   - [WithMaxRetries]        — Max retry attempts per query, total = n+1 (default: 2)
//   - [WithCacheTTL]          — TTL for the built-in in-memory cache (default: 5m)
//   - [WithCache]             — Custom Cache implementation; pass nil to disable
//   - [WithDigests]           — Digest-based cache keys via a custom hash function;
//     key format: "nawala_checker:<digest>" (e.g. hex SHA-256); pass nil to disable
//   - [WithConcurrency]       — Max concurrent DNS checks, semaphore size (default: 100)
//   - [WithEDNS0Size]         — EDNS0 UDP buffer size, prevents fragmentation (default: 1232)
//   - [WithProtocol]          — DNS transport: "udp" (default), "tcp", or "tcp-tls" (DoT)
//   - [WithTLSServerName]     — SNI server name for tcp-tls; required when the server address is
//     an IP and the cert is issued for a hostname (works with trusted CA certs; set
//     tls_skip_verify: false for full verification)
//   - [WithTLSSkipVerify]     — Disable TLS cert verification for tcp-tls (only for self-signed
//     certs where no valid server name can be provided; never use in production)
//   - [WithDNSClient]         — Custom client for full transport control (TCP, TLS, dialer)
//   - [WithServer]            — (Deprecated: use [Checker.SetServers] for hot-reloading) Add or replace a single DNS server
//   - [WithServers]           — Replace all DNS servers (default: Nawala servers)
//   - [Checker.SetServers]    — Hot-reload: Add or replace servers at runtime safely
//   - [Checker.HasServer]     — Hot-reload: Check if a server is configured at runtime safely
//   - [Checker.DeleteServers] — Hot-reload: Remove servers at runtime safely
//   - [Checker.Concurrency]   — Returns the configured concurrency limit (semaphore size);
//     useful for sizing output channel buffers to match in-flight capacity
//   - [WithKeepAlive]         — Persistent TCP/TLS conn pool (idle conns per server);
//     no-op for UDP; requires [RFC 7766] (tcp) or [RFC 7858] (tcp-tls) server support —
//     use with DoT providers or modern custom resolvers, NOT the default Nawala
//     ISP servers (UDP-optimised, close TCP after each query); call [Checker.Close] when done
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
//	// Stream-check domains through a channel pipeline.
//	// Domains are read from In and results are sent to Out as they complete.
//	// Memory usage stays constant regardless of input size.
//	in := make(chan string)
//	out := make(chan nawala.Result, c.Concurrency())
//	go func() {
//	    for _, d := range domains { in <- d }
//	    close(in)
//	}()
//	err := c.CheckStream(ctx, nawala.Stream{In: in, Out: out})
//
//	// Read the configured concurrency (semaphore size).
//	n := c.Concurrency()
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
//	// Check if a server is currently configured.
//	if c.HasServer("203.0.113.1") {
//	    fmt.Println("Server is active")
//	}
//
//	// Hot-reload: Remove servers at runtime by address (concurrency-safe).
//	// The address must match exactly the value used in [DNSServer.Address]
//	// (the same format: plain IP, IP:port, hostname, or hostname:port).
//	c.DeleteServers("203.0.113.1")
//
//	// Release idle keep-alive connections (call when checker is no longer needed).
//	defer c.Close()
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
// # Cache Key Format
//
// All cache keys produced by this SDK are namespaced with the prefix
// "nawala_checker:" to prevent collisions when multiple packages share the
// same cache backend (e.g., Redis). The default key format is:
//
//	nawala_checker:<domain>:<server>:<keyword>:<qtype>
//
// When [WithDigests] is configured, the raw components are passed to the
// provided hash function and the returned string becomes the key body:
//
//	nawala_checker:<digest>
//
// Example — SHA-256 digested keys:
//
//	import (
//	    "crypto/sha256"
//	    "encoding/hex"
//	)
//
//	c := nawala.New(
//	    nawala.WithDigests(func(data string) string {
//	        sum := sha256.Sum256([]byte(data))
//	        return hex.EncodeToString(sum[:])
//	    }),
//	)
//
// Use [WithDigests] when:
//
//   - The cache backend enforces a maximum key length.
//   - Internal server addresses must not appear in cache keys in plain text.
//   - A consistent, fixed-width key format is required (e.g., 64-char hex).
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
//   - [examples/hotreload] — Safely update DNS servers while concurrent
//     checks are running
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
// [examples/hotreload]: https://github.com/H0llyW00dzZ/nawala-checker/blob/master/examples/hotreload/main.go
// [examples/]: https://github.com/H0llyW00dzZ/nawala-checker/blob/master/examples
// [k8s]: https://kubernetes.io
// [idiomatic Go]: https://go.dev/doc/effective_go
// [RFC 7766]: https://www.rfc-editor.org/rfc/rfc7766.html
// [RFC 7858]: https://www.rfc-editor.org/rfc/rfc7858.html
package nawala

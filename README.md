# 🌐 Nawala Checker

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25.6-blue?logo=go)](https://go.dev/dl/)
[![Go Reference](https://pkg.go.dev/badge/github.com/H0llyW00dzZ/nawala-checker.svg)](https://pkg.go.dev/github.com/H0llyW00dzZ/nawala-checker)
[![Go Report Card](https://goreportcard.com/badge/github.com/H0llyW00dzZ/nawala-checker)](https://goreportcard.com/report/github.com/H0llyW00dzZ/nawala-checker)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/H0llyW00dzZ/nawala-checker/graph/badge.svg?token=3GU9QRYAUX)](https://codecov.io/gh/H0llyW00dzZ/nawala-checker)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/H0llyW00dzZ/nawala-checker)
[![Baca dalam Bahasa Indonesia](https://img.shields.io/badge/🇮🇩-Baca%20dalam%20Bahasa%20Indonesia-red)](README.id.md)

A Go SDK for checking whether domains are blocked by Indonesian ISP DNS filters (Nawala/Kominfo (now Komdigi)). It works by querying configurable DNS servers and scanning the responses for blocking keywords such as `internetpositif.id` redirects or `trustpositif.komdigi.go.id` EDE indicators.

> [!IMPORTANT]
> This SDK requires an **Indonesian network** to function correctly. Nawala DNS servers only return blocking responses when queried from within Indonesia. Ensure your connection uses a pure Indonesian IP with no routing through networks outside Indonesia.

> [!NOTE]
> **This SDK is not deprecated or outdated.** Despite rumors that the original Nawala project may cease operations, this module remains a **general-purpose DNS checking toolkit** built from the ground up with customizable DNS server and client configurations. You can point it at any DNS server, define your own blocking keywords, and plug in custom `*dns.Client` instances (TCP, DNS-over-TLS, custom dialers, etc.). The default Nawala servers are simply pre-configured defaults; the SDK itself is fully independent and actively maintained.

> [!TIP]
> If running on cloud infrastructure (e.g., VPS, microservices, k8s), it's better to implement your own DNS server using an Indonesian network, then from the cloud infrastructure it just calls it.

## Features

- **Concurrent domain checking** — check multiple domains in parallel with a single call
- **DNS server failover** — automatic fallback to secondary servers when the primary fails
- **Retry with exponential backoff** — resilient against transient network errors
- **Built-in caching** — in-memory cache with configurable TTL to avoid redundant queries
- **Custom cache backends** — plug in Redis, memcached, or any backend via the `Cache` interface
- **Server health checks** — monitor online/offline status and latency of DNS servers
- **Panic recovery** — goroutines are protected from panics with automatic recovery and typed errors
- **Functional options** — clean, [idiomatic Go](https://go.dev/doc/effective_go) configuration pattern
- **Context-aware** — full support for timeouts and cancellation via `context.Context`
- **Domain validation** — automatic normalization and validation of domain names
- **Typed errors** — sentinel errors for `errors.Is` matching (`ErrNoDNSServers`, `ErrAllDNSFailed`, `ErrInvalidDomain`, `ErrDNSTimeout`, `ErrInternalPanic`)

## Installation

```bash
go get github.com/H0llyW00dzZ/nawala-checker
```

Requires **Go 1.25.6** or later.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

func main() {
    c := nawala.New()

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    results, err := c.Check(ctx, "google.com", "reddit.com", "github.com", "exam_ple.com")
    if err != nil {
        log.Fatal(err)
    }

    for _, r := range results {
        status := "not blocked"
        if r.Blocked {
            status = "BLOCKED"
        }
        if r.Error != nil {
            status = fmt.Sprintf("error: %v", r.Error)
        }
        fmt.Printf("%-20s %s (server: %s)\n", r.Domain, status, r.Server)
    }
}
```

## Configuration

Use functional options to customize the checker:

```go
c := nawala.New(
    // Increase timeout for slow networks.
    nawala.WithTimeout(15 * time.Second),

    // Allow more retries (3 retries = 4 total attempts).
    nawala.WithMaxRetries(3),

    // Cache results for 10 minutes.
    nawala.WithCacheTTL(10 * time.Minute),

    // Replace all DNS servers with custom ones.
    nawala.WithServers([]nawala.DNSServer{
        {
            Address:   "8.8.8.8",
            Keyword:   "blocked",
            QueryType: "A",
        },
        {
            Address:   "8.8.4.4",
            Keyword:   "blocked",
            QueryType: "A",
        },
    }),

    // Limit concurrent checks to 50 goroutines.
    nawala.WithConcurrency(50),

    // Use a custom DNS client (e.g., for TCP or DNS-over-TLS).
    nawala.WithDNSClient(&dns.Client{
        Timeout: 10 * time.Second,
        Net:     "tcp-tls",
    }),

    // Set custom EDNS0 size (default is 1232 to prevent fragmentation).
    nawala.WithEDNS0Size(4096),
)
```

### Available Options

| Option | Default | Description |
|---|---|---|
| `WithTimeout(d)` | `5s` | Timeout for each DNS query |
| `WithMaxRetries(n)` | `2` | Max retry attempts per query (total = n+1) |
| `WithCacheTTL(d)` | `5m` | TTL for the built-in in-memory cache |
| `WithCache(c)` | in-memory | Custom `Cache` implementation (pass `nil` to disable) |
| `WithConcurrency(n)` | `100` | Max concurrent DNS checks (semaphore size) |
| `WithEDNS0Size(n)` | `1232` | EDNS0 UDP buffer size (prevents fragmentation) |
| `WithDNSClient(c)` | UDP client | Custom `*dns.Client` for TCP, TLS, or custom dialer |
| `WithServer(s)` | — | Add or replace a single DNS server |
| `WithServers(s)` | Nawala defaults | Replace all DNS servers |

## API

### Core Methods

```go
// Check multiple domains concurrently.
results, err := c.Check(ctx, "example.com", "another.com")

// Check a single domain.
result, err := c.CheckOne(ctx, "example.com")

// Check DNS server health and latency.
statuses, err := c.DNSStatus(ctx)

// Clear the result cache.
c.FlushCache()

// Get configured servers.
servers := c.Servers()
```

### Validation

```go
// Validate a domain name before checking.
ok := nawala.IsValidDomain("example.com") // true
ok  = nawala.IsValidDomain("invalid")     // false (single label, no TLD)
```

### Types

```go
// Result of checking a single domain.
type Result struct {
    Domain  string  // The domain that was checked
    Blocked bool    // Whether the domain is blocked
    Server  string  // DNS server IP used for the check
    Error   error   // Non-nil if the check failed
}

// Health status of a DNS server.
type ServerStatus struct {
    Server    string  // DNS server IP address
    Online    bool    // Whether the server is responding
    LatencyMs int64   // Round-trip time in milliseconds
    Error     error   // Non-nil if the health check failed
}

// DNS server configuration.
type DNSServer struct {
    Address   string  // IP address of the DNS server
    Keyword   string  // Blocking keyword to search for in responses
    QueryType string  // DNS record type: "A", "AAAA", "CNAME", "TXT", etc.
}
```

### Errors

```go
var (
    ErrNoDNSServers  // No DNS servers configured
    ErrAllDNSFailed  // All DNS servers failed to respond
    ErrInvalidDomain // Domain name failed validation
    ErrDNSTimeout    // DNS query exceeded the configured timeout
    ErrInternalPanic // An internal panic was recovered during execution
    ErrNXDOMAIN      // Domain does not exist (NXDOMAIN)
)
```

### Custom Cache

Implement the `Cache` interface to use a custom cache backend:

```go
type Cache interface {
    Get(key string) (Result, bool)
    Set(key string, val Result)
    Flush()
}
```

## Examples

Runnable examples are available in the [`examples/`](examples/) directory:

| Example | Description |
|---|---|
| [`basic`](examples/basic) | Check multiple domains with default configuration |
| [`custom`](examples/custom) | Advanced configuration with custom servers, timeouts, retries, and caching |
| [`status`](examples/status) | Monitor DNS server health and latency |

Run an example:

```bash
go run ./examples/basic
```

## Default DNS Servers

The checker comes pre-configured with known Nawala DNS servers:

| Server | Keyword | Query Type |
|---|---|---|
| `180.131.144.144` | `internetpositif` | `A` |
| `180.131.145.145` | `internetpositif` | `A` |

Nawala blocks domains by returning CNAME redirects to known block pages (`internetpositif.id` or `internetsehatku.com`). Komdigi blocks domains by returning an A record with EDE 15 (Blocked) containing `trustpositif.komdigi.go.id`. The keyword is matched against the full DNS record string for broad detection.

## How Blocking Works

Indonesian ISP DNS filters use two distinct blocking mechanisms:

### Nawala — CNAME Redirect

Nawala intercepts DNS queries for blocked domains and returns a **CNAME redirect** to a landing page instead of the real IP address:

```
;; ANSWER SECTION:
blocked.example.    3600    IN    CNAME    internetpositif.id.
```

The checker detects this by scanning all DNS record sections (Answer, Authority, Additional) for the keyword `internetpositif` in any record's string representation.

### Komdigi — EDE 15 (Blocked)

Komdigi uses the newer **Extended DNS Errors** mechanism ([RFC 8914](https://datatracker.ietf.org/doc/rfc8914/)). The response returns an A record pointing to a block page IP, along with an EDE option code 15 (Blocked) in the OPT pseudo-section:

```
;; OPT PSEUDOSECTION:
; EDE: 15 (Blocked): (source=block-list-zone;
;   blockListUrl=https://trustpositif.komdigi.go.id/assets/db/domains_isp;
;   domain=reddit.com)

;; ANSWER SECTION:
reddit.com.    30    IN    A    103.155.26.29
```

The checker detects this by scanning the Extra section (which contains the OPT record) for the keyword `trustpositif` or `komdigi`. To use this detection, configure a server with the appropriate keyword:

```go
nawala.WithServers([]nawala.DNSServer{
    {
        Address:   "103.155.26.28",
        Keyword:   "trustpositif",
        QueryType: "A",
    },
    {
        Address:   "103.155.26.29",
        Keyword:   "komdigi",
        QueryType: "A",
    },
})
```

## Project Structure

```
nawala-checker/
├── .github/            # CI workflows and Dependabot configuration
├── examples/           # Runnable usage examples (basic, custom, status)
├── Makefile            # Build and test shortcuts
└── src/
    └── nawala/          # Core SDK package (checker, cache, DNS, options, types)
```

## Testing

```bash
# Run tests with race detector.
make test

# Run tests with verbose output.
make test-verbose

# Run tests with coverage report.
make test-cover

# Skip live DNS tests.
make test-short
```

## Roadmap

- [ ] Upgrade `github.com/miekg/dns` to v2 or use a modern alternative for improved networking performance and features, due to its implementation in Go and its high effectiveness for networking.

## License

[BSD 3-Clause License](LICENSE) — Copyright (c) 2026, H0llyW00dzZ

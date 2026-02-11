# Nawala Checker

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25.6-blue?logo=go)](https://go.dev/dl/)
[![Go Reference](https://pkg.go.dev/badge/github.com/H0llyW00dzZ/nawala-checker.svg)](https://pkg.go.dev/github.com/H0llyW00dzZ/nawala-checker)
[![Go Report Card](https://goreportcard.com/badge/github.com/H0llyW00dzZ/nawala-checker)](https://goreportcard.com/report/github.com/H0llyW00dzZ/nawala-checker)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/H0llyW00dzZ/nawala-checker/graph/badge.svg?token=3GU9QRYAUX)](https://codecov.io/gh/H0llyW00dzZ/nawala-checker)
[![View on DeepWiki](https://img.shields.io/badge/View%20on-DeepWiki-blue)](https://deepwiki.com/H0llyW00dzZ/nawala-checker)

A Go SDK for checking whether domains are blocked by Indonesian ISP DNS filters (Nawala/Kominfo). It works by querying configurable DNS servers and scanning the responses for blocking keywords such as `internetpositif.id` redirects.

> [!IMPORTANT]
> This SDK requires an **Indonesian network** to function correctly. Nawala DNS servers only return blocking responses when queried from within Indonesia. If running on cloud infrastructure (e.g., VPS), you must use a pure Indonesian VPS with no routing through networks outside Indonesia.

## Features

- **Concurrent domain checking** — check multiple domains in parallel with a single call
- **DNS server failover** — automatic fallback to secondary servers when the primary fails
- **Retry with exponential backoff** — resilient against transient network errors
- **Built-in caching** — in-memory cache with configurable TTL to avoid redundant queries
- **Custom cache backends** — plug in Redis, memcached, or any backend via the `Cache` interface
- **Server health checks** — monitor online/offline status and latency of DNS servers
- **Panic recovery** — goroutines are protected from panics with automatic recovery and typed errors
- **Functional options** — clean, idiomatic Go configuration pattern
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

    results, err := c.Check(ctx, "google.com", "reddit.com", "github.com")
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

    // Add a custom DNS server alongside the defaults.
    nawala.WithServer(nawala.DNSServer{
        Address:   "8.8.8.8",
        Keyword:   "blocked",
        QueryType: "A",
    }),

    // Limit concurrent checks to 50 goroutines.
    nawala.WithConcurrency(50),
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

Nawala blocks domains by returning CNAME redirects to known block pages (`internetpositif.id` or `internetsehatku.com`). The keyword is matched against the full DNS record string for broad detection.

## Project Structure

```
nawala-checker/
├── .github/            # CI workflows and Dependabot configuration
├── examples/           # Runnable usage examples (basic, custom, status)
└── src/
    └── nawala/          # Core SDK package (checker, cache, DNS, options, types)
```

## Testing

```bash
# Run unit tests.
go test ./src/nawala/...

# Run all tests including live DNS checks.
go test -v ./src/nawala/...

# Run with race detector.
go test -race ./src/nawala/...

# Skip live DNS tests.
go test -short ./src/nawala/...

# Run with coverage.
go test -cover ./src/nawala/...
```

## License

[BSD 3-Clause License](LICENSE) — Copyright (c) 2026, H0llyW00dzZ

# Examples

[![Go Reference](https://pkg.go.dev/badge/github.com/H0llyW00dzZ/nawala-checker.svg)](https://pkg.go.dev/github.com/H0llyW00dzZ/nawala-checker)
[![Baca dalam Bahasa Indonesia](https://img.shields.io/badge/🇮🇩-Baca%20dalam%20Bahasa%20Indonesia-red)](README.id.md)

This directory contains self-contained, runnable examples for the
[nawala-checker](https://github.com/H0llyW00dzZ/nawala-checker) SDK — a
DNS-based domain blocking checker for Indonesian ISP DNS filters
(Nawala/Kominfo, now Komdigi).

> [!IMPORTANT]
> These examples require an **Indonesian network** to produce meaningful
> blocking results. Nawala and Komdigi DNS servers only return block
> indicators when queried from an Indonesian IP address. If you are running
> from outside Indonesia, configure a custom DNS server hosted on an
> Indonesian network and point the checker at it via `WithServers`.

| Example | Description |
|---|---|
| [`basic/`](basic/main.go) | Check multiple domains with default configuration |
| [`custom/`](custom/main.go) | Advanced configuration: custom servers, timeouts, retries, caching |
| [`status/`](status/main.go) | Monitor DNS server health and latency |

## Prerequisites

- **Go 1.25.6** or later
- An **Indonesian network connection** (or a custom DNS relay on an
  Indonesian network — see the tip below)

> [!TIP]
> When running on cloud infrastructure (e.g., VPS, microservices, k8s),
> deploy a DNS server inside your cluster on an Indonesian network node,
> then point the checker at it using `WithServers`. The Nawala/Komdigi
> servers will only respond with block indicators when they see an
> Indonesian source IP.

## Running an Example

```bash
go run ./examples/basic
go run ./examples/custom
go run ./examples/status
```

---

## `basic` — Default Configuration

[`basic/main.go`](basic/main.go) checks a small list of domains against
the pre-configured Nawala DNS servers with zero manual setup.

```go
c := nawala.New()

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := c.Check(ctx, "google.com", "reddit.com", "github.com", "exam_ple.com")
if err != nil {
    log.Fatalf("check failed: %v", err)
}

for _, r := range results {
    status := "not blocked"
    if r.Blocked {
        status = "BLOCKED"
    }
    if r.Error != nil {
        status = fmt.Sprintf("error: %v", r.Error)
    }
    fmt.Printf("  %-20s %s (server: %s)\n", r.Domain, status, r.Server)
}
```

**Expected output** (from an Indonesian network):

```
=== Nawala DNS Blocker Check ===

  exam_ple.com         error: nawala: nxdomain: domain does not exist (NXDOMAIN) (server: 180.131.144.144)
  google.com           not blocked (server: 180.131.144.144)
  reddit.com           BLOCKED (server: 180.131.144.144)
  github.com           not blocked (server: 180.131.144.144)
```

**What this demonstrates:**

- `nawala.New()` with no options uses the built-in Nawala DNS servers
  (`180.131.144.144`, `180.131.145.145`)
- `Check` checks all domains **concurrently** and returns one `Result` per domain
- `Result.Blocked` is `true` when the DNS response contains a blocking
  keyword (e.g., a CNAME redirect to `internetpositif.id`)
- `Result.Error` is non-nil when the check itself failed (network error,
  timeout, etc.) — separate from blocking status

---

## `custom` — Advanced Configuration

[`custom/main.go`](custom/main.go) shows how to use functional options to
tune the checker, add extra DNS servers, and demonstrate the built-in cache.

```go
c := nawala.New(
    // Add a custom DNS server alongside the defaults.
    nawala.WithServer(nawala.DNSServer{
        Address:   "8.8.8.8",
        Keyword:   "blocked",
        QueryType: "A",
    }),

    // Increase timeout for slow networks.
    nawala.WithTimeout(15 * time.Second),

    // Allow more retries (3 retries = 4 total attempts).
    nawala.WithMaxRetries(3),

    // Cache results for 10 minutes.
    nawala.WithCacheTTL(10 * time.Minute),
)
```

**Expected output** (from an Indonesian network):

```
=== Custom Configuration Check ===

Configured DNS servers:
  180.131.144.144 (keyword="internetpositif", type=A)
  180.131.145.145 (keyword="internetpositif", type=A)
  8.8.8.8 (keyword="blocked", type=A)

  google.com: not blocked (server: 180.131.144.144)

Second check (cached):
  google.com: blocked=false (took 4.2µs)
```

**What this demonstrates:**

- `WithServer` **appends** a single server to the existing list; use
  `WithServers` to **replace** all servers
- `WithTimeout` and `WithMaxRetries` control per-query resilience
- `WithCacheTTL` enables the in-memory TTL cache — the second `CheckOne`
  call returns in microseconds because the result is served from cache
- `c.Servers()` returns the full list of configured DNS servers at runtime

---

## `status` — DNS Server Health Check

[`status/main.go`](status/main.go) queries all configured DNS servers and
reports their online/offline status and round-trip latency.

```go
c := nawala.New()

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

statuses, err := c.DNSStatus(ctx)
if err != nil {
    log.Fatalf("status check failed: %v", err)
}

for _, s := range statuses {
    status := "OFFLINE"
    if s.Online {
        status = fmt.Sprintf("ONLINE (%dms)", s.LatencyMs)
    }
    fmt.Printf("  %-18s %s\n", s.Server, status)
    if s.Error != nil {
        fmt.Printf("    error: %v\n", s.Error)
    }
}
```

**Expected output** (from an Indonesian network):

```
=== Nawala DNS Server Status ===

  180.131.144.144    ONLINE (12ms)
  180.131.145.145    ONLINE (14ms)
```

**What this demonstrates:**

- `DNSStatus` probes all configured servers and returns a `ServerStatus`
  per server
- `ServerStatus.Online` is `true` when the server responds to a health
  probe within the configured timeout
- `ServerStatus.LatencyMs` is an `int64` of **whole milliseconds** (e.g., `12`
  meaning 12 ms); it is only populated when `Online` is `true` — offline
  servers leave it as `0`
- `ServerStatus.Error` is non-nil when the health probe itself failed
- Useful for monitoring or pre-flight checks before running bulk domain checks

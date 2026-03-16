---
name: check_domains
description: Check whether domains are blocked by Indonesian ISP DNS filters (Nawala/Komdigi)
---

# Check Domains Skill

Check one or more domains against Indonesian ISP DNS filters and return structured results indicating whether each domain is blocked, not blocked, or encountered an error.

## Prerequisites

Before using this skill, verify that the `nawala` CLI is installed:

```bash
nawala --version
```

If the command is not found, install it:

```bash
go install github.com/H0llyW00dzZ/nawala-checker/cmd/nawala@latest
```

## Integration Methods

### Method 1: CLI (Any Language)

Run the `nawala` CLI with `--format json` for machine-readable output.

#### Check domains

```bash
nawala check --format json google.com reddit.com github.com
```

#### Output format

The JSON response uses the nawala envelope format:

```json
{
  "nawala": {
    "version": "0.7.1",
    "result": [
      {"domain": "google.com", "blocked": false, "server": "180.131.144.144"},
      {"domain": "reddit.com", "blocked": true, "server": "180.131.144.144"},
      {"domain": "github.com", "blocked": false, "server": "180.131.144.144"}
    ]
  }
}
```

Each result object contains:

| Field | Type | Description |
|---|---|---|
| `domain` | string | The domain that was checked |
| `blocked` | bool | `true` if blocked, omitted on error |
| `server` | string | DNS server IP used for the check |
| `error` | string | Error message (only present on failure) |

#### Check from a file

```bash
nawala check --format json --file domains.txt
```

The file should contain one domain per line. Lines starting with `#` are comments.

#### With custom config

```bash
nawala check --format json --config config.json google.com
```

#### Exit codes

- `0` — all checks succeeded
- `1` — partial or complete failure (check individual result errors)

#### Parsing the output

1. Parse stdout as JSON
2. Access `nawala.result` array
3. For each result: check `error` first, then read `blocked`
4. A result with `"blocked": true` means the domain is filtered by the ISP

### Method 2: Go SDK (Go-based AI Tools)

> [!WARNING]
> This method is a **programmatic interface** and requires an **advanced AI agent setup** with a properly configured Go development environment. Your AI framework must be able to compile and run Go code natively. If unsure, use **Method 1 (CLI)** instead.

Import the SDK directly for native performance and streaming support.

#### Installation

```bash
go get github.com/H0llyW00dzZ/nawala-checker
```

#### Basic usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

func main() {
    checker := nawala.New()
    defer checker.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    results, err := checker.Check(ctx, "google.com", "reddit.com")
    if err != nil {
        // Handle error
        return
    }

    for _, r := range results {
        if r.Error != nil {
            fmt.Printf("%s: error: %v\n", r.Domain, r.Error)
            continue
        }
        fmt.Printf("%s: blocked=%v (server: %s)\n", r.Domain, r.Blocked, r.Server)
    }
}
```

#### Single domain check

```go
result, err := checker.CheckOne(ctx, "example.com")
```

#### Streaming (constant memory for large lists)

```go
in := make(chan string)
out := make(chan nawala.Result, checker.Concurrency())

go func() {
    defer close(in)
    for _, d := range domains {
        in <- d
    }
}()

go func() {
    _ = checker.CheckStream(ctx, nawala.Stream{In: in, Out: out})
    close(out)
}()

for r := range out {
    // Process each result as it arrives
}
```

#### Custom configuration

```go
checker := nawala.New(
    nawala.WithTimeout(10 * time.Second),
    nawala.WithMaxRetries(3),
    nawala.WithConcurrency(50),
    nawala.WithServers([]nawala.DNSServer{
        {Address: "8.8.8.8", Keyword: "blocked", QueryType: "A"},
    }),
)
```

#### Result type

```go
type Result struct {
    Domain  string // Domain that was checked
    Blocked bool   // Whether blocked (only valid when Error is nil)
    Server  string // DNS server IP used
    Error   error  // Non-nil if check failed
}
```

#### Error handling

Always check `Error` before reading `Blocked`:

```go
if errors.Is(r.Error, nawala.ErrInvalidDomain) {
    // Domain name is invalid
} else if errors.Is(r.Error, nawala.ErrNXDOMAIN) {
    // Domain does not exist
} else if errors.Is(r.Error, nawala.ErrDNSTimeout) {
    // DNS query timed out
} else if errors.Is(r.Error, nawala.ErrAllDNSFailed) {
    // All DNS servers failed
}
```

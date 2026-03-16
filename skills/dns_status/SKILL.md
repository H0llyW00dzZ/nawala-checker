---
name: dns_status
description: Check the health and latency of configured DNS servers
---

# DNS Status Skill

Check the online/offline status and round-trip latency of all configured DNS servers. Useful for diagnosing connectivity issues or verifying server availability before running domain checks.

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

#### Check server health

```bash
nawala status --format json
```

#### Output format

```json
{
  "nawala": {
    "version": "0.7.1",
    "status": [
      {"server": "180.131.144.144", "online": true, "latency_ms": 12},
      {"server": "180.131.145.145", "online": false, "error": "i/o timeout"}
    ]
  }
}
```

Each status object contains:

| Field | Type | Description |
|---|---|---|
| `server` | string | DNS server IP address |
| `online` | bool | `true` if server is responding |
| `latency_ms` | int | Round-trip time in ms (only when online) |
| `error` | string | Error message (only when offline) |

#### With custom config

```bash
nawala status --format json --config config.json
```

#### Interpreting results

1. Parse stdout as JSON
2. Access `nawala.status` array
3. Servers with `"online": true` are healthy — `latency_ms` shows response time
4. Servers with `"online": false` have connectivity issues — check `error` for details

### Method 2: Go SDK (Go-based AI Tools)

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

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    statuses, err := checker.DNSStatus(ctx)
    if err != nil {
        // Handle error
        return
    }

    for _, s := range statuses {
        if s.Error != nil {
            fmt.Printf("%s: OFFLINE (%v)\n", s.Server, s.Error)
            continue
        }
        fmt.Printf("%s: ONLINE (%dms)\n", s.Server, s.LatencyMs)
    }
}
```

#### Result type

```go
type ServerStatus struct {
    Server    string // DNS server address
    Online    bool   // Whether responding (only valid when Error is nil)
    LatencyMs int64  // Round-trip time in ms
    Error     error  // Non-nil if health check failed
}
```

#### Checking custom servers

```go
checker := nawala.New(
    nawala.WithServers([]nawala.DNSServer{
        {Address: "8.8.8.8", Keyword: "blocked", QueryType: "A"},
        {Address: "1.1.1.1", Keyword: "blocked", QueryType: "A"},
    }),
)

statuses, _ := checker.DNSStatus(ctx)
```

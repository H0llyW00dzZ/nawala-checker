---
name: config_inspect
description: Inspect effective configuration and generate config files for nawala-checker
---

# Config Inspect Skill

Inspect the effective configuration of nawala-checker (all defaults and overrides) and optionally generate a config file. Useful for understanding the current setup or creating custom configurations.

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

#### Show effective config as JSON

```bash
nawala config --json
```

#### Output format

```json
{
  "nawala": {
    "version": "0.7.1",
    "configuration": {
      "timeout": "5s",
      "command_timeout": "30s",
      "max_retries": 2,
      "cache_ttl": "5m",
      "disable_cache": false,
      "concurrency": 100,
      "edns0_size": 1232,
      "protocol": "udp",
      "tls_server_name": "",
      "tls_skip_verify": false,
      "keep_alive_pool_size": 0,
      "servers": [
        {"address": "180.131.144.144", "keyword": "internetpositif", "query_type": "A"},
        {"address": "103.155.26.28", "keyword": "trustpositif", "query_type": "A"}
      ]
    }
  }
}
```

#### Generate a config file

```bash
nawala config --json -o myconfig.json
```

This writes the effective config to `myconfig.json`, which can then be customized and loaded with `--config`.

#### Key configuration fields

| Field | Default | Description |
|---|---|---|
| `timeout` | `5s` | Per-query DNS timeout |
| `command_timeout` | `30s` | Overall CLI command timeout |
| `max_retries` | `2` | Retry attempts per query |
| `cache_ttl` | `5m` | In-memory cache TTL |
| `disable_cache` | `false` | Set `true` to disable caching |
| `concurrency` | `100` | Max concurrent DNS checks |
| `protocol` | `udp` | Transport: `udp`, `tcp`, or `tcp-tls` |
| `servers` | Nawala defaults | DNS servers to query |

### Method 2: Go SDK (Go-based AI Tools)

> [!WARNING]
> This method is a **programmatic interface** and requires an **advanced AI agent setup** with a properly configured Go development environment. Your AI framework must be able to compile and run Go code natively. If unsure, use **Method 1 (CLI)** instead.

The Go SDK uses functional options for configuration. There is no "inspect" method — configuration is set at construction time.

#### Default configuration

```go
checker := nawala.New() // All defaults applied
```

#### Custom configuration

```go
checker := nawala.New(
    nawala.WithTimeout(10 * time.Second),
    nawala.WithMaxRetries(3),
    nawala.WithCacheTTL(10 * time.Minute),
    nawala.WithConcurrency(50),
    nawala.WithProtocol("tcp-tls"),
    nawala.WithTLSServerName("dns.example.com"),
    nawala.WithEDNS0Size(4096),
    nawala.WithKeepAlive(5),
    nawala.WithServers([]nawala.DNSServer{
        {Address: "8.8.8.8", Keyword: "blocked", QueryType: "A"},
    }),
)
defer checker.Close()
```

#### Runtime server management

```go
// Check configured servers
servers := checker.Servers()

// Hot-reload: add or replace a server
checker.SetServers(nawala.DNSServer{
    Address: "203.0.113.1", Keyword: "blocked", QueryType: "A",
})

// Check if a server is configured
if checker.HasServer("203.0.113.1") {
    // Server is active
}

// Remove a server
checker.DeleteServers("203.0.113.1")
```

#### Available options reference

| Option | Default | Description |
|---|---|---|
| `WithTimeout(d)` | `5s` | Per-query DNS timeout |
| `WithMaxRetries(n)` | `2` | Retry attempts per query |
| `WithCacheTTL(d)` | `5m` | In-memory cache TTL |
| `WithCache(c)` | in-memory | Custom cache backend (`nil` to disable) |
| `WithConcurrency(n)` | `100` | Max concurrent DNS checks |
| `WithProtocol(s)` | `udp` | Transport: `udp`, `tcp`, `tcp-tls` |
| `WithServers(s)` | Nawala defaults | Replace all DNS servers |
| `WithKeepAlive(n)` | disabled | TCP/TLS connection pool size |
| `WithDigests(fn)` | disabled | Hash function for cache keys |

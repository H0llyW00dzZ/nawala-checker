// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package cli implements the nawala command-line interface.
//
// It provides a Cobra-based CLI with three subcommands:
//
//   - check  — check whether domains are blocked by Indonesian ISP DNS filters
//   - status — display the health and latency of configured DNS servers
//   - config — print the effective configuration (defaults or loaded file)
//
// The root command delegates to check when positional domain arguments are
// provided, so "nawala google.com" is equivalent to "nawala check google.com".
//
// # Configuration
//
// A JSON or YAML configuration file can be passed via the --config (-c) flag.
// The file format is auto-detected by extension (.json, .yaml, .yml). All
// fields are optional; unset fields fall through to the nawala SDK defaults.
// Use "nawala config" to generate a template with all defaults filled in.
//
// Example YAML (nawala envelope format):
//
//	nawala:
//	  configuration:
//	    timeout: 10s
//	    max_retries: 3
//	    cache_ttl: 10m
//	    disable_cache: false
//	    concurrency: 50
//	    protocol: udp
//	    tls_server_name: ""
//	    tls_skip_verify: false
//	    servers:
//	      - address: "180.131.144.144"
//	        keyword: "internetpositif"
//	        query_type: "A"
//
// Set disable_cache: true to disable the built-in in-memory cache entirely.
// When set, the cache_ttl field has no effect.
//
// Set protocol to "udp" (default), "tcp", or "tcp-tls" (DNS over TLS / DoT)
// to select the DNS transport without needing a custom WithDNSClient.
// For tcp-tls, two optional TLS fields are available:
//
//   - tls_server_name: the TLS identity (cert hostname) to verify against. The server
//     address (IP) and this name are independent — the IP is the transport destination,
//     the server name is what TLS verifies the cert against (no IP SAN needed in the cert).
//   - tls_skip_verify: disables TLS certificate verification entirely. Only for self-signed
//     certs; never use in production.
//
// # Domain Input
//
// Domains for the check command come from positional CLI arguments, a text
// file (--file, -f), or both. The text file format is one domain per line;
// lines starting with '#' are comments and blank lines are ignored.
// Duplicates across both sources are removed automatically.
//
// # Output
//
// Results are streamed to stdout by default, or written to a file with the
// --output (-o) flag. Output formats are controlled by the --format flag:
//
//   - text (default) — tab-separated columns: domain, status, server
//   - json           — one JSON object per line, suitable for piping (NDJSON)
//   - html           — a styled table report with color-coded status cells
//   - xlsx           — an XLSX spreadsheet with colored cells
//
// # Embedded Usage Text
//
// Long-form help descriptions for each command are stored as plain-text
// files under usage/ and compiled into the binary via [embed]. This keeps
// command descriptions editable without modifying Go source code.
// See [magic_embed.go] for the embed directives.
//
// [magic_embed.go]: https://github.com/H0llyW00dzZ/nawala-checker/blob/master/internal/cli/magic_embed.go
package cli

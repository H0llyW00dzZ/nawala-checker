// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package cli implements the nawala command-line interface.
//
// It provides a Cobra-based CLI with two subcommands:
//
//   - check — check whether domains are blocked by Indonesian ISP DNS filters
//   - status — display the health and latency of configured DNS servers
//
// The root command delegates to check when positional domain arguments are
// provided, so "nawala google.com" is equivalent to "nawala check google.com".
//
// # Configuration
//
// A JSON or YAML configuration file can be passed via the --config (-c) flag.
// The file format is auto-detected by extension (.json, .yaml, .yml). All
// fields are optional; unset fields fall through to the nawala SDK defaults.
//
// Example YAML:
//
//	timeout: 10s
//	max_retries: 3
//	cache_ttl: 10m
//	concurrency: 50
//	servers:
//	  - address: "180.131.144.144"
//	    keyword: "internetpositif"
//	    query_type: "A"
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
// --output (-o) flag. Two formats are available:
//
//   - Text (default) — tab-separated columns: domain, status, server
//   - NDJSON (--json) — one JSON object per line, suitable for piping
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

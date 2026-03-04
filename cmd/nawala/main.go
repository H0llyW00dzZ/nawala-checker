// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// nawala is a CLI tool for checking whether domains are blocked
// by Indonesian ISP DNS filters (Nawala/Kominfo/Komdigi).
//
// # Installation
//
//	go install github.com/H0llyW00dzZ/nawala-checker/cmd/nawala@latest
//
// # Usage
//
//	nawala [domains...]                       shorthand for "nawala check"
//	nawala check [domains...] [flags]         check domains for DNS blocking
//	nawala status [flags]                     show DNS server health and latency
//
// # Global Flags
//
//	-c, --config string   path to a JSON or YAML configuration file
//	-v, --version         print version and exit
//	-h, --help            help for nawala
//
// # Check Flags
//
//	-f, --file string     path to a .txt file with one domain per line
//	-o, --output string   write results to a file instead of stdout
//	    --json            output results as NDJSON (one JSON object per line)
//
// # Configuration File
//
// All detailed settings are configured via a JSON or YAML file passed
// with --config. The format is auto-detected by file extension. All
// fields are optional; unset fields use the nawala SDK defaults.
//
// Example (YAML):
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
// # Exit Codes
//
//	0   all checks completed successfully
//	1   one or more checks encountered errors
//	2   fatal error (invalid config, missing domains, etc.)
package main

import (
	"fmt"
	"os"

	"github.com/H0llyW00dzZ/nawala-checker/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

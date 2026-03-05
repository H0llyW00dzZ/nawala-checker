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
//	nawala config [flags]                     print the effective configuration
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
//	    --format string   output format (text, json, html, xlsx) (default "text")
//
// # Config Flags
//
//	-o, --output string   write config to a file instead of stdout
//	    --json            output as JSON
//	    --yaml            output as YAML (default)
//
// # Configuration File
//
// All settings are configured via a JSON or YAML file passed with --config.
// The format is auto-detected by file extension. All fields are optional;
// unset fields use the nawala SDK defaults.
//
// The file uses the nawala envelope format (JSON):
//
//	{
//	  "nawala": {
//	    "configuration": {
//	      "timeout": "10s",
//	      "max_retries": 3,
//	      "cache_ttl": "10m",
//	      "disable_cache": false,
//	      "concurrency": 50,
//	      "protocol": "udp",
//	      "tls_server_name": "",
//	      "tls_skip_verify": false,
//	      "servers": [
//	        {"address": "180.131.144.144", "keyword": "internetpositif", "query_type": "A"}
//	      ]
//	    }
//	  }
//	}
//
// Or in YAML:
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
// Use "nawala config" to generate a template with all SDK defaults filled in.
//
// The protocol field selects the DNS transport: "udp" (default), "tcp",
// or "tcp-tls" (DNS over TLS / DoT).
// For tcp-tls, set tls_server_name to override the SNI (useful when the server
// address is an IP), and tls_skip_verify: true only for self-signed certs.
//
// # Exit Codes
//
//	0   all checks completed successfully
//	1   one or more checks encountered errors
//	2   fatal error (invalid config, missing domains, etc.)
package main

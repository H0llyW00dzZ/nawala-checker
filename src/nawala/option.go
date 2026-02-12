// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"time"

	"github.com/miekg/dns"
)

// Option is a functional option for configuring a [Checker].
type Option func(*Checker)

// WithServer adds or replaces a DNS server in the checker's configuration.
// If a server with the same address already exists, it will be replaced.
func WithServer(server DNSServer) Option {
	return func(c *Checker) {
		for i, s := range c.servers {
			if s.Address == server.Address {
				c.servers[i] = server
				return
			}
		}
		c.servers = append(c.servers, server)
	}
}

// WithServers replaces all configured DNS servers.
// This overrides the default Nawala DNS servers.
func WithServers(servers []DNSServer) Option {
	return func(c *Checker) {
		c.servers = servers
	}
}

// WithTimeout sets the timeout for each DNS query.
// The default is 5 seconds.
//
// This option has no effect if a custom DNS client is set via [WithDNSClient],
// as the custom client's own Timeout configuration takes precedence.
func WithTimeout(d time.Duration) Option {
	return func(c *Checker) {
		c.timeout = d
	}
}

// WithMaxRetries sets the maximum number of retry attempts per DNS query.
// The default is 2 retries (3 total attempts).
func WithMaxRetries(n int) Option {
	return func(c *Checker) {
		c.maxRetries = n
	}
}

// WithCache sets a custom [Cache] implementation.
// By default, the checker uses an in-memory cache with a 5-minute TTL.
//
// Pass nil to disable caching entirely.
func WithCache(cache Cache) Option {
	return func(c *Checker) {
		c.cache = cache
	}
}

// WithCacheTTL sets the TTL for the built-in in-memory cache.
// This has no effect if a custom cache is set via [WithCache].
// The default is 5 minutes.
func WithCacheTTL(d time.Duration) Option {
	return func(c *Checker) {
		c.cacheTTL = d
	}
}

// WithConcurrency sets the maximum number of concurrent DNS checks.
// The default is 100.
func WithConcurrency(n int) Option {
	return func(c *Checker) {
		if n > 0 {
			c.concurrency = n
		}
	}
}

// WithDNSClient sets a custom [dns.Client] for all DNS operations.
// This allows full control over the transport configuration, including:
//
//   - TCP transport (Net: "tcp")
//   - DNS-over-TLS (Net: "tcp-tls" with TLSConfig)
//   - Custom Dialer for proxy or interface binding
//   - SingleInflight for connection deduplication
//
// When set, the [WithTimeout] option will not affect DNS queries;
// the client's own Timeout and TLSConfig will be used instead.
//
// Note: This is mostly for advanced usage. If you know how to configure
// a [dns.Client] perfectly, you possess a deep understanding of how the internet works.
//
// Passing nil is a no-op and the default UDP client will be used.
func WithDNSClient(client *dns.Client) Option {
	return func(c *Checker) {
		if client != nil {
			c.dnsClient = client
		}
	}
}

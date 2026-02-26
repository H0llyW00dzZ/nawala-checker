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

// WithServers replaces all configured DNS servers.
// This overrides the default Nawala DNS servers.
func WithServers(servers []DNSServer) Option {
	return func(c *Checker) {
		c.servers = servers
	}
}

// SetServers adds or replaces DNS servers on a running [Checker].
// It is safe to call concurrently with [Checker.Check], [Checker.CheckOne],
// and [Checker.DNSStatus].
//
// For each server provided, if a server with the same address is already
// configured it is replaced in-place; otherwise it is appended.
// The change takes effect for all DNS queries that start after this call
// returns — in-flight queries use their own snapshot of the server list.
//
// Passing zero servers is a no-op.
//
// Example — hot-reload a single server at runtime:
//
//	c.SetServers(nawala.DNSServer{
//	    Address:   "203.0.113.1",
//	    Keyword:   "blocked",
//	    QueryType: "A",
//	})
func (c *Checker) SetServers(servers ...DNSServer) {
	if len(servers) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, server := range servers {
		updated := false
		for i, s := range c.servers {
			if s.Address == server.Address {
				c.servers[i] = server
				updated = true
				break
			}
		}
		if !updated {
			c.servers = append(c.servers, server)
		}
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
		if n < 0 {
			n = defaultRetries // Use default on negative input
		}
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
// a [dns.Client] perfectly, you possess a deep understanding of [how the internet works].
//
// Passing nil is a no-op and the default UDP client will be used.
//
// [how the internet works]: https://youtu.be/x3c1ih2NJEg?si=PEX0eGMsxOiMcEHk
func WithDNSClient(client *dns.Client) Option {
	return func(c *Checker) {
		if client != nil {
			c.dnsClient = client
		}
	}
}

// WithEDNS0Size sets the EDNS0 UDP buffer size.
// The default is 1232 bytes, which is the recommended size to prevent
// IP fragmentation over UDP.
//
// See: https://dnsflagday.net/2020/
func WithEDNS0Size(size uint16) Option {
	return func(c *Checker) {
		if size > 0 {
			c.edns0Size = size
		}
	}
}

// DeleteServers removes one or more servers from the checker's active
// configuration at runtime. It is concurrency-safe and will safely remove
// servers identified by their Address field.
//
// Passing zero servers or non-existent addresses is a no-op.
func (c *Checker) DeleteServers(addresses ...string) {
	if len(addresses) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	toDelete := make(map[string]struct{}, len(addresses))
	for _, addr := range addresses {
		toDelete[addr] = struct{}{}
	}

	var newServers []DNSServer
	for _, s := range c.servers {
		if _, deleteMe := toDelete[s.Address]; !deleteMe {
			newServers = append(newServers, s)
		}
	}
	c.servers = newServers
}

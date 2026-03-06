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
// If multiple servers with identical configurations (Address, Keyword, and QueryType) are provided, only the first occurrence is kept.
func WithServers(servers []DNSServer) Option {
	return func(c *Checker) {
		if len(servers) == 0 {
			c.servers = servers
			return
		}

		type serverKey struct {
			Address   string
			Keyword   string
			QueryType string
		}
		seen := make(map[serverKey]struct{}, len(servers))
		deduped := make([]DNSServer, 0, len(servers))

		for _, s := range servers {
			key := serverKey(s)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				deduped = append(deduped, s)
			}
		}
		c.servers = deduped
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

// HasServer returns true if a DNS server with the given address is
// currently configured. It is safe to call concurrently with other
// runtime configuration methods.
//
// The address should match exactly the address provided in [DNSServer.Address].
func (c *Checker) HasServer(address string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, s := range c.servers {
		if s.Address == address {
			return true
		}
	}
	return false
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
		c.cacheSet = true
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

// WithProtocol sets the DNS transport protocol used by the default DNS client.
// The default is "udp".
//
// Valid values:
//
//   - "udp"     — standard DNS over UDP (default; [RFC 1035])
//   - "tcp"     — DNS over TCP ([RFC 1035])
//   - "tcp-tls" — DNS over TLS, also known as DoT ([RFC 7858])
//
// Invalid values are ignored and the default "udp" is kept.
// This option has no effect if a custom DNS client is set via [WithDNSClient],
// as the custom client's own Net configuration takes precedence.
//
// [RFC 1035]: https://www.rfc-editor.org/rfc/rfc1035.html
// [RFC 7858]: https://www.rfc-editor.org/rfc/rfc7858.html
func WithProtocol(net string) Option {
	return func(c *Checker) {
		switch net {
		case "udp", "tcp", "tcp-tls":
			c.dnsProtocol = net
		}
	}
}

// WithTLSServerName overrides the TLS server name (SNI) used when connecting
// to a DoT (DNS-over-TLS) server.
//
// The server address (IP) and server name are independent. You connect to the IP
// as the transport destination, while the server name declares the TLS identity
// to verify the certificate against. The cert does not need to contain the IP.
// This is identical to how HTTPS works: connect to a resolved IP, verify the
// cert against the hostname.
//
// Example — IP address with a trusted CA cert:
//
//	c := nawala.New(
//	    nawala.WithProtocol("tcp-tls"),
//	    nawala.WithTLSServerName("dns.example.com"), // cert subject
//	    // tls_skip_verify: false (default) — full cert verification
//	)
//
// Only applies when [WithProtocol]("tcp-tls") is in use. Has no effect for
// "udp" or "tcp" protocols, or when a custom client is set via [WithDNSClient].
func WithTLSServerName(name string) Option {
	return func(c *Checker) {
		c.tlsServerName = name
	}
}

// WithTLSSkipVerify disables TLS certificate verification when connecting to a
// DoT (DNS-over-TLS) server.
//
// WARNING: this disables all TLS security guarantees. Only use this in
// controlled environments (e.g. testing with a self-signed certificate).
// Never use this in production against untrusted networks.
//
// Only applies when [WithProtocol]("tcp-tls") is in use. Has no effect for
// "udp" or "tcp" protocols, or when a custom client is set via [WithDNSClient].
func WithTLSSkipVerify() Option {
	return func(c *Checker) {
		c.tlsSkipVerify = true
	}
}

// WithDigests enables digest-based cache keys using the provided hash function.
// When set, the raw cache key components (domain, server address, keyword, and
// query type) are concatenated and passed to hash, and the returned string
// (the digest) is used as the key body.  The final cache key is always
// prefixed with the SDK namespace so it has the form:
//
//	nawala_checker:<digest>
//
// The hash function receives the raw key string and must return a
// deterministic, collision-resistant representation of it, for example:
//
//	import "crypto/sha256"
//	import "encoding/hex"
//
//	nawala.WithDigests(func(data string) string {
//	    sum := sha256.Sum256([]byte(data))
//	    return hex.EncodeToString(sum[:])
//	})
//
// Passing nil is a no-op; the default prefixed-but-unhashed key is kept.
//
// This option is useful when:
//   - The cache backend imposes a maximum key length.
//   - Sensitive information (e.g. internal server addresses) must not appear
//     in cache keys in plain text.
//   - A consistent fixed-width key format is desired (e.g. hex SHA-256).
func WithDigests(hash func(data string) string) Option {
	return func(c *Checker) {
		if hash != nil {
			c.digestHash = hash
		}
	}
}

// WithKeepAlive enables a persistent TCP connection pool for DNS queries,
// reusing established connections across queries to avoid the per-query
// overhead of TCP (or TLS) handshakes.
//
// poolSize is the maximum number of idle connections to maintain per
// configured DNS server. Values ≤ 0 default to min(concurrency, 10).
//
// # Protocol support
//
// Keep-alive is only activated when the transport is "tcp" or "tcp-tls"
// (see [WithProtocol]). For the default "udp" transport this option is
// accepted but has no effect, since UDP is stateless and connectionless.
//
// # Server compatibility
//
// Not all DNS servers support persistent (keep-alive) TCP connections.
// The original RFC 1035 specified that DNS/TCP connections should be
// closed after each query-response pair. RFC 7766 (2016) later introduced
// persistent connections as a SHOULD-level requirement for modern
// implementations, and RFC 7858 (DNS-over-TLS) requires connection reuse.
//
// In practice:
//
//   - DNS-over-TLS (tcp-tls) — strongly recommended; all major DoT
//     providers (Cloudflare 1.1.1.1:853, Google 8.8.8.8:853, etc.) and
//     modern resolvers fully support RFC 7858 reuse.
//   - Custom / local resolvers (Unbound, BIND 9.x, Knot DNS, PowerDNS) —
//     supported; these implement RFC 7766 persistent connections.
//   - Legacy or ISP-managed DNS servers — may not support persistent
//     connections and close TCP after each response. The pool handles this
//     transparently (EOF → automatic redial), so queries never fail, but
//     there is no connection-reuse benefit.
//
// The default Nawala/Komdigi ISP servers are optimised for high-volume UDP
// traffic and are not expected to keep TCP connections open. WithKeepAlive
// with those servers provides no performance benefit; it is designed for
// custom deployments using a modern resolver that supports RFC 7766 or
// RFC 7858.
//
// When keep-alive is enabled, call [Checker.Close] when the checker is no
// longer needed to release idle connections:
//
//	c := nawala.New(
//	    nawala.WithProtocol("tcp-tls"),
//	    nawala.WithKeepAlive(5),
//	)
//	defer c.Close()
func WithKeepAlive(poolSize int) Option {
	return func(c *Checker) {
		c.keepAlive = true
		c.poolSize = poolSize
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

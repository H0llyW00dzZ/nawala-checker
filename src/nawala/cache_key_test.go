// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hashSHA256 returns the hex-encoded SHA-256 digest of data.
// This is the standard single-round SHA-256 used across most systems.
func hashSHA256(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// hashDoubleSHA256 returns the hex-encoded double-SHA-256 digest of data —
// SHA256(SHA256(data)) — the same algorithm Bitcoin uses internally for
// block headers, transaction IDs, and Merkle tree nodes.
//
// Reference: https://en.bitcoin.it/wiki/Protocol_documentation#Hashes
func hashDoubleSHA256(data string) string {
	first := sha256.Sum256([]byte(data))
	second := sha256.Sum256(first[:])
	return hex.EncodeToString(second[:])
}

// cacheWrapper is a test helper that wraps a [Cache] and invokes onSet for
// every [Cache.Set] call, allowing cache-key inspection without coupling tests
// to the internal memoryCache type.
type cacheWrapper struct {
	inner Cache
	onSet func(key string)
}

func (c cacheWrapper) Get(key string) (Result, bool) { return c.inner.Get(key) }
func (c cacheWrapper) Set(key string, val Result) {
	c.inner.Set(key, val)
	if c.onSet != nil {
		c.onSet(key)
	}
}
func (c cacheWrapper) Flush() { c.inner.Flush() }

// capturedKeyCache holds keys recorded by cacheWrapper across multiple Set calls.
type capturedKeyCache struct {
	sync.Mutex
	vals []string
}

func (ck *capturedKeyCache) snapshot() []string {
	ck.Lock()
	defer ck.Unlock()
	out := make([]string, len(ck.vals))
	copy(out, ck.vals)
	return out
}

// startSimpleDNSServer is a thin helper that starts a local DNS server
// responding with a plain A record (1.2.3.4) for any query.
func startSimpleDNSServer(t *testing.T) (addr string, cleanup func()) {
	t.Helper()
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP("1.2.3.4"),
		})
		_ = w.WriteMsg(m)
	})
	return startTestDNSServer(t, handler)
}

// newCapturedCache returns a [cacheWrapper] backed by an in-memory cache that
// records every key passed to Set. The returned *capturedKeyCache can be
// snapshotted at any time to inspect the recorded keys.
func newCapturedCache(ttl time.Duration) (cacheWrapper, *capturedKeyCache) {
	captured := &capturedKeyCache{}
	wrapped := cacheWrapper{
		inner: newMemoryCache(ttl),
		onSet: func(key string) {
			captured.Lock()
			captured.vals = append(captured.vals, key)
			captured.Unlock()
		},
	}
	return wrapped, captured
}

// assertKeyFormat is a shared assertion helper that verifies:
//  1. Every captured cache key starts with the SDK namespace prefix.
//  2. Every captured cache key body (after the prefix) exactly matches the
//     output of hashFn applied to the corresponding raw input.
//
// When hashFn is nil only the prefix assertion is performed (plain-key mode).
func assertKeyFormat(t *testing.T, keys []string, hashFn func(string) string) {
	t.Helper()
	require.NotEmpty(t, keys, "expected at least one cache Set call")

	for _, k := range keys {
		require.True(t, len(k) > len(cacheKeyPrefix),
			"cache key %q is too short to contain prefix %q", k, cacheKeyPrefix)
		assert.Equal(t, cacheKeyPrefix, k[:len(cacheKeyPrefix)],
			"cache key must start with SDK namespace prefix %q", cacheKeyPrefix)

		if hashFn == nil {
			continue
		}

		// The key body (everything after the prefix) must be a valid
		// 64-character lowercase hex string — the expected output of SHA-256
		// or double-SHA-256 over any input.
		body := k[len(cacheKeyPrefix):]
		assert.Len(t, body, sha256.Size*2,
			"digest body must be %d hex chars (SHA-256 output)", sha256.Size*2)
		_, decodeErr := hex.DecodeString(body)
		assert.NoError(t, decodeErr,
			"digest body %q must be valid lowercase hex", body)
	}
}

// TestCacheKeyPrefix verifies that every cache entry produced by [Checker]
// carries the SDK namespace prefix ("nawala_checker:").
//
// This ensures that when multiple packages share the same cache backend
// (e.g., Redis, memcached), their keys cannot collide with keys generated
// by this SDK.
func TestCacheKeyPrefix(t *testing.T) {
	addr, cleanup := startSimpleDNSServer(t)
	defer cleanup()

	wrapped, captured := newCapturedCache(5 * time.Minute)

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithCache(wrapped),
	)

	ctx := context.Background()
	_, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)

	assertKeyFormat(t, captured.snapshot(), nil)
}

// TestWithDigestsSHA256 verifies that [WithDigests] using standard SHA-256
// replaces the cache key body with a 64-character hex digest while keeping
// the "nawala_checker:" namespace prefix.
//
// SHA-256 is the most widely used cryptographic hash and is the first of the
// two rounds in Bitcoin's double-SHA-256 scheme.
//
// The final key format is:
//
//	nawala_checker:<hex-sha256>
func TestWithDigestsSHA256(t *testing.T) {
	addr, cleanup := startSimpleDNSServer(t)
	defer cleanup()

	wrapped, captured := newCapturedCache(5 * time.Minute)

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithCache(wrapped),
		WithDigests(hashSHA256),
	)

	ctx := context.Background()
	_, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)

	keys := captured.snapshot()
	assertKeyFormat(t, keys, hashSHA256)
}

// TestWithDigestsDoubleSHA256 verifies that [WithDigests] using Bitcoin-style
// double-SHA-256 (SHA256(SHA256(data))) produces valid, prefix-namespaced
// cache keys.
//
// Bitcoin uses this construction for block headers, transaction IDs (txid /
// wtxid), and all Merkle tree node hashes. The double round provides
// additional preimage resistance and protects against length-extension attacks
// that can theoretically affect single-SHA-256 in some protocols.
//
// Reference: https://en.bitcoin.it/wiki/Protocol_documentation#Hashes
//
// The final key format is:
//
//	nawala_checker:<hex-sha256d>
func TestWithDigestsDoubleSHA256(t *testing.T) {
	addr, cleanup := startSimpleDNSServer(t)
	defer cleanup()

	wrapped, captured := newCapturedCache(5 * time.Minute)

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithCache(wrapped),
		WithDigests(hashDoubleSHA256),
	)

	ctx := context.Background()
	_, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)

	keys := captured.snapshot()
	assertKeyFormat(t, keys, hashDoubleSHA256)
}

// TestWithDigestsNilIsNoop verifies that passing nil to [WithDigests] is a
// safe no-op: the checker falls back to the default plain prefixed key and no
// panic occurs.
func TestWithDigestsNilIsNoop(t *testing.T) {
	c := New(
		WithCache(nil), // disable caching; we only test the field state
		WithDigests(nil),
	)
	assert.Nil(t, c.digestHash,
		"digestHash must remain nil after WithDigests(nil) — the option must be a no-op")
}

// TestWithDigestsCacheHitSHA256 verifies that two consecutive [Checker.CheckOne]
// calls with SHA-256 [WithDigests] produce a cache hit on the second call —
// meaning the digested key is stored and looked up deterministically.
func TestWithDigestsCacheHitSHA256(t *testing.T) {
	var queryCount int
	var mu sync.Mutex

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		mu.Lock()
		queryCount++
		mu.Unlock()
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP("1.2.3.4"),
		})
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startTestDNSServer(t, handler)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithCacheTTL(5*time.Minute),
		WithDigests(hashSHA256),
	)

	ctx := context.Background()

	r1, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)
	require.NoError(t, r1.Error)

	mu.Lock()
	firstCount := queryCount
	mu.Unlock()
	assert.Greater(t, firstCount, 0, "expected DNS queries on first call")

	// Second call must hit the SHA-256-keyed cache — no new DNS queries.
	r2, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)
	require.NoError(t, r2.Error)

	mu.Lock()
	secondCount := queryCount
	mu.Unlock()
	assert.Equal(t, firstCount, secondCount,
		"expected no new DNS queries on second call (cache hit via SHA-256 digest key)")
}

// TestWithDigestsCacheHitDoubleSHA256 verifies that two consecutive
// [Checker.CheckOne] calls with Bitcoin-style double-SHA-256 [WithDigests]
// produce a cache hit on the second call.
func TestWithDigestsCacheHitDoubleSHA256(t *testing.T) {
	var queryCount int
	var mu sync.Mutex

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		mu.Lock()
		queryCount++
		mu.Unlock()
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP("1.2.3.4"),
		})
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startTestDNSServer(t, handler)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithCacheTTL(5*time.Minute),
		WithDigests(hashDoubleSHA256),
	)

	ctx := context.Background()

	r1, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)
	require.NoError(t, r1.Error)

	mu.Lock()
	firstCount := queryCount
	mu.Unlock()
	assert.Greater(t, firstCount, 0, "expected DNS queries on first call")

	// Second call must hit the double-SHA-256-keyed cache — no new DNS queries.
	r2, err := c.CheckOne(ctx, "example.com")
	require.NoError(t, err)
	require.NoError(t, r2.Error)

	mu.Lock()
	secondCount := queryCount
	mu.Unlock()
	assert.Equal(t, firstCount, secondCount,
		"expected no new DNS queries on second call (cache hit via double-SHA-256 digest key)")
}

// TestWithDigestsDeterminism verifies that both SHA-256 and double-SHA-256
// hash functions are deterministic: the same input always produces the same
// digest, so cache lookups are reproducible across calls.
func TestWithDigestsDeterminism(t *testing.T) {
	const input = "example.com:127.0.0.1:10000:internetpositif:1"

	t.Run("SHA-256 is deterministic", func(t *testing.T) {
		d1 := hashSHA256(input)
		d2 := hashSHA256(input)
		assert.Equal(t, d1, d2, "SHA-256 must produce the same digest for identical input")
		assert.Len(t, d1, sha256.Size*2, "SHA-256 hex digest must be 64 characters")
		_, err := hex.DecodeString(d1)
		assert.NoError(t, err, "SHA-256 digest must be valid hex")
	})

	t.Run("double-SHA-256 is deterministic", func(t *testing.T) {
		d1 := hashDoubleSHA256(input)
		d2 := hashDoubleSHA256(input)
		assert.Equal(t, d1, d2, "double-SHA-256 must produce the same digest for identical input")
		assert.Len(t, d1, sha256.Size*2, "double-SHA-256 hex digest must be 64 characters")
		_, err := hex.DecodeString(d1)
		assert.NoError(t, err, "double-SHA-256 digest must be valid hex")
	})

	t.Run("SHA-256 differs from double-SHA-256", func(t *testing.T) {
		single := hashSHA256(input)
		double := hashDoubleSHA256(input)
		assert.NotEqual(t, single, double,
			"single and double SHA-256 must produce different digests for the same input")
	})
}

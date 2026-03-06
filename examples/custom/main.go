// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main demonstrates advanced configuration of the nawala
// checker SDK using functional options.
//
// It shows how to add custom DNS servers, configure timeouts,
// retries, cache TTL, and digest-based cache keys (SHA-256 and
// Bitcoin-style double-SHA-256).
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

// digestSHA256 returns the hex-encoded SHA-256 digest of data.
// This is the standard single-round hash used in most systems.
func digestSHA256(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// digestDoubleSHA256 returns the hex-encoded double-SHA-256 digest of data —
// SHA256(SHA256(data)) — the same algorithm Bitcoin uses internally for block
// headers, transaction IDs (txid / wtxid), and all Merkle tree nodes.
//
// Reference: https://en.bitcoin.it/wiki/Protocol_documentation#Hashes
func digestDoubleSHA256(data string) string {
	first := sha256.Sum256([]byte(data))
	second := sha256.Sum256(first[:])
	return hex.EncodeToString(second[:])
}

func main() {
	// ── 1. Standard configuration ─────────────────────────────────────────────

	// Create a checker with custom configuration.
	c := nawala.New(
		// Increase timeout for slow networks.
		nawala.WithTimeout(15*time.Second),

		// Allow more retries.
		nawala.WithMaxRetries(3),

		// Cache results for 10 minutes.
		nawala.WithCacheTTL(10*time.Minute),
	)

	ctx := context.Background()

	// Add a custom DNS server alongside the defaults.
	c.SetServers(nawala.DNSServer{
		Address:   "8.8.8.8",
		Keyword:   "blocked",
		QueryType: "A",
	})

	fmt.Println("=== Custom Configuration Check ===")
	fmt.Println()

	// Print configured servers.
	fmt.Println("Configured DNS servers:")
	for _, srv := range c.Servers() {
		fmt.Printf("  %s (keyword=%q, type=%s)\n", srv.Address, srv.Keyword, srv.QueryType)
	}
	fmt.Println()

	// Check a single domain.
	result, err := c.CheckOne(ctx, "google.com")
	if err != nil {
		log.Fatalf("check failed: %v", err)
	}

	status := "not blocked"
	if result.Blocked {
		status = "BLOCKED"
	}
	fmt.Printf("  %s: %s (server: %s)\n", result.Domain, status, result.Server)

	// ── 2. Runtime hot-reload ─────────────────────────────────────────────────

	fmt.Println("\n=== Runtime Reconfiguration ===")

	newIP := "203.0.113.1"
	fmt.Printf("\nAdding new server %s at runtime...\n", newIP)

	// Add server safely at runtime.
	c.SetServers(nawala.DNSServer{
		Address:   newIP,
		Keyword:   "blocked",
		QueryType: "A",
	})

	// Verify it was added.
	if c.HasServer(newIP) {
		fmt.Printf("Successfully verified %s is active!\n", newIP)
	}

	// Hot-reload: remove it.
	fmt.Printf("Removing server %s...\n", newIP)
	c.DeleteServers(newIP)
	if !c.HasServer(newIP) {
		fmt.Printf("Successfully verified %s was removed!\n", newIP)
	}

	// Demonstrate cache — second check is instant.
	fmt.Println()
	fmt.Println("Second check (cached):")
	start := time.Now()
	result2, _ := c.CheckOne(ctx, "google.com")
	elapsed := time.Since(start)
	fmt.Printf("  %s: blocked=%v (took %v)\n", result2.Domain, result2.Blocked, elapsed)

	// ── 3. Digest-based cache keys ────────────────────────────────────────────
	//
	// WithDigests replaces the plain cache key body with a hash digest.
	// All keys are always prefixed with "nawala_checker:", so the final
	// format is: nawala_checker:<digest>
	//
	// This is useful when:
	//   - The cache backend enforces a maximum key length.
	//   - Server addresses must not appear in keys in plain text.
	//   - A consistent fixed-width key format is required.

	fmt.Println("\n=== Digest-Based Cache Keys ===")

	// SHA-256 (single-round) — 64-char hex, widely used standard.
	cSHA256 := nawala.New(
		nawala.WithCacheTTL(5*time.Minute),
		nawala.WithDigests(digestSHA256),
	)

	r1, err := cSHA256.CheckOne(ctx, "google.com")
	if err != nil {
		log.Fatalf("SHA-256 check failed: %v", err)
	}
	fmt.Printf("  [SHA-256 key]        google.com: blocked=%v (server: %s)\n", r1.Blocked, r1.Server)

	// Second call — served from the SHA-256-keyed cache.
	start = time.Now()
	r1cached, _ := cSHA256.CheckOne(ctx, "google.com")
	fmt.Printf("  [SHA-256 cache hit]  google.com: blocked=%v (took %v)\n", r1cached.Blocked, time.Since(start))

	fmt.Println()

	// Double-SHA-256 (Bitcoin-style: SHA256(SHA256(data))) — same 64-char
	// hex output, extra preimage resistance and length-extension protection.
	cDSHA256 := nawala.New(
		nawala.WithCacheTTL(5*time.Minute),
		nawala.WithDigests(digestDoubleSHA256),
	)

	r2, err := cDSHA256.CheckOne(ctx, "google.com")
	if err != nil {
		log.Fatalf("double-SHA-256 check failed: %v", err)
	}
	fmt.Printf("  [double-SHA-256 key]        google.com: blocked=%v (server: %s)\n", r2.Blocked, r2.Server)

	// Second call — served from the double-SHA-256-keyed cache.
	start = time.Now()
	r2cached, _ := cDSHA256.CheckOne(ctx, "google.com")
	fmt.Printf("  [double-SHA-256 cache hit]  google.com: blocked=%v (took %v)\n", r2cached.Blocked, time.Since(start))
}

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main demonstrates advanced configuration of the nawala
// checker SDK using functional options.
//
// It shows how to add custom DNS servers, configure timeouts,
// retries, and cache TTL.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

func main() {
	// Create a checker with custom configuration.
	c := nawala.New(
		// Add a custom DNS server alongside the defaults.
		nawala.WithServer(nawala.DNSServer{
			Address:   "8.8.8.8",
			Keyword:   "blocked",
			QueryType: "A",
		}),

		// Increase timeout for slow networks.
		nawala.WithTimeout(15*time.Second),

		// Allow more retries.
		nawala.WithMaxRetries(3),

		// Cache results for 10 minutes.
		nawala.WithCacheTTL(10*time.Minute),
	)

	ctx := context.Background()

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

	// Demonstrate cache — second check is instant.
	fmt.Println()
	fmt.Println("Second check (cached):")
	start := time.Now()
	result2, _ := c.CheckOne(ctx, "google.com")
	elapsed := time.Since(start)
	fmt.Printf("  %s: blocked=%v (took %v)\n", result2.Domain, result2.Blocked, elapsed)
}

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main demonstrates how to use SetServers to hot-reload
// DNS servers safely while concurrent DNS checks are running.
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

func main() {
	fmt.Println("=== Nawala DNS Hot-Reload Example ===")

	// 1. Initialize checker with a safe initial server (e.g., Google DNS).
	// We'll intentionally configure it to look for a word that doesn't exist
	// so the initial checking loops return "not blocked".
	c := nawala.New(
		nawala.WithServers([]nawala.DNSServer{
			{
				Address:   "8.8.8.8",
				Keyword:   "this-will-never-match",
				QueryType: "A",
			},
		}),
		// Disable cache so we can immediately see the effect of server changes.
		nawala.WithCache(nil),
	)

	// Context for the entire example: max 10 seconds.
	exampleCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// 2. Start a continuous checking loop in the background.
	// This simulates live traffic flowing through the checker continuously.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-exampleCtx.Done():
				return // Context cancelled, exit loop.
			default:
			}

			// We use a short timeout for each individual check.
			ctx, checkCancel := context.WithTimeout(exampleCtx, 1*time.Second)
			// Check reddit.com (which IS blocked by Komdigi/Nawala).
			result, err := c.CheckOne(ctx, "reddit.com")
			checkCancel()

			if err == nil {
				status := "not blocked"
				if result.Blocked {
					status = "BLOCKED"
				}
				fmt.Printf("[%s] %-15s -> %-12s (Server: %s, Keyword: %s)\n",
					time.Now().Format("15:04:05.000"),
					result.Domain,
					status,
					result.Server,
					// We fetch the current keyword just for display purposes
					getCurrentKeywordFor(c, result.Server),
				)
			}

			// Wait a bit before the next check to avoid spamming output too fast.
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// 3. Main thread: Wait 2 seconds, then HOT-RELOAD the configuration.
	time.Sleep(2 * time.Second)
	fmt.Println("\n>>> TRIGGERING HOT-RELOAD: Adding Nawala Block Server...")

	// The background goroutine is actively reading from the checker.
	// This SetServers call safely acquires a write lock, updates the slice,
	// and releases the lock without breaking in-flight queries.
	c.SetServers(nawala.DNSServer{
		Address:   "180.131.144.144",
		Keyword:   "internetpositif",
		QueryType: "A",
	})

	time.Sleep(3 * time.Second)
	fmt.Println("\n>>> TRIGGERING HOT-RELOAD: Changing Keyword...")

	// Demonstrating that replacing by Address completely updates Keyword and QueryType.
	c.SetServers(nawala.DNSServer{
		Address:   "180.131.144.144",
		Keyword:   "changed-keyword",
		QueryType: "A",
	})

	time.Sleep(3 * time.Second)
	fmt.Println("\n>>> Example complete.")
	cancel()  // Trigger shutdown
	wg.Wait() // Wait for background routine to finish
}

// Helper function to just display the currently configured keyword for an IP.
func getCurrentKeywordFor(c *nawala.Checker, ip string) string {
	servers := c.Servers()
	for _, s := range servers {
		if s.Address == ip {
			return s.Keyword
		}
	}
	return "UNKNOWN"
}

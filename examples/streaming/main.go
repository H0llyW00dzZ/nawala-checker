// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main demonstrates streaming domain checking with the nawala checker SDK.
//
// This example shows how to use CheckStream for constant-memory processing
// of large domain lists, processing results as they arrive.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

func main() {
	// Create a checker with default Nawala DNS servers.
	c := nawala.New()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Domains to check - can be millions with constant memory usage.
	domains := []string{
		"exam_ple.com",
		"google.com",
		"reddit.com",
		"github.com",
		"stackoverflow.com",
		"youtube.com",
	}

	fmt.Println("=== Nawala DNS Streaming Check ===")
	fmt.Printf("Processing %d domains with constant memory usage...\n\n", len(domains))

	// Create bidirectional channels for streaming.
	in := make(chan string)
	out := make(chan nawala.Result, c.Concurrency())

	// Start domain input goroutine.
	go func() {
		defer close(in)
		for _, domain := range domains {
			in <- domain
		}
	}()

	// Start result processing goroutine.
	go func() {
		for result := range out {
			status := "not blocked"
			if result.Blocked {
				status = "BLOCKED"
			}
			if result.Error != nil {
				status = fmt.Sprintf("error: %v", result.Error)
			}
			fmt.Printf("  %-20s %s (server: %s)\n", result.Domain, status, result.Server)
		}
	}()

	// Perform streaming check.
	err := c.CheckStream(ctx, nawala.Stream{In: in, Out: out})
	if err != nil {
		log.Fatalf("streaming check failed: %v", err)
	}

	fmt.Println("\nStreaming check completed successfully!")
}

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main demonstrates basic usage of the nawala checker SDK.
//
// It checks a list of domains against the default Nawala DNS servers
// and prints whether each domain is blocked or not.
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

	// Check multiple domains concurrently.
	domains := []string{
		"exam_ple.com",
		"google.com",
		"reddit.com",
		"github.com",
	}

	fmt.Println("=== Nawala DNS Blocker Check ===")
	fmt.Println()

	results, err := c.Check(ctx, domains...)
	if err != nil {
		log.Fatalf("check failed: %v", err)
	}

	for _, r := range results {
		status := "not blocked"
		if r.Blocked {
			status = "BLOCKED"
		}
		if r.Error != nil {
			status = fmt.Sprintf("error: %v", r.Error)
		}
		fmt.Printf("  %-20s %s (server: %s)\n", r.Domain, status, r.Server)
	}
}

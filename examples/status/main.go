// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main demonstrates the DNS server health check feature
// of the nawala checker SDK.
//
// It queries all configured Nawala DNS servers and reports their
// online status and latency.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

func main() {
	c := nawala.New()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Println("=== Nawala DNS Server Status ===")
	fmt.Println()

	statuses, err := c.DNSStatus(ctx)
	if err != nil {
		log.Fatalf("status check failed: %v", err)
	}

	for _, s := range statuses {
		status := "OFFLINE"
		if s.Online {
			status = fmt.Sprintf("ONLINE (%dms)", s.LatencyMs)
		}
		fmt.Printf("  %-18s %s\n", s.Server, status)
		if s.Error != nil {
			fmt.Printf("    error: %v\n", s.Error)
		}
	}
}

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main demonstrates TCP/TLS connection pooling with the nawala checker SDK.
//
// This example shows how to enable keep-alive connection pooling for
// improved performance with DNS over TCP/TLS servers.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
	"github.com/miekg/dns"
)

// startMockTCPServer starts a mock TCP DNS server that supports connection reuse
func startMockTCPServer() (*dns.Server, string) {
	// Create a TCP handler that responds to A queries
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)

		// Add a simple A record response
		if len(r.Question) > 0 && r.Question[0].Qtype == dns.TypeA {
			rr, _ := dns.NewRR(fmt.Sprintf("%s 300 IN A 127.0.0.1", r.Question[0].Name))
			m.Answer = append(m.Answer, rr)
		}

		w.WriteMsg(m)
	})

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	// Start TCP server
	server := &dns.Server{
		Addr:    addr,
		Net:     "tcp",
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("mock TCP server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	return server, addr
}

func main() {
	// Start a mock TCP DNS server for demonstration
	mockServer, serverAddr := startMockTCPServer()
	defer mockServer.Shutdown()

	// Create a checker with TCP protocol and connection pooling enabled.
	// Pool size defaults to min(concurrency, 10) = 10 for this example.
	c := nawala.New(
		// Use TCP transport for connection reuse demonstration.
		nawala.WithProtocol("tcp"),

		// Enable connection pooling with default size.
		nawala.WithKeepAlive(0), // 0 means use default sizing

		// Use our mock TCP server
		nawala.WithServers([]nawala.DNSServer{
			{
				Address:   serverAddr, // Our mock TCP server
				Keyword:   "blocked",  // Won't match our responses
				QueryType: "A",
			},
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Domains to check - connection pooling will reuse TCP connections to our mock server.
	domains := []string{
		"example.com",
		"test.com",
		"demo.com",
		"sample.com",
	}

	fmt.Println("=== Nawala DNS TCP Connection Pooling ===")
	fmt.Printf("Checking %d domains using TCP with connection pooling on mock server...\n\n", len(domains))

	// First check - establishes pooled connections.
	fmt.Println("First batch (connection establishment):")
	results1, err := c.Check(ctx, domains...)
	if err != nil {
		log.Fatalf("first check failed: %v", err)
	}

	for _, r := range results1 {
		status := "not blocked"
		if r.Blocked {
			status = "BLOCKED"
		}
		if r.Error != nil {
			status = fmt.Sprintf("error: %v", r.Error)
		}
		fmt.Printf("  %-20s %s (server: %s)\n", r.Domain, status, r.Server)
	}

	fmt.Println("\nSecond batch (connection reuse from pool):")
	// Second check - reuses pooled connections for better performance.
	results2, err := c.Check(ctx, domains...)
	if err != nil {
		log.Fatalf("second check failed: %v", err)
	}

	for _, r := range results2 {
		status := "not blocked"
		if r.Blocked {
			status = "BLOCKED"
		}
		if r.Error != nil {
			status = fmt.Sprintf("error: %v", r.Error)
		}
		fmt.Printf("  %-20s %s (server: %s)\n", r.Domain, status, r.Server)
	}

	// Clean up connection pools.
	c.Close()

	fmt.Println("\nConnection pooling example completed!")
	fmt.Println("Note: Performance improvement is most noticeable with DNS over TLS (DoT)")
	fmt.Println("and servers that support persistent TCP connections.")
}

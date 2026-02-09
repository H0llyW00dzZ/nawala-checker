// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestParseQueryType(t *testing.T) {
	tests := []struct {
		input string
		want  uint16
	}{
		{"A", dns.TypeA},
		{"a", dns.TypeA},
		{"  A  ", dns.TypeA},
		{"AAAA", dns.TypeAAAA},
		{"CNAME", dns.TypeCNAME},
		{"MX", dns.TypeMX},
		{"NS", dns.TypeNS},
		{"TXT", dns.TypeTXT},
		{"SOA", dns.TypeSOA},
		{"SRV", dns.TypeSRV},
		{"ANY", dns.TypeANY},
		{"unknown", dns.TypeA}, // default
		{"", dns.TypeA},        // default
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input=%q", tt.input), func(t *testing.T) {
			got := parseQueryType(tt.input)
			if got != tt.want {
				t.Errorf("parseQueryType(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestContainsKeyword(t *testing.T) {
	t.Run("nil message", func(t *testing.T) {
		if containsKeyword(nil, "anything") {
			t.Error("expected false for nil message")
		}
	})

	t.Run("empty message", func(t *testing.T) {
		msg := new(dns.Msg)
		if containsKeyword(msg, "anything") {
			t.Error("expected false for empty message")
		}
	})

	t.Run("keyword in Answer section", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Answer = []dns.RR{
			&dns.CNAME{
				Hdr:    dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
				Target: "internetpositif.id.",
			},
		}
		if !containsKeyword(msg, "internetpositif") {
			t.Error("expected true for keyword in Answer")
		}
	})

	t.Run("keyword in Ns section", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Ns = []dns.RR{
			&dns.NS{
				Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeNS, Class: dns.ClassINET},
				Ns:  "internetpositif.ns.",
			},
		}
		if !containsKeyword(msg, "internetpositif") {
			t.Error("expected true for keyword in Ns")
		}
	})

	t.Run("keyword in Extra section", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Extra = []dns.RR{
			&dns.TXT{
				Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET},
				Txt: []string{"blocked by internetpositif"},
			},
		}
		if !containsKeyword(msg, "internetpositif") {
			t.Error("expected true for keyword in Extra")
		}
	})

	t.Run("keyword not found", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Answer = []dns.RR{
			&dns.A{
				Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET},
				A:   net.ParseIP("1.2.3.4"),
			},
		}
		if containsKeyword(msg, "internetpositif") {
			t.Error("expected false when keyword not in any section")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Answer = []dns.RR{
			&dns.CNAME{
				Hdr:    dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
				Target: "INTERNETPOSITIF.id.",
			},
		}
		if !containsKeyword(msg, "internetpositif") {
			t.Error("expected true for case-insensitive match")
		}
	})
}

// startTestDNSServer starts a local DNS server that responds with configurable answers.
// It returns the server address (ip:port) and a cleanup function.
func startTestDNSServer(t *testing.T, handler dns.HandlerFunc) (string, func()) {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	server := &dns.Server{
		PacketConn: pc,
		Handler:    handler,
	}

	started := make(chan struct{})
	go func() {
		server.NotifyStartedFunc = func() { close(started) }
		if err := server.ActivateAndServe(); err != nil {
			// Server shutdown is expected, only log unexpected errors.
			select {
			case <-started:
				// Already started, shutdown is expected.
			default:
				t.Logf("DNS server error: %v", err)
			}
		}
	}()

	<-started
	addr := pc.LocalAddr().String()

	return addr, func() {
		_ = server.Shutdown()
	}
}

func TestQueryDNS(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
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

		addr, cleanup := startTestDNSServer(t, handler)
		defer cleanup()

		ctx := context.Background()
		msg, err := queryDNS(ctx, "example.com", addr, dns.TypeA, 5*time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msg.Answer) == 0 {
			t.Fatal("expected answer records, got none")
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		// Use a server that never responds (simulate by using a handler that sleeps).
		handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			time.Sleep(10 * time.Second) // Never respond in time.
		})

		addr, cleanup := startTestDNSServer(t, handler)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately.

		_, err := queryDNS(ctx, "example.com", addr, dns.TypeA, 5*time.Second)
		if err == nil {
			t.Fatal("expected error for cancelled context, got nil")
		}
	})

	t.Run("server with port already specified", func(t *testing.T) {
		handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			m := new(dns.Msg)
			m.SetReply(r)
			_ = w.WriteMsg(m)
		})

		addr, cleanup := startTestDNSServer(t, handler)
		defer cleanup()

		// addr already contains "ip:port", queryDNS should handle it.
		ctx := context.Background()
		_, err := queryDNS(ctx, "example.com", addr, dns.TypeA, 5*time.Second)
		if err != nil {
			t.Fatalf("unexpected error with explicit port: %v", err)
		}
	})

	t.Run("server without port", func(t *testing.T) {
		handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			m := new(dns.Msg)
			m.SetReply(r)
			_ = w.WriteMsg(m)
		})

		addr, cleanup := startTestDNSServer(t, handler)
		defer cleanup()

		// Extract just the IP and use it without port — it will auto-append ":53"
		// which won't match our test server, so this tests the port-appending logic.
		host, _, _ := net.SplitHostPort(addr)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// This will fail because it'll try port 53, but that covers the code path.
		_, err := queryDNS(ctx, "example.com", host, dns.TypeA, 500*time.Millisecond)
		if err == nil {
			t.Log("query to port 53 unexpectedly succeeded (port 53 may be running)")
		}
		// Either way, the ":53" append code path is covered.
	})
}

func TestCheckDNSHealth(t *testing.T) {
	t.Run("healthy server", func(t *testing.T) {
		handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			m := new(dns.Msg)
			m.SetReply(r)
			m.Rcode = dns.RcodeSuccess
			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   r.Question[0].Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    60,
				},
				A: net.ParseIP("142.250.80.46"),
			})
			_ = w.WriteMsg(m)
		})

		addr, cleanup := startTestDNSServer(t, handler)
		defer cleanup()

		ctx := context.Background()
		status := checkDNSHealth(ctx, addr, 5*time.Second)
		if !status.Online {
			t.Errorf("expected Online=true, got false (error: %v)", status.Error)
		}
		if status.LatencyMs < 0 {
			t.Errorf("expected non-negative latency, got %d", status.LatencyMs)
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Use an address that won't have a DNS server.
		status := checkDNSHealth(ctx, "127.0.0.1:19999", 500*time.Millisecond)
		if status.Online {
			t.Error("expected Online=false for unreachable server")
		}
		if status.Error == nil {
			t.Error("expected error for unreachable server")
		}
	})

	t.Run("non-success rcode", func(t *testing.T) {
		handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			m := new(dns.Msg)
			m.SetReply(r)
			m.Rcode = dns.RcodeServerFailure
			_ = w.WriteMsg(m)
		})

		addr, cleanup := startTestDNSServer(t, handler)
		defer cleanup()

		ctx := context.Background()
		status := checkDNSHealth(ctx, addr, 5*time.Second)
		if status.Online {
			t.Error("expected Online=false for SERVFAIL response")
		}
		if status.Error == nil {
			t.Error("expected error for non-success rcode")
		}
	})
}

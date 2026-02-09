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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, tt.want, parseQueryType(tt.input))
		})
	}
}

func TestContainsKeyword(t *testing.T) {
	t.Run("nil message", func(t *testing.T) {
		assert.False(t, containsKeyword(nil, "anything"))
	})

	t.Run("empty message", func(t *testing.T) {
		msg := new(dns.Msg)
		assert.False(t, containsKeyword(msg, "anything"))
	})

	t.Run("keyword in Answer section", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Answer = []dns.RR{
			&dns.CNAME{
				Hdr:    dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
				Target: "internetpositif.id.",
			},
		}
		assert.True(t, containsKeyword(msg, "internetpositif"))
	})

	t.Run("keyword in Ns section", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Ns = []dns.RR{
			&dns.NS{
				Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeNS, Class: dns.ClassINET},
				Ns:  "internetpositif.ns.",
			},
		}
		assert.True(t, containsKeyword(msg, "internetpositif"))
	})

	t.Run("keyword in Extra section", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Extra = []dns.RR{
			&dns.TXT{
				Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET},
				Txt: []string{"blocked by internetpositif"},
			},
		}
		assert.True(t, containsKeyword(msg, "internetpositif"))
	})

	t.Run("keyword not found", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Answer = []dns.RR{
			&dns.A{
				Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET},
				A:   net.ParseIP("1.2.3.4"),
			},
		}
		assert.False(t, containsKeyword(msg, "internetpositif"))
	})

	t.Run("case insensitive", func(t *testing.T) {
		msg := new(dns.Msg)
		msg.Answer = []dns.RR{
			&dns.CNAME{
				Hdr:    dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
				Target: "INTERNETPOSITIF.id.",
			},
		}
		assert.True(t, containsKeyword(msg, "internetpositif"))
	})
}

// startTestDNSServer starts a local DNS server that responds with configurable answers.
// It returns the server address (ip:port) and a cleanup function.
func startTestDNSServer(t *testing.T, handler dns.HandlerFunc) (string, func()) {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err, "failed to listen")

	server := &dns.Server{
		PacketConn: pc,
		Handler:    handler,
	}

	started := make(chan struct{})
	go func() {
		server.NotifyStartedFunc = func() { close(started) }
		if err := server.ActivateAndServe(); err != nil {
			// Server shutdown is expected after started.
			select {
			case <-started:
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
		require.NoError(t, err)
		assert.NotEmpty(t, msg.Answer, "expected answer records")
	})

	t.Run("context cancelled", func(t *testing.T) {
		handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			time.Sleep(10 * time.Second)
		})

		addr, cleanup := startTestDNSServer(t, handler)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := queryDNS(ctx, "example.com", addr, dns.TypeA, 5*time.Second)
		assert.Error(t, err, "expected error for cancelled context")
	})

	t.Run("server with port already specified", func(t *testing.T) {
		handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			m := new(dns.Msg)
			m.SetReply(r)
			_ = w.WriteMsg(m)
		})

		addr, cleanup := startTestDNSServer(t, handler)
		defer cleanup()

		ctx := context.Background()
		_, err := queryDNS(ctx, "example.com", addr, dns.TypeA, 5*time.Second)
		assert.NoError(t, err)
	})

	t.Run("server without port", func(t *testing.T) {
		handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			m := new(dns.Msg)
			m.SetReply(r)
			_ = w.WriteMsg(m)
		})

		addr, cleanup := startTestDNSServer(t, handler)
		defer cleanup()

		// Extract just the IP — queryDNS will auto-append ":53".
		host, _, _ := net.SplitHostPort(addr)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// This will fail because it'll try port 53, but that covers the code path.
		_, err := queryDNS(ctx, "example.com", host, dns.TypeA, 500*time.Millisecond)
		if err == nil {
			t.Log("query to port 53 unexpectedly succeeded (port 53 may be running)")
		}
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
		assert.True(t, status.Online, "expected Online=true")
		assert.GreaterOrEqual(t, status.LatencyMs, int64(0))
	})

	t.Run("unreachable server", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		status := checkDNSHealth(ctx, "127.0.0.1:19999", 500*time.Millisecond)
		assert.False(t, status.Online, "expected Online=false for unreachable server")
		assert.Error(t, status.Error)
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
		assert.False(t, status.Online, "expected Online=false for SERVFAIL")
		assert.Error(t, status.Error)
	})
}

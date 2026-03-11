// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
)

// collectDomains is a helper wrapper for tests to consume streamDomains into a slice.
func collectDomains(args []string, filePath string) ([]string, error) {
	ctx := context.Background()
	in := make(chan string)

	errCh := make(chan error, 1)
	go func() {
		errCh <- streamDomains(ctx, args, filePath, in)
	}()

	var domains []string
	for d := range in {
		domains = append(domains, d)
	}

	return domains, <-errCh
}

// newCheckCmd creates a fresh check command for isolated testing.
func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "check [domains...]",
		Args: cobra.ArbitraryArgs,
		RunE: runCheck,
	}
	cmd.Flags().StringP("file", "f", "", "path to a .txt file with one domain per line")
	cmd.Flags().StringP("output", "o", "", "write results to a file instead of stdout")
	cmd.Flags().StringSlice("format", []string{"text"}, "output format (text, json, html, xlsx)")
	return cmd
}

func createMockDNSServer(t *testing.T) (string, func()) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 512)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				close(done)
				return
			}
			if n >= 12 {
				// Create a valid DNS response (Standard query response, No error)
				resp := append([]byte(nil), buf[:n]...) // copy request
				resp[2] |= 0x80                         // Set QR bit (Response)
				_, _ = conn.WriteToUDP(resp, addr)
			}
		}
	}()
	return conn.LocalAddr().String(), func() {
		_ = conn.Close()
		<-done
	}
}

// collectDomainsCtx is like collectDomains but accepts a context,
// enabling tests for context-cancellation paths.
func collectDomainsCtx(ctx context.Context, args []string, filePath string) ([]string, error) {
	in := make(chan string)

	errCh := make(chan error, 1)
	go func() {
		errCh <- streamDomains(ctx, args, filePath, in)
	}()

	var domains []string
	for d := range in {
		domains = append(domains, d)
	}

	return domains, <-errCh
}

// createSlowDNSServer starts a proper miekg/dns server that delays before
// responding, used to guarantee context-deadline expiration during a DNS query.
// Uses dns.Server for cross-platform reliability (matching the SDK internal tests).
func createSlowDNSServer(t *testing.T, delay time.Duration) (string, func()) {
	t.Helper()

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		time.Sleep(delay)
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP("93.184.216.34"),
		})
		_ = w.WriteMsg(m)
	})

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	server := &dns.Server{
		PacketConn: pc,
		Handler:    handler,
	}

	started := make(chan error, 1)
	go func() {
		server.NotifyStartedFunc = func() { started <- nil }
		if err := server.ActivateAndServe(); err != nil {
			select {
			case started <- err:
			default:
			}
		}
	}()

	if err := <-started; err != nil {
		t.Fatal(err)
	}

	return pc.LocalAddr().String(), func() { _ = server.Shutdown() }
}

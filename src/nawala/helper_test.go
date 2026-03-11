// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

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

	started := make(chan error, 1)
	go func() {
		server.NotifyStartedFunc = func() { started <- nil }
		if err := server.ActivateAndServe(); err != nil {
			// If the channel is not full, it means startup failed (NotifyStartedFunc didn't run).
			// If full (or drained continuously), we try to send.
			// Ideally, we only want to signal startup failure.
			select {
			case started <- err:
			default:
				// Startup already signaled (success) or channel full.
				// Just log the error as it happened after start.
				t.Logf("DNS server error: %v", err)
			}
		}
	}()

	if err := <-started; err != nil {
		require.NoError(t, err, "failed to start server")
	}
	addr := pc.LocalAddr().String()

	return addr, func() {
		_ = server.Shutdown()
	}
}

// startBlockingDNSServer starts a local DNS server that responds with a CNAME
// to "internetpositif.id." to simulate Nawala blocking behavior.
func startBlockingDNSServer(t *testing.T) (string, func()) {
	t.Helper()

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.CNAME{
			Hdr:    dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
			Target: "internetpositif.id.",
		})
		_ = w.WriteMsg(m)
	})

	return startTestDNSServer(t, handler)
}

// startNormalDNSServer starts a local DNS server that responds normally (not blocked).
func startNormalDNSServer(t *testing.T) (string, func()) {
	t.Helper()

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET},
			A:   net.ParseIP("1.2.3.4"),
		})
		_ = w.WriteMsg(m)
	})

	return startTestDNSServer(t, handler)
}

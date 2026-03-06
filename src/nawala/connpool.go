// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// connPool is a fixed-size pool of persistent [dns.Conn] connections to a
// single DNS server. It is only used when the protocol is TCP or TCP-TLS,
// where connection establishment overhead (TCP handshake / TLS handshake)
// is non-trivial and worth amortising across multiple queries.
//
// Design principles:
//   - The pool is backed by a buffered channel of [*dns.Conn]. Getting a
//     connection is non-blocking: if the channel is empty a new connection is
//     dialled immediately. Returning a connection is also non-blocking: if the
//     channel is full the connection is closed and discarded.
//   - There are no background goroutines. Connections that have gone stale
//     (e.g. the server enforced an idle timeout) will surface as an [io.EOF]
//     or similar error on the next [connPool.exchange] call. The broken
//     connection is discarded and the caller's existing retry / failover logic
//     handles the rest.
//   - [connPool.close] drains and closes every idle connection in the pool.
//     It is called from [Checker.Close].
type connPool struct {
	client *dns.Client
	addr   string
	pool   chan *dns.Conn
}

// newConnPool constructs a [connPool] for the given client and server address.
// size is the maximum number of idle connections to keep open simultaneously.
func newConnPool(client *dns.Client, addr string, size int) *connPool {
	return &connPool{
		client: client,
		addr:   addr,
		pool:   make(chan *dns.Conn, size),
	}
}

// get returns an idle connection from the pool, dialling a new one when the
// pool is empty. The returned connection must be passed back to [connPool.put]
// after use if it is still healthy.
func (p *connPool) get(ctx context.Context) (*dns.Conn, error) {
	select {
	case conn := <-p.pool:
		return conn, nil
	default:
		return p.client.DialContext(ctx, p.addr)
	}
}

// put returns conn to the pool. If the pool is already at capacity the
// connection is closed instead. put is a no-op when conn is nil.
func (p *connPool) put(conn *dns.Conn) {
	if conn == nil {
		return
	}
	select {
	case p.pool <- conn:
	default:
		_ = conn.Close()
	}
}

// exchange acquires a connection, executes a single DNS exchange, and returns
// the connection to the pool on success. On any transport error the connection
// is discarded so the next call dials fresh.
//
// The error handling contract mirrors [queryDNS]: the caller is responsible
// for timeout and NXDOMAIN classification; exchange only handles the
// connection lifecycle.
func (p *connPool) exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, time.Duration, error) {
	conn, err := p.get(ctx)
	if err != nil {
		return nil, 0, err
	}

	r, rtt, err := p.client.ExchangeWithConnContext(ctx, msg, conn)
	if err != nil {
		// The connection may be broken (idle timeout, EOF, etc.).
		// Close it and do not return it to the pool; the caller's
		// retry / failover logic will handle the error.
		_ = conn.Close()

		// Transparently redial once for the common "idle connection
		// expired" case (e.g. server-side TCP keepalive timeout).
		// If the fresh dial or exchange also fails we surface that error.
		if isConnError(err) {
			conn2, dialErr := p.client.DialContext(ctx, p.addr)
			if dialErr != nil {
				return nil, 0, err // return the original error
			}
			r2, rtt2, err2 := p.client.ExchangeWithConnContext(ctx, msg, conn2)
			if err2 != nil {
				_ = conn2.Close()
				return nil, 0, err2
			}
			p.put(conn2)
			return r2, rtt2, nil
		}

		return nil, 0, err
	}

	p.put(conn)
	return r, rtt, nil
}

// isConnError reports whether err indicates a broken or stale connection that
// warrants a transparent redial. It returns false for application-level errors
// (e.g. context cancellation, deadlines) so those are surfaced to the caller.
//
// Sources of stale-connection errors according to miekg/dns:
//   - [dns.ErrConnEmpty]   — underlying net.Conn is nil (conn was closed and
//     its internal Conn field set to nil).
//   - [io.EOF] / [io.ErrUnexpectedEOF] — server closed the TCP connection.
//   - *[net.OpError] wrapping an OS-level string such as
//     "use of closed network connection" (internal/poll.ErrNetClosing) or
//     "broken pipe" (syscall.EPIPE); both occur when writing to a locally
//     closed socket.
func isConnError(err error) bool {
	if err == nil {
		return false
	}
	// dns.ErrConnEmpty: the *dns.Conn had its net.Conn set to nil.
	if errors.Is(err, dns.ErrConnEmpty) {
		return true
	}
	// Raw EOF variants returned by ReadMsg on a server-closed connection.
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// *net.OpError wraps OS-level errors when the local end was closed or the
	// remote peer reset the connection.
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Err != nil {
		msg := opErr.Err.Error()
		if strings.Contains(msg, "use of closed network connection") ||
			strings.Contains(msg, "broken pipe") ||
			strings.Contains(msg, "connection reset by peer") {
			return true
		}
	}
	return false
}

// close drains all idle connections from the pool and closes them.
// It is safe to call multiple times.
func (p *connPool) close() {
	for {
		select {
		case conn := <-p.pool:
			_ = conn.Close()
		default:
			return
		}
	}
}

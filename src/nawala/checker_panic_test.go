// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// panicCache is a Cache implementation that panics on Get.
type panicCache struct{}

func (c *panicCache) Get(key string) (Result, bool) {
	panic("cache panic")
}

func (c *panicCache) Set(key string, result Result) {
	// No-op
}

func (c *panicCache) Flush() {
	// No-op
}

func TestCheckPanicRecovery(t *testing.T) {
	// Start a normal server so the check proceeds to the cache lookup
	addr, cleanup := startNormalDNSServer(t)
	defer cleanup()

	c := New(
		WithServers([]DNSServer{
			{Address: addr, Keyword: "internetpositif", QueryType: "A"},
		}),
		WithCache(&panicCache{}), // Injected faulty cache
	)

	ctx := context.Background()
	results, err := c.Check(ctx, "example.com")

	// The check should return success (nil error on the top level),
	// but the specific result for the domain should allow verifying the error.
	// NOTE: In the current implementation of Checker.Check, it returns ([]Result, error).
	// If a panic is caught, we expect it to be populated in the Result.Error.
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Error(t, results[0].Error, "expected error in result due to panic")
	assert.True(t, errors.Is(results[0].Error, ErrInternalPanic), "expected ErrInternalPanic, got: %v", results[0].Error)
}

func TestDNSStatusPanicRecovery(t *testing.T) {
	// To exercise the recover() block in DNSStatus goroutines (checker.go
	// lines 212-219), we nil out the DNS client after construction.
	// This causes a nil pointer dereference panic inside checkDNSHealth →
	// queryDNS → q.client.ExchangeContext(), which the deferred recover()
	// must catch and wrap as ErrInternalPanic.
	c := New(
		WithServers([]DNSServer{
			{Address: "127.0.0.1:19998", Keyword: "test", QueryType: "A"},
			{Address: "127.0.0.1:19999", Keyword: "test", QueryType: "A"},
		}),
	)
	// Sabotage the client to force a nil pointer panic inside the goroutine.
	c.dnsClient = nil

	ctx := context.Background()
	assert.NotPanics(t, func() {
		statuses, _ := c.DNSStatus(ctx)
		require.Len(t, statuses, 2)
		for i, s := range statuses {
			t.Logf("status[%d] server=%s error=%v", i, s.Server, s.Error)
			assert.False(t, s.Online, "status[%d] expected offline", i)
			assert.Error(t, s.Error, "status[%d] expected error", i)
			assert.ErrorIs(t, s.Error, ErrInternalPanic, "status[%d] expected ErrInternalPanic", i)
		}
	})
}

func TestCheckContextCancellationEarly(t *testing.T) {
	// Test that if context is cancelled, we stop processing.
	c := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	domains := []string{"a.com", "b.com", "c.com"}
	results, err := c.Check(ctx, domains...)

	assert.ErrorIs(t, err, context.Canceled)
	require.Len(t, results, 3)
	for i, r := range results {
		assert.ErrorIs(t, r.Error, context.Canceled, "result[%d]", i)
	}
}

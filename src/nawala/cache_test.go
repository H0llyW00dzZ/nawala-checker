// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCacheGetSet(t *testing.T) {
	c := newMemoryCache(5 * time.Minute)

	// Miss on empty cache.
	_, ok := c.Get("miss")
	assert.False(t, ok, "expected miss on empty cache")

	// Set and hit.
	want := Result{Domain: "example.com", Blocked: true, Server: "1.2.3.4"}
	c.Set("hit", want)

	got, ok := c.Get("hit")
	require.True(t, ok, "expected hit after Set")
	assert.Equal(t, want.Domain, got.Domain)
	assert.Equal(t, want.Blocked, got.Blocked)
	assert.Equal(t, want.Server, got.Server)
}

func TestMemoryCacheExpiration(t *testing.T) {
	c := newMemoryCache(50 * time.Millisecond)

	c.Set("expiring", Result{Domain: "test.com"})

	// Immediately should be a hit.
	_, ok := c.Get("expiring")
	require.True(t, ok, "expected hit before expiration")

	// Wait for expiration.
	time.Sleep(100 * time.Millisecond)

	_, ok = c.Get("expiring")
	assert.False(t, ok, "expected miss after expiration")

	// Verify the expired entry was lazily deleted.
	c.mu.RLock()
	_, exists := c.entries["expiring"]
	c.mu.RUnlock()
	assert.False(t, exists, "expected expired entry to be lazily deleted")
}

func TestMemoryCacheFlush(t *testing.T) {
	c := newMemoryCache(5 * time.Minute)

	c.Set("a", Result{Domain: "a.com"})
	c.Set("b", Result{Domain: "b.com"})

	c.Flush()

	_, ok := c.Get("a")
	assert.False(t, ok, "expected miss after Flush for key 'a'")

	_, ok = c.Get("b")
	assert.False(t, ok, "expected miss after Flush for key 'b'")
}

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"testing"
	"time"
)

func TestMemoryCacheGetSet(t *testing.T) {
	c := newMemoryCache(5 * time.Minute)

	// Miss on empty cache.
	_, ok := c.Get("miss")
	if ok {
		t.Error("expected miss on empty cache")
	}

	// Set and hit.
	want := Result{Domain: "example.com", Blocked: true, Server: "1.2.3.4"}
	c.Set("hit", want)

	got, ok := c.Get("hit")
	if !ok {
		t.Fatal("expected hit after Set")
	}
	if got.Domain != want.Domain || got.Blocked != want.Blocked || got.Server != want.Server {
		t.Errorf("Get = %+v, want %+v", got, want)
	}
}

func TestMemoryCacheExpiration(t *testing.T) {
	c := newMemoryCache(50 * time.Millisecond)

	c.Set("expiring", Result{Domain: "test.com"})

	// Immediately should be a hit.
	if _, ok := c.Get("expiring"); !ok {
		t.Fatal("expected hit before expiration")
	}

	// Wait for expiration.
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Get("expiring"); ok {
		t.Error("expected miss after expiration")
	}

	// Verify the expired entry was lazily deleted.
	c.mu.RLock()
	_, exists := c.entries["expiring"]
	c.mu.RUnlock()
	if exists {
		t.Error("expected expired entry to be lazily deleted")
	}
}

func TestMemoryCacheFlush(t *testing.T) {
	c := newMemoryCache(5 * time.Minute)

	c.Set("a", Result{Domain: "a.com"})
	c.Set("b", Result{Domain: "b.com"})

	c.Flush()

	if _, ok := c.Get("a"); ok {
		t.Error("expected miss after Flush")
	}
	if _, ok := c.Get("b"); ok {
		t.Error("expected miss after Flush")
	}
}

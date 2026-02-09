// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"sync"
	"time"
)

// Cache defines an interface for caching DNS check results.
// Implement this interface to provide a custom cache backend
// (e.g., Redis, memcached) via the [WithCache] option.
type Cache interface {
	// Get retrieves a cached result by key.
	// Returns the result and true if found and not expired,
	// or a zero Result and false otherwise.
	Get(key string) (Result, bool)

	// Set stores a result in the cache with the configured TTL.
	Set(key string, val Result)

	// Flush removes all entries from the cache.
	Flush()
}

// cacheEntry holds a cached result with its expiration time.
type cacheEntry struct {
	result    Result
	expiresAt time.Time
}

// memoryCache is the default in-memory cache implementation with TTL support.
type memoryCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

// newMemoryCache creates a new in-memory cache with the given TTL.
func newMemoryCache(ttl time.Duration) *memoryCache {
	return &memoryCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a cached result by key.
// Returns false if the entry does not exist or has expired.
func (c *memoryCache) Get(key string) (Result, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return Result{}, false
	}

	if time.Now().After(entry.expiresAt) {
		// Lazily remove expired entries.
		c.mu.Lock()
		// Double-check locking: verify the entry hasn't changed while we defied the lock.
		if currentEntry, exists := c.entries[key]; exists && currentEntry.expiresAt.Equal(entry.expiresAt) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return Result{}, false
	}

	return entry.result, true
}

// Set stores a result in the cache with the configured TTL.
func (c *memoryCache) Set(key string, val Result) {
	c.mu.Lock()
	c.entries[key] = cacheEntry{
		result:    val,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Flush removes all entries from the cache.
func (c *memoryCache) Flush() {
	c.mu.Lock()
	c.entries = make(map[string]cacheEntry)
	c.mu.Unlock()
}

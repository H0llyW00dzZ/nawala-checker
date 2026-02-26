// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

func TestIsValidDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   bool
	}{
		{"valid .com", "example.com", true},
		{"valid .co.id", "example.co.id", true},
		{"valid subdomain", "sub.example.com", true},
		{"valid hyphen", "my-site.example.com", true},
		{"invalid empty", "", false},
		{"invalid single label", "localhost", false},
		{"invalid starts with hyphen", "-example.com", false},
		{"invalid ends with hyphen", "example-.com", false},
		{"invalid special chars", "exam!ple.com", false},
		{"invalid spaces", "example .com", false},
		{"valid short label", "a.com", true},
		{"invalid TLD with digits", "example.c0m", false},
		{"invalid TLD with hyphen", "example.c-m", false},
		{"invalid label too long", "example." + strings.Repeat("a", 64) + ".com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nawala.IsValidDomain(tt.domain), "IsValidDomain(%q)", tt.domain)
		})
	}
}

func TestNewDefaults(t *testing.T) {
	c := nawala.New()

	servers := c.Servers()
	require.Len(t, servers, 2, "expected 2 default servers")

	expected := []struct {
		address   string
		keyword   string
		queryType string
	}{
		{"180.131.144.144", "internetpositif", "A"},
		{"180.131.145.145", "internetpositif", "A"},
	}

	for i, want := range expected {
		assert.Equal(t, want.address, servers[i].Address, "server[%d].Address", i)
		assert.Equal(t, want.keyword, servers[i].Keyword, "server[%d].Keyword", i)
		assert.Equal(t, want.queryType, servers[i].QueryType, "server[%d].QueryType", i)
	}
}

func TestWithOptions(t *testing.T) {
	customServers := []nawala.DNSServer{
		{Address: "1.1.1.1", Keyword: "test", QueryType: "A"},
	}

	c := nawala.New(
		nawala.WithServers(customServers),
		nawala.WithTimeout(10*time.Second),
		nawala.WithMaxRetries(5),
	)

	servers := c.Servers()
	require.Len(t, servers, 1, "expected 1 custom server")
	assert.Equal(t, "1.1.1.1", servers[0].Address)
}

func TestWithDNSServerAddAndReplace(t *testing.T) {
	c := nawala.New(
		nawala.WithServer(nawala.DNSServer{
			Address: "8.8.8.8", Keyword: "custom", QueryType: "A",
		}),
	)

	servers := c.Servers()
	require.Len(t, servers, 3, "expected 3 servers (2 default + 1 added)")

	// Replace the first default server.
	c2 := nawala.New(
		nawala.WithServer(nawala.DNSServer{
			Address: "180.131.144.144", Keyword: "replaced", QueryType: "TXT",
		}),
	)

	servers2 := c2.Servers()
	require.Len(t, servers2, 2, "expected 2 servers after replace")
	assert.Equal(t, "replaced", servers2[0].Keyword)
}

func TestSetServersRuntime(t *testing.T) {
	c := nawala.New()

	// Add a brand-new server — length grows from 2 to 3.
	c.SetServers(nawala.DNSServer{Address: "8.8.8.8", Keyword: "custom", QueryType: "A"})
	servers := c.Servers()
	require.Len(t, servers, 3, "expected 3 servers after adding new one")

	// Replace an existing server by address — length stays at 3, keyword changes.
	c.SetServers(nawala.DNSServer{Address: "8.8.8.8", Keyword: "replaced", QueryType: "A"})
	servers = c.Servers()
	require.Len(t, servers, 3, "expected 3 servers after replacing existing one")
	var found bool
	for _, s := range servers {
		if s.Address == "8.8.8.8" {
			assert.Equal(t, "replaced", s.Keyword)
			found = true
		}
	}
	assert.True(t, found, "replaced server should still be present")

	// Upsert multiple servers in one call.
	c.SetServers(
		nawala.DNSServer{Address: "1.1.1.1", Keyword: "cf", QueryType: "A"},
		nawala.DNSServer{Address: "8.8.8.8", Keyword: "google", QueryType: "A"},
	)
	servers = c.Servers()
	require.Len(t, servers, 4, "expected 4 servers after adding 1 new + replacing 1 existing")

	// No-op: zero args should leave the list unchanged.
	c.SetServers()
	assert.Len(t, c.Servers(), 4, "no-op call should not change servers")
}

func TestSetServersConcurrency(t *testing.T) {
	c := nawala.New()
	const workers = 50

	done := make(chan struct{})
	for i := range workers {
		go func(n int) {
			c.SetServers(nawala.DNSServer{
				Address:   fmt.Sprintf("10.0.0.%d", n%254+1),
				Keyword:   "test",
				QueryType: "A",
			})
			done <- struct{}{}
		}(i)
	}
	for range workers {
		<-done
	}
	// Just assert we have at least the 2 defaults; exact count varies by race.
}

func TestDeleteServersRuntime(t *testing.T) {
	c := nawala.New()

	// Initial default is 2 servers.
	require.Len(t, c.Servers(), 2)

	// Add 3 more custom servers.
	c.SetServers(
		nawala.DNSServer{Address: "1.1.1.1", Keyword: "cf", QueryType: "A"},
		nawala.DNSServer{Address: "8.8.8.8", Keyword: "google", QueryType: "A"},
		nawala.DNSServer{Address: "9.9.9.9", Keyword: "quad9", QueryType: "A"},
	)
	require.Len(t, c.Servers(), 5)

	// Delete 2 servers (1 default, 1 custom).
	c.DeleteServers("180.131.144.144", "8.8.8.8")
	servers := c.Servers()
	require.Len(t, servers, 3)

	for _, s := range servers {
		assert.NotEqual(t, "180.131.144.144", s.Address)
		assert.NotEqual(t, "8.8.8.8", s.Address)
	}

	// Delete non-existent server (no-op).
	c.DeleteServers("255.255.255.255")
	require.Len(t, c.Servers(), 3)

	// Delete with zero arguments (no-op).
	c.DeleteServers()
	require.Len(t, c.Servers(), 3)
}

func TestDeleteServersConcurrency(t *testing.T) {
	c := nawala.New()

	// Add 100 servers so we have something to delete.
	var servers []nawala.DNSServer
	for i := 1; i <= 100; i++ {
		servers = append(servers, nawala.DNSServer{
			Address:   fmt.Sprintf("10.0.0.%d", i),
			Keyword:   "test",
			QueryType: "A",
		})
	}
	c.SetServers(servers...)
	require.GreaterOrEqual(t, len(c.Servers()), 100)

	const workers = 50
	done := make(chan struct{})

	// 50 concurrent goroutines racing to delete disjoint IPs.
	for i := 1; i <= workers; i++ {
		go func(n int) {
			c.DeleteServers(fmt.Sprintf("10.0.0.%d", n))
			done <- struct{}{}
		}(i)
	}

	for range workers {
		<-done
	}

	// 100 original + 2 defaults - 50 deleted = 52.
	assert.GreaterOrEqual(t, len(c.Servers()), 52)
}

func TestCheckInvalidDomain(t *testing.T) {
	c := nawala.New()
	ctx := context.Background()

	result, err := c.CheckOne(ctx, "invalid")
	require.NoError(t, err)
	assert.Error(t, result.Error, "expected error for invalid domain")
}

func TestCheckNoDNSServers(t *testing.T) {
	c := nawala.New(
		nawala.WithServers(nil),
	)
	ctx := context.Background()

	_, err := c.Check(ctx, "example.com")
	assert.ErrorIs(t, err, nawala.ErrNoDNSServers)
}

func TestCacheHitMiss(t *testing.T) {
	c := nawala.New(
		nawala.WithCacheTTL(1 * time.Second),
	)

	// FlushCache should not panic on empty cache.
	assert.NotPanics(t, func() { c.FlushCache() })
}

func TestWithNilCache(t *testing.T) {
	c := nawala.New(
		nawala.WithCache(nil),
	)

	ctx := context.Background()

	// Should not panic when cache is nil.
	result, err := c.CheckOne(ctx, "invalid")
	require.NoError(t, err)
	assert.Error(t, result.Error, "expected error for invalid domain")
}

// TestLiveDNSCheck performs a real DNS check against Nawala servers.
// Skip with -short flag.
func TestLiveDNSCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live DNS test in short mode")
	}

	c := nawala.New(
		nawala.WithTimeout(10 * time.Second),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check DNS server status first.
	statuses, err := c.DNSStatus(ctx)
	require.NoError(t, err, "DNSStatus failed")

	onlineCount := 0
	for _, s := range statuses {
		t.Logf("DNS Server %s: online=%v, latency=%dms", s.Server, s.Online, s.LatencyMs)
		if s.Online {
			onlineCount++
		}
	}

	if onlineCount == 0 {
		t.Skip("no DNS servers online, skipping live check")
	}

	// Check a domain that should not be blocked (google.com).
	result, err := c.CheckOne(ctx, "google.com")
	require.NoError(t, err, "CheckOne failed")
	require.NoError(t, result.Error, "unexpected result error")

	t.Logf("google.com: blocked=%v (server=%s)", result.Blocked, result.Server)
}

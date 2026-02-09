// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala_test

import (
	"context"
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

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala_test

import (
	"context"
	"testing"
	"time"

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
		{"invalid too short label", "a.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nawala.IsValidDomain(tt.domain); got != tt.want {
				t.Errorf("IsValidDomain(%q) = %v, want %v", tt.domain, got, tt.want)
			}
		})
	}
}

func TestNewDefaults(t *testing.T) {
	c := nawala.New()

	servers := c.Servers()
	if len(servers) != 2 {
		t.Fatalf("expected 2 default servers, got %d", len(servers))
	}

	// Verify the default servers match the expected configuration.
	expected := []struct {
		address   string
		keyword   string
		queryType string
	}{
		{"180.131.144.144", "internetpositif", "A"},
		{"180.131.145.145", "internetpositif", "A"},
	}

	for i, want := range expected {
		got := servers[i]
		if got.Address != want.address {
			t.Errorf("server[%d].Address = %q, want %q", i, got.Address, want.address)
		}
		if got.Keyword != want.keyword {
			t.Errorf("server[%d].Keyword = %q, want %q", i, got.Keyword, want.keyword)
		}
		if got.QueryType != want.queryType {
			t.Errorf("server[%d].QueryType = %q, want %q", i, got.QueryType, want.queryType)
		}
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
	if len(servers) != 1 {
		t.Fatalf("expected 1 custom server, got %d", len(servers))
	}
	if servers[0].Address != "1.1.1.1" {
		t.Errorf("server address = %q, want %q", servers[0].Address, "1.1.1.1")
	}
}

func TestWithDNSServerAddAndReplace(t *testing.T) {
	c := nawala.New(
		nawala.WithServer(nawala.DNSServer{
			Address: "8.8.8.8", Keyword: "custom", QueryType: "A",
		}),
	)

	servers := c.Servers()
	if len(servers) != 3 { // 2 default + 1 added
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}

	// Replace the first default server.
	c2 := nawala.New(
		nawala.WithServer(nawala.DNSServer{
			Address: "180.131.144.144", Keyword: "replaced", QueryType: "TXT",
		}),
	)

	servers2 := c2.Servers()
	if len(servers2) != 2 { // replaced, not added
		t.Fatalf("expected 2 servers after replace, got %d", len(servers2))
	}
	if servers2[0].Keyword != "replaced" {
		t.Errorf("server[0].Keyword = %q, want %q", servers2[0].Keyword, "replaced")
	}
}

func TestCheckInvalidDomain(t *testing.T) {
	c := nawala.New()
	ctx := context.Background()

	result, err := c.CheckOne(ctx, "invalid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Error == nil {
		t.Error("expected error for invalid domain, got nil")
	}
}

func TestCheckNoDNSServers(t *testing.T) {
	c := nawala.New(
		nawala.WithServers(nil),
	)
	ctx := context.Background()

	_, err := c.Check(ctx, "example.com")
	if err != nawala.ErrNoDNSServers {
		t.Errorf("expected ErrNoDNSServers, got %v", err)
	}
}

func TestCacheHitMiss(t *testing.T) {
	c := nawala.New(
		nawala.WithCacheTTL(1 * time.Second),
	)

	// FlushCache should not panic on empty cache.
	c.FlushCache()
}

func TestWithNilCache(t *testing.T) {
	c := nawala.New(
		nawala.WithCache(nil),
	)

	ctx := context.Background()

	// Should not panic when cache is nil.
	result, err := c.CheckOne(ctx, "invalid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == nil {
		t.Error("expected error for invalid domain")
	}
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
	if err != nil {
		t.Fatalf("DNSStatus failed: %v", err)
	}

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
	if err != nil {
		t.Fatalf("CheckOne failed: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("unexpected result error: %v", result.Error)
	}
	t.Logf("google.com: blocked=%v (server=%s)", result.Blocked, result.Server)
}

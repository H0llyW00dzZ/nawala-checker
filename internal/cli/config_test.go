// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_JSON(t *testing.T) {
	content := `{
		"timeout": "10s",
		"max_retries": 3,
		"cache_ttl": "5m",
		"concurrency": 50,
		"edns0_size": 4096,
		"servers": [
			{"address": "8.8.8.8", "keyword": "blocked", "query_type": "A"}
		]
	}`

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig(%q) error: %v", path, err)
	}

	if cfg.Timeout != "10s" {
		t.Errorf("Timeout = %q, want %q", cfg.Timeout, "10s")
	}
	if cfg.MaxRetries == nil || *cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", cfg.MaxRetries)
	}
	if cfg.CacheTTL != "5m" {
		t.Errorf("CacheTTL = %q, want %q", cfg.CacheTTL, "5m")
	}
	if cfg.Concurrency == nil || *cfg.Concurrency != 50 {
		t.Errorf("Concurrency = %v, want 50", cfg.Concurrency)
	}
	if cfg.EDNS0Size == nil || *cfg.EDNS0Size != 4096 {
		t.Errorf("EDNS0Size = %v, want 4096", cfg.EDNS0Size)
	}
	if len(cfg.Servers) != 1 {
		t.Fatalf("len(Servers) = %d, want 1", len(cfg.Servers))
	}
	if cfg.Servers[0].Address != "8.8.8.8" {
		t.Errorf("Servers[0].Address = %q, want %q", cfg.Servers[0].Address, "8.8.8.8")
	}
}

func TestLoadConfig_YAML(t *testing.T) {
	content := `
timeout: 15s
max_retries: 5
cache_ttl: 10m
concurrency: 200
edns0_size: 2048
servers:
  - address: "1.1.1.1"
    keyword: "cloudflare"
    query_type: "AAAA"
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig(%q) error: %v", path, err)
	}

	if cfg.Timeout != "15s" {
		t.Errorf("Timeout = %q, want %q", cfg.Timeout, "15s")
	}
	if cfg.MaxRetries == nil || *cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %v, want 5", cfg.MaxRetries)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].QueryType != "AAAA" {
		t.Errorf("Servers = %+v, want 1 server with QueryType AAAA", cfg.Servers)
	}
}

func TestLoadConfig_YML(t *testing.T) {
	content := `timeout: 5s`
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig(%q) error: %v", path, err)
	}
	if cfg.Timeout != "5s" {
		t.Errorf("Timeout = %q, want %q", cfg.Timeout, "5s")
	}
}

func TestLoadConfig_UnsupportedFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("key = 'value'"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := loadConfig("/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{invalid json}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte(":\n  :\n    - [invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestConfig_ToOptions_AllFields(t *testing.T) {
	retries := 3
	concurrency := 50
	edns0 := uint16(4096)
	cfg := &Config{
		Timeout:     "10s",
		MaxRetries:  &retries,
		CacheTTL:    "5m",
		Concurrency: &concurrency,
		EDNS0Size:   &edns0,
		Servers: []ServerDef{
			{Address: "8.8.8.8", Keyword: "blocked", QueryType: "A"},
			{Address: "8.8.4.4", Keyword: "blocked", QueryType: "A"},
		},
	}

	opts, err := cfg.toOptions()
	if err != nil {
		t.Fatalf("toOptions() error: %v", err)
	}

	// timeout + retries + cacheTTL + concurrency + edns0 + servers = 6
	if len(opts) != 6 {
		t.Errorf("len(opts) = %d, want 6", len(opts))
	}
}

func TestConfig_ToOptions_Empty(t *testing.T) {
	cfg := &Config{}

	opts, err := cfg.toOptions()
	if err != nil {
		t.Fatalf("toOptions() error: %v", err)
	}

	if len(opts) != 0 {
		t.Errorf("len(opts) = %d, want 0", len(opts))
	}
}

func TestConfig_ToOptions_InvalidTimeout(t *testing.T) {
	cfg := &Config{Timeout: "not-a-duration"}

	_, err := cfg.toOptions()
	if err == nil {
		t.Fatal("expected error for invalid timeout, got nil")
	}
}

func TestConfig_ToOptions_InvalidCacheTTL(t *testing.T) {
	cfg := &Config{CacheTTL: "bad"}

	_, err := cfg.toOptions()
	if err == nil {
		t.Fatal("expected error for invalid cache_ttl, got nil")
	}
}

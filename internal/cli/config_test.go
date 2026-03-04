// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := loadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "10s", cfg.Timeout)
	require.NotNil(t, cfg.MaxRetries)
	assert.Equal(t, 3, *cfg.MaxRetries)

	assert.Equal(t, "5m", cfg.CacheTTL)
	require.NotNil(t, cfg.Concurrency)
	assert.Equal(t, 50, *cfg.Concurrency)

	require.NotNil(t, cfg.EDNS0Size)
	assert.Equal(t, uint16(4096), *cfg.EDNS0Size)

	require.Len(t, cfg.Servers, 1)
	assert.Equal(t, "8.8.8.8", cfg.Servers[0].Address)
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
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := loadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "15s", cfg.Timeout)
	require.NotNil(t, cfg.MaxRetries)
	assert.Equal(t, 5, *cfg.MaxRetries)
	
	require.Len(t, cfg.Servers, 1)
	assert.Equal(t, "AAAA", cfg.Servers[0].QueryType)
}

func TestLoadConfig_YML(t *testing.T) {
	content := `timeout: 5s`
	path := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := loadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "5s", cfg.Timeout)
}

func TestLoadConfig_UnsupportedFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("key = 'value'"), 0644))

	_, err := loadConfig(path)
	assert.Error(t, err, "expected error for unsupported format")
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := loadConfig("/nonexistent/config.json")
	assert.Error(t, err, "expected error for missing file")
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid json}"), 0644))

	_, err := loadConfig(path)
	assert.Error(t, err, "expected error for invalid JSON")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte(":\n  :\n    - [invalid"), 0644))

	_, err := loadConfig(path)
	assert.Error(t, err, "expected error for invalid YAML")
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
	require.NoError(t, err)
	assert.Len(t, opts, 6)
}

func TestConfig_ToOptions_Empty(t *testing.T) {
	cfg := &Config{}

	opts, err := cfg.toOptions()
	require.NoError(t, err)
	assert.Len(t, opts, 0)
}

func TestConfig_ToOptions_InvalidTimeout(t *testing.T) {
	cfg := &Config{Timeout: "not-a-duration"}

	_, err := cfg.toOptions()
	assert.Error(t, err, "expected error for invalid timeout")
}

func TestConfig_ToOptions_InvalidCacheTTL(t *testing.T) {
	cfg := &Config{CacheTTL: "bad"}

	_, err := cfg.toOptions()
	assert.Error(t, err, "expected error for invalid cache_ttl")
}

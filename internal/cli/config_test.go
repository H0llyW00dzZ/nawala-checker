// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_JSON(t *testing.T) {
	content := `{
		"nawala": {
			"configuration": {
				"timeout": "10s",
				"max_retries": 3,
				"cache_ttl": "5m",
				"disable_cache": true,
				"concurrency": 50,
				"edns0_size": 4096,
				"servers": [
					{"address": "8.8.8.8", "keyword": "blocked", "query_type": "A"}
				]
			}
		}
	}`

	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := loadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "10s", cfg.Timeout)
	require.NotNil(t, cfg.MaxRetries)
	assert.Equal(t, 3, *cfg.MaxRetries)

	assert.Equal(t, "5m", cfg.CacheTTL)
	require.NotNil(t, cfg.DisableCache)
	assert.True(t, *cfg.DisableCache)

	require.NotNil(t, cfg.Concurrency)
	assert.Equal(t, 50, *cfg.Concurrency)

	require.NotNil(t, cfg.EDNS0Size)
	assert.Equal(t, uint16(4096), *cfg.EDNS0Size)

	require.Len(t, cfg.Servers, 1)
	assert.Equal(t, "8.8.8.8", cfg.Servers[0].Address)
}

func TestLoadConfig_YAML(t *testing.T) {
	content := `
nawala:
  configuration:
    timeout: 15s
    max_retries: 5
    cache_ttl: 10m
    disable_cache: false
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

	require.NotNil(t, cfg.DisableCache)
	assert.False(t, *cfg.DisableCache)

	require.Len(t, cfg.Servers, 1)
	assert.Equal(t, "AAAA", cfg.Servers[0].QueryType)
}

func TestLoadConfig_YML(t *testing.T) {
	content := "nawala:\n  configuration:\n    timeout: 5s\n"
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
	disableCache := true
	cfg := &Config{
		Timeout:      "10s",
		MaxRetries:   &retries,
		CacheTTL:     "5m",
		DisableCache: &disableCache,
		Concurrency:  &concurrency,
		EDNS0Size:    &edns0,
		Servers: []ServerDef{
			{Address: "8.8.8.8", Keyword: "blocked", QueryType: "A"},
			{Address: "8.8.4.4", Keyword: "blocked", QueryType: "A"},
		},
	}

	opts, err := cfg.toOptions()
	require.NoError(t, err)
	assert.Len(t, opts, 7)
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

// TestConfig_ToOptions_DisableCacheTrue verifies that disable_cache:true
// emits a WithCache(nil) option, which is absent when disable_cache:false.
func TestConfig_ToOptions_DisableCacheTrue(t *testing.T) {
	disableTrue := true
	disableFalse := false

	cfgTrue := &Config{DisableCache: &disableTrue}
	cfgFalse := &Config{DisableCache: &disableFalse}
	cfgNil := &Config{}

	optsTrue, err := cfgTrue.toOptions()
	require.NoError(t, err)
	assert.Len(t, optsTrue, 1, "disable_cache:true should produce exactly 1 option (WithCache(nil))")

	optsFalse, err := cfgFalse.toOptions()
	require.NoError(t, err)
	assert.Len(t, optsFalse, 0, "disable_cache:false should produce no options")

	optsNil, err := cfgNil.toOptions()
	require.NoError(t, err)
	assert.Len(t, optsNil, 0, "nil DisableCache should produce no options")
}

// TestBuildChecker_DisableCacheNoCache verifies the full CLI→SDK path:
// a config file with disable_cache:true must produce a Checker whose
// FlushCache() is a safe no-op (cache is nil, not re-created by New()).
func TestBuildChecker_DisableCacheNoCache(t *testing.T) {
	content := `{"disable_cache": true}`
	path := filepath.Join(t.TempDir(), "nocache.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	saved := configPath
	configPath = path
	defer func() { configPath = saved }()

	checker, _, err := buildChecker()
	require.NoError(t, err)
	require.NotNil(t, checker)

	// FlushCache must not panic when cache is nil.
	assert.NotPanics(t, func() { checker.FlushCache() })
}

func TestConfig_ToOptions_ParseProtocol(t *testing.T) {
	// Set: returns WithProtocol option
	cfg := &Config{Protocol: "tcp"}
	opts, err := cfg.toOptions()
	require.NoError(t, err)
	require.Len(t, opts, 1, "expected one option for Protocol=tcp")

	// Empty: no option
	cfg2 := &Config{}
	opts2, err := cfg2.toOptions()
	require.NoError(t, err)
	assert.Empty(t, opts2)
}

func TestConfig_ToOptions_ParseTLSServerName(t *testing.T) {
	cfg := &Config{TLSServerName: "dns.example.com"}
	opts, err := cfg.toOptions()
	require.NoError(t, err)
	require.Len(t, opts, 1)

	cfg2 := &Config{}
	opts2, err := cfg2.toOptions()
	require.NoError(t, err)
	assert.Empty(t, opts2)
}

func TestConfig_ToOptions_ParseTLSSkipVerify(t *testing.T) {
	t1 := true
	cfg := &Config{TLSSkipVerify: &t1}
	opts, err := cfg.toOptions()
	require.NoError(t, err)
	require.Len(t, opts, 1)

	// false → no option
	f := false
	cfg2 := &Config{TLSSkipVerify: &f}
	opts2, err := cfg2.toOptions()
	require.NoError(t, err)
	assert.Empty(t, opts2)

	// nil → no option
	cfg3 := &Config{}
	opts3, err := cfg3.toOptions()
	require.NoError(t, err)
	assert.Empty(t, opts3)
}

func TestConfig_ParseCommandTimeout_Valid(t *testing.T) {
	cfg := &Config{CommandTimeout: "45s"}
	d, err := cfg.parseCommandTimeout()
	require.NoError(t, err)
	assert.Equal(t, 45*time.Second, d)
}

func TestConfig_ParseCommandTimeout_Empty(t *testing.T) {
	cfg := &Config{}
	d, err := cfg.parseCommandTimeout()
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), d, "empty command_timeout should return zero")
}

func TestConfig_ParseCommandTimeout_Invalid(t *testing.T) {
	cfg := &Config{CommandTimeout: "not-a-duration"}
	_, err := cfg.parseCommandTimeout()
	assert.Error(t, err, "expected error for invalid command_timeout")
}

func TestBuildChecker_CustomCommandTimeout(t *testing.T) {
	content := `{"nawala":{"configuration":{"command_timeout":"2m"}}}`
	path := filepath.Join(t.TempDir(), "cmd_timeout.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	saved := configPath
	configPath = path
	defer func() { configPath = saved }()

	_, cmdTimeout, err := buildChecker()
	require.NoError(t, err)
	assert.Equal(t, 2*time.Minute, cmdTimeout)
}

func TestBuildChecker_InvalidCommandTimeout(t *testing.T) {
	content := `{"nawala":{"configuration":{"command_timeout":"bad"}}}`
	path := filepath.Join(t.TempDir(), "bad_cmd_timeout.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	saved := configPath
	configPath = path
	defer func() { configPath = saved }()

	_, _, err := buildChecker()
	assert.Error(t, err, "expected error for invalid command_timeout")
}

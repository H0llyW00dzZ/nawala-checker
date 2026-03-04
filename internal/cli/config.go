// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
	"gopkg.in/yaml.v3"
)

// Config holds the CLI configuration loaded from a JSON or YAML file.
type Config struct {
	Timeout      string      `json:"timeout"       yaml:"timeout"`
	MaxRetries   *int        `json:"max_retries"   yaml:"max_retries"`
	CacheTTL     string      `json:"cache_ttl"     yaml:"cache_ttl"`
	DisableCache *bool       `json:"disable_cache" yaml:"disable_cache"`
	Concurrency  *int        `json:"concurrency"   yaml:"concurrency"`
	EDNS0Size    *uint16     `json:"edns0_size"    yaml:"edns0_size"`
	Servers      []ServerDef `json:"servers"       yaml:"servers"`
}

// configFile is the top-level envelope that wraps Config in a JSON or YAML
// config file. The file format mirrors the JSON output envelope:
//
//	{"nawala":{"configuration":{...}}}
//
// Or in YAML:
//
//	nawala:
//	  configuration:
//	    timeout: 10s
type configFile struct {
	Nawala struct {
		Configuration Config `json:"configuration" yaml:"configuration"`
	} `json:"nawala" yaml:"nawala"`
}

// ServerDef defines a DNS server in the config file.
type ServerDef struct {
	Address   string `json:"address"    yaml:"address"`
	Keyword   string `json:"keyword"    yaml:"keyword"`
	QueryType string `json:"query_type" yaml:"query_type"`
}

// loadConfig reads and parses a JSON or YAML config file.
// The file must use the nawala envelope format:
//
//	{"nawala":{"configuration":{...}}}  (JSON)
//	
//	nawala:                              (YAML)
//	  configuration:
//	    timeout: 10s
//
// The format is auto-detected by file extension (.json, .yaml, .yml).
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cf configFile
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cf); err != nil {
			return nil, fmt.Errorf("parsing JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cf); err != nil {
			return nil, fmt.Errorf("parsing YAML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format %q (use .json, .yaml, or .yml)", ext)
	}

	cfg := cf.Nawala.Configuration
	return &cfg, nil
}

// toOptions converts a Config into nawala SDK functional options.
// Zero-value fields are ignored, allowing the SDK defaults to apply.
func (c *Config) toOptions() ([]nawala.Option, error) {
	parsers := []func() (nawala.Option, error){
		c.parseTimeout,
		c.parseMaxRetries,
		c.parseCacheTTL,
		c.parseDisableCache,
		c.parseConcurrency,
		c.parseEDNS0Size,
		c.parseServers,
	}

	var opts []nawala.Option
	for _, parse := range parsers {
		opt, err := parse()
		if err != nil {
			return nil, err
		}
		// Only append if the field was defined (not nil)
		if opt != nil {
			opts = append(opts, opt)
		}
	}

	return opts, nil
}

// parseTimeout parses the timeout string into a time.Duration and returns a WithTimeout option.
func (c *Config) parseTimeout() (nawala.Option, error) {
	if c.Timeout == "" {
		return nil, nil
	}
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout %q: %w", c.Timeout, err)
	}
	return nawala.WithTimeout(d), nil
}

// parseMaxRetries returns a WithMaxRetries option if the MaxRetries field is defined.
func (c *Config) parseMaxRetries() (nawala.Option, error) {
	if c.MaxRetries == nil {
		return nil, nil
	}
	return nawala.WithMaxRetries(*c.MaxRetries), nil
}

// parseDisableCache returns a WithCache(nil) option if DisableCache is true.
func (c *Config) parseDisableCache() (nawala.Option, error) {
	if c.DisableCache != nil && *c.DisableCache {
		return nawala.WithCache(nil), nil
	}
	return nil, nil
}

// parseCacheTTL parses the cache_ttl string into a time.Duration and returns a WithCacheTTL option.
func (c *Config) parseCacheTTL() (nawala.Option, error) {
	if c.CacheTTL == "" {
		return nil, nil
	}
	d, err := time.ParseDuration(c.CacheTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid cache_ttl %q: %w", c.CacheTTL, err)
	}
	return nawala.WithCacheTTL(d), nil
}

// parseConcurrency returns a WithConcurrency option if the Concurrency field is defined.
func (c *Config) parseConcurrency() (nawala.Option, error) {
	if c.Concurrency == nil {
		return nil, nil
	}
	return nawala.WithConcurrency(*c.Concurrency), nil
}

// parseEDNS0Size returns a WithEDNS0Size option if the EDNS0Size field is defined.
func (c *Config) parseEDNS0Size() (nawala.Option, error) {
	if c.EDNS0Size == nil {
		return nil, nil
	}
	return nawala.WithEDNS0Size(*c.EDNS0Size), nil
}

// parseServers converts the config ServerDefs into nawala.DNSServer structs and returns a WithServers option.
func (c *Config) parseServers() (nawala.Option, error) {
	if c.Servers == nil {
		return nil, nil
	}
	servers := make([]nawala.DNSServer, len(c.Servers))
	for i, s := range c.Servers {
		servers[i] = nawala.DNSServer{
			Address:   s.Address,
			Keyword:   s.Keyword,
			QueryType: s.QueryType,
		}
	}
	return nawala.WithServers(servers), nil
}

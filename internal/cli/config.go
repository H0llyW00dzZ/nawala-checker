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
	Timeout     string      `json:"timeout"      yaml:"timeout"`
	MaxRetries  *int        `json:"max_retries"   yaml:"max_retries"`
	CacheTTL    string      `json:"cache_ttl"     yaml:"cache_ttl"`
	Concurrency *int        `json:"concurrency"   yaml:"concurrency"`
	EDNS0Size   *uint16     `json:"edns0_size"    yaml:"edns0_size"`
	Servers     []ServerDef `json:"servers"       yaml:"servers"`
}

// ServerDef defines a DNS server in the config file.
type ServerDef struct {
	Address   string `json:"address"    yaml:"address"`
	Keyword   string `json:"keyword"    yaml:"keyword"`
	QueryType string `json:"query_type" yaml:"query_type"`
}

// loadConfig reads and parses a JSON or YAML config file.
// The format is detected by file extension (.json, .yaml, .yml).
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing YAML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format %q (use .json, .yaml, or .yml)", ext)
	}

	return &cfg, nil
}

// toOptions converts a Config into nawala SDK functional options.
// Zero-value fields are ignored, allowing the SDK defaults to apply.
func (c *Config) toOptions() ([]nawala.Option, error) {
	var opts []nawala.Option

	if c.Timeout != "" {
		d, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %q: %w", c.Timeout, err)
		}
		opts = append(opts, nawala.WithTimeout(d))
	}

	if c.MaxRetries != nil {
		opts = append(opts, nawala.WithMaxRetries(*c.MaxRetries))
	}

	if c.CacheTTL != "" {
		d, err := time.ParseDuration(c.CacheTTL)
		if err != nil {
			return nil, fmt.Errorf("invalid cache_ttl %q: %w", c.CacheTTL, err)
		}
		opts = append(opts, nawala.WithCacheTTL(d))
	}

	if c.Concurrency != nil {
		opts = append(opts, nawala.WithConcurrency(*c.Concurrency))
	}

	if c.EDNS0Size != nil {
		opts = append(opts, nawala.WithEDNS0Size(*c.EDNS0Size))
	}

	if c.Servers != nil {
		servers := make([]nawala.DNSServer, len(c.Servers))
		for i, s := range c.Servers {
			servers[i] = nawala.DNSServer{
				Address:   s.Address,
				Keyword:   s.Keyword,
				QueryType: s.QueryType,
			}
		}
		opts = append(opts, nawala.WithServers(servers))
	}

	return opts, nil
}

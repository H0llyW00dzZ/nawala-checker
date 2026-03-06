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
// ConfigVersion records the version string declared in the file envelope
// (nawala.version). It is populated by [loadConfig] and is not serialised
// back to disk — it is a read-only metadata field used only for version
// compatibility checks.
type Config struct {
	// ConfigVersion is the "version" field from the file envelope, e.g. "0.6.5".
	// It is empty when the file omits the field.
	ConfigVersion  string
	Timeout        string  `json:"timeout"         yaml:"timeout"`
	CommandTimeout string  `json:"command_timeout" yaml:"command_timeout"`
	MaxRetries     *int    `json:"max_retries"     yaml:"max_retries"`
	CacheTTL       string  `json:"cache_ttl"       yaml:"cache_ttl"`
	DisableCache   *bool   `json:"disable_cache"   yaml:"disable_cache"`
	Concurrency    *int    `json:"concurrency"     yaml:"concurrency"`
	EDNS0Size      *uint16 `json:"edns0_size"      yaml:"edns0_size"`
	Protocol       string  `json:"protocol"        yaml:"protocol"`
	TLSServerName  string  `json:"tls_server_name" yaml:"tls_server_name"`
	TLSSkipVerify  *bool   `json:"tls_skip_verify" yaml:"tls_skip_verify"`
	// KeepAlivePoolSize enables TCP/TLS keep-alive when non-nil and > 0.
	// Only effective when Protocol is "tcp" or "tcp-tls" and the upstream
	// DNS server supports RFC 7766 / RFC 7858 persistent connections.
	KeepAlivePoolSize *int        `json:"keep_alive_pool_size" yaml:"keep_alive_pool_size"`
	Servers           []ServerDef `json:"servers"              yaml:"servers"`
}

// configFile is the top-level envelope that wraps Config in a JSON or YAML
// config file. The file format mirrors the JSON output envelope:
//
//	{"nawala":{"version":"0.6.5","configuration":{...}}}
//
// Or in YAML:
//
//	nawala:
//	  version: "0.6.5"
//	  configuration:
//	    timeout: 5s
type configFile struct {
	Nawala struct {
		Version       string `json:"version"       yaml:"version"`
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
	cfg.ConfigVersion = cf.Nawala.Version
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
		c.parseProtocol,
		c.parseTLSServerName,
		c.parseTLSSkipVerify,
		c.parseKeepAlive,
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

// parseProtocol returns a WithProtocol option if the Protocol field is set.
// Valid values are "udp", "tcp", and "tcp-tls"; others are ignored by the SDK.
func (c *Config) parseProtocol() (nawala.Option, error) {
	if c.Protocol == "" {
		return nil, nil
	}
	return nawala.WithProtocol(c.Protocol), nil
}

// parseTLSServerName returns a WithTLSServerName option if TLSServerName is set.
// Only effective when protocol is "tcp-tls".
func (c *Config) parseTLSServerName() (nawala.Option, error) {
	if c.TLSServerName == "" {
		return nil, nil
	}
	return nawala.WithTLSServerName(c.TLSServerName), nil
}

// parseTLSSkipVerify returns a WithTLSSkipVerify option if TLSSkipVerify is true.
// Only effective when protocol is "tcp-tls".
func (c *Config) parseTLSSkipVerify() (nawala.Option, error) {
	if c.TLSSkipVerify != nil && *c.TLSSkipVerify {
		return nawala.WithTLSSkipVerify(), nil
	}
	return nil, nil
}

// parseKeepAlive returns a WithKeepAlive option when KeepAlivePoolSize is set.
// A value of 0 is still passed to WithKeepAlive so the SDK can apply its own
// default (min(concurrency, 10)). Nil means keep-alive is disabled entirely.
// Only effective when protocol is "tcp" or "tcp-tls" and the DNS server
// supports RFC 7766 / RFC 7858 persistent connections.
func (c *Config) parseKeepAlive() (nawala.Option, error) {
	if c.KeepAlivePoolSize == nil {
		return nil, nil
	}
	return nawala.WithKeepAlive(*c.KeepAlivePoolSize), nil
}

// parseCommandTimeout parses the command_timeout string into a time.Duration.
// This is a CLI-level concern and is NOT converted into a nawala.Option.
// Returns the zero duration when the field is not set.
func (c *Config) parseCommandTimeout() (time.Duration, error) {
	if c.CommandTimeout == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(c.CommandTimeout)
	if err != nil {
		return 0, fmt.Errorf("invalid command_timeout %q: %w", c.CommandTimeout, err)
	}
	return d, nil
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

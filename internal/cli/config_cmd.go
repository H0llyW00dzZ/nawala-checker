// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configCmd is the "config" subcommand.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the effective configuration",
	Long:  configLong,
	Args:  cobra.NoArgs,
	RunE:  runConfig,
}

func init() {
	configCmd.Flags().StringP("output", "o", "", "write config to a file instead of stdout")
	configCmd.Flags().Bool("json", false, "output as JSON")
	configCmd.Flags().Bool("yaml", false, "output as YAML (default)")
}

// effectiveConfig is the JSON/YAML-serialisable view of the resolved
// configuration. Values are expressed as strings (durations) or primitives
// so the output is always a valid config file the user can reuse.
type effectiveConfig struct {
	Timeout       string      `json:"timeout"         yaml:"timeout"`
	MaxRetries    int         `json:"max_retries"     yaml:"max_retries"`
	CacheTTL      string      `json:"cache_ttl"       yaml:"cache_ttl"`
	DisableCache  bool        `json:"disable_cache"   yaml:"disable_cache"`
	Concurrency   int         `json:"concurrency"     yaml:"concurrency"`
	EDNS0Size     uint16      `json:"edns0_size"      yaml:"edns0_size"`
	Protocol      string      `json:"protocol"        yaml:"protocol"`
	TLSServerName string      `json:"tls_server_name" yaml:"tls_server_name"`
	TLSSkipVerify bool        `json:"tls_skip_verify" yaml:"tls_skip_verify"`
	Servers       []ServerDef `json:"servers"         yaml:"servers"`
}

// coalesce returns *ptr if ptr is non-nil, otherwise def.
// Used to apply optional Config fields over effectiveConfig defaults.
func coalesce[T any](ptr *T, def T) T {
	if ptr != nil {
		return *ptr
	}
	return def
}

// resolveEffectiveConfig builds the final configuration by starting from the
// SDK defaults and overlaying any explicitly set fields from cfg (nil = use default).
// cfg may be nil when no config file is loaded.
func resolveEffectiveConfig(cfg *Config) effectiveConfig {
	c := nawala.New()
	servers := c.Servers()
	defs := make([]ServerDef, len(servers))
	for i, s := range servers {
		defs[i] = ServerDef{Address: s.Address, Keyword: s.Keyword, QueryType: s.QueryType}
	}

	eff := effectiveConfig{
		Timeout:      (5 * time.Second).String(),
		MaxRetries:   2,
		CacheTTL:     (5 * time.Minute).String(),
		DisableCache: false,
		Concurrency:  100,
		EDNS0Size:    1232,
		Protocol:     "udp",
		Servers:      defs,
	}

	if cfg == nil {
		return eff
	}

	// Pointer fields: coalesce dereferences the pointer or keeps the default.
	eff.MaxRetries = coalesce(cfg.MaxRetries, eff.MaxRetries)
	eff.DisableCache = coalesce(cfg.DisableCache, eff.DisableCache)
	eff.Concurrency = coalesce(cfg.Concurrency, eff.Concurrency)
	eff.EDNS0Size = coalesce(cfg.EDNS0Size, eff.EDNS0Size)
	eff.TLSSkipVerify = coalesce(cfg.TLSSkipVerify, eff.TLSSkipVerify)

	// String/slice fields: empty/nil means "not set".
	if cfg.Timeout != "" {
		eff.Timeout = cfg.Timeout
	}
	if cfg.CacheTTL != "" {
		eff.CacheTTL = cfg.CacheTTL
	}
	if cfg.Protocol != "" {
		eff.Protocol = cfg.Protocol
	}
	if cfg.TLSServerName != "" {
		eff.TLSServerName = cfg.TLSServerName
	}
	if cfg.Servers != nil {
		eff.Servers = cfg.Servers
	}

	return eff
}

// runConfig resolves the effective configuration and writes it to stdout or a file.
func runConfig(cmd *cobra.Command, _ []string) error {
	outputPath, _ := cmd.Flags().GetString("output")
	jsonMode, _ := cmd.Flags().GetBool("json")

	// Resolve effective config: SDK defaults overlaid with any loaded file.
	var cfg *Config
	if configPath != "" {
		loaded, err := loadConfig(configPath)
		if err != nil {
			return err
		}
		cfg = loaded
	}
	eff := resolveEffectiveConfig(cfg)

	// Build the envelope.
	type envelope struct {
		Nawala struct {
			Configuration effectiveConfig `json:"configuration" yaml:"configuration"`
		} `json:"nawala" yaml:"nawala"`
	}
	var env envelope
	env.Nawala.Configuration = eff

	// Serialise.
	// effectiveConfig only contains string/int/bool/uint16 and []ServerDef —
	// neither json.Marshal nor yaml.Marshal can fail for these types.
	var output []byte
	if jsonMode {
		output, _ = json.MarshalIndent(env, "", "  ")
		output = append(output, '\n')
	} else {
		output, _ = yaml.Marshal(env)
	}

	// Write to file or stdout.
	if outputPath == "" {
		_, err := fmt.Fprint(cmd.OutOrStdout(), string(output))
		return err
	}

	w, err := NewWriter(outputPath, FormatText)
	if err != nil {
		return err
	}
	defer func() {
		_ = w.Close()
	}()
	_, _ = w.w.WriteString(string(output))
	return nil
}

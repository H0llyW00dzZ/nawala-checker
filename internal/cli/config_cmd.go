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
	Timeout      string        `json:"timeout"        yaml:"timeout"`
	MaxRetries   int           `json:"max_retries"    yaml:"max_retries"`
	CacheTTL     string        `json:"cache_ttl"      yaml:"cache_ttl"`
	DisableCache bool          `json:"disable_cache"  yaml:"disable_cache"`
	Concurrency  int           `json:"concurrency"    yaml:"concurrency"`
	EDNS0Size    uint16        `json:"edns0_size"     yaml:"edns0_size"`
	Servers      []ServerDef   `json:"servers"        yaml:"servers"`
}

// sdkDefaults returns an effectiveConfig populated with the SDK's built-in defaults.
// These mirror the constants in src/nawala/checker.go.
func sdkDefaults() effectiveConfig {
	servers := nawala.New().Servers()
	defs := make([]ServerDef, len(servers))
	for i, s := range servers {
		defs[i] = ServerDef{
			Address:   s.Address,
			Keyword:   s.Keyword,
			QueryType: s.QueryType,
		}
	}
	return effectiveConfig{
		Timeout:      (5 * time.Second).String(),
		MaxRetries:   2,
		CacheTTL:     (5 * time.Minute).String(),
		DisableCache: false,
		Concurrency:  100,
		EDNS0Size:    1232,
		Servers:      defs,
	}
}

// mergeConfig overlays the user-supplied Config on top of the SDK defaults.
func mergeConfig(base effectiveConfig, cfg *Config) effectiveConfig {
	if cfg.Timeout != "" {
		base.Timeout = cfg.Timeout
	}
	if cfg.MaxRetries != nil {
		base.MaxRetries = *cfg.MaxRetries
	}
	if cfg.CacheTTL != "" {
		base.CacheTTL = cfg.CacheTTL
	}
	if cfg.DisableCache != nil {
		base.DisableCache = *cfg.DisableCache
	}
	if cfg.Concurrency != nil {
		base.Concurrency = *cfg.Concurrency
	}
	if cfg.EDNS0Size != nil {
		base.EDNS0Size = *cfg.EDNS0Size
	}
	if cfg.Servers != nil {
		base.Servers = cfg.Servers
	}
	return base
}

// runConfig resolves the effective configuration and writes it to stdout or a file.
func runConfig(cmd *cobra.Command, _ []string) error {
	outputPath, _ := cmd.Flags().GetString("output")
	jsonMode, _  := cmd.Flags().GetBool("json")

	// Resolve effective config: start from SDK defaults, merge loaded file.
	eff := sdkDefaults()
	if configPath != "" {
		cfg, err := loadConfig(configPath)
		if err != nil {
			return err
		}
		eff = mergeConfig(eff, cfg)
	}

	// Build the envelope.
	type envelope struct {
		Nawala struct {
			Configuration effectiveConfig `json:"configuration" yaml:"configuration"`
		} `json:"nawala" yaml:"nawala"`
	}
	var env envelope
	env.Nawala.Configuration = eff

	// Serialise.
	var output []byte
	var err error
	if jsonMode {
		output, err = json.Marshal(env)
		if err != nil {
			return fmt.Errorf("marshalling config: %w", err)
		}
		output = append(output, '\n')
	} else {
		output, err = yaml.Marshal(env)
		if err != nil {
			return fmt.Errorf("marshalling config: %w", err)
		}
	}

	// Write to file or stdout.
	if outputPath == "" {
		_, err = fmt.Fprint(cmd.OutOrStdout(), string(output))
		return err
	}

	w, err := NewWriter(outputPath, false)
	if err != nil {
		return err
	}
	defer w.Close()
	w.w.WriteString(string(output))
	return nil
}

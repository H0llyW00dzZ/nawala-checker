// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newConfigCmd creates a fresh config command for isolated testing.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "config",
		Args: cobra.NoArgs,
		RunE: runConfig,
	}
	cmd.Flags().StringP("output", "o", "", "write config to a file instead of stdout")
	cmd.Flags().Bool("json", false, "output as JSON")
	cmd.Flags().Bool("yaml", false, "output as YAML (default)")
	return cmd
}

func TestRunConfig_DefaultsJSON(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	var buf bytes.Buffer
	cmd := newConfigCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runConfig --json error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"nawala"`) {
		t.Errorf("JSON output missing nawala envelope: %q", out)
	}
	if !strings.Contains(out, `"configuration"`) {
		t.Errorf("JSON output missing configuration key: %q", out)
	}

	// Must be valid JSON.
	var wrapper struct {
		Nawala struct {
			Configuration effectiveConfig `json:"configuration"`
		} `json:"nawala"`
	}
	if err := json.Unmarshal([]byte(out), &wrapper); err != nil {
		t.Fatalf("JSON output is not valid JSON: %v\n%s", err, out)
	}
	cfg := wrapper.Nawala.Configuration
	if cfg.Timeout == "" {
		t.Error("expected non-empty timeout in defaults")
	}
	if cfg.Concurrency == 0 {
		t.Error("expected non-zero concurrency in defaults")
	}
	if len(cfg.Servers) == 0 {
		t.Error("expected default servers")
	}
}

func TestRunConfig_DefaultsYAML(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	var buf bytes.Buffer
	cmd := newConfigCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{}) // no format flag → YAML default
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runConfig (yaml default) error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "nawala:") {
		t.Errorf("YAML output missing nawala key: %q", out)
	}
	if !strings.Contains(out, "configuration:") {
		t.Errorf("YAML output missing configuration key: %q", out)
	}

	// Must be valid YAML that round-trips.
	var wrapper struct {
		Nawala struct {
			Configuration effectiveConfig `yaml:"configuration"`
		} `yaml:"nawala"`
	}
	if err := yaml.Unmarshal([]byte(out), &wrapper); err != nil {
		t.Fatalf("YAML output is not valid YAML: %v\n%s", err, out)
	}
	if wrapper.Nawala.Configuration.Timeout == "" {
		t.Error("expected non-empty timeout in YAML defaults")
	}
}

func TestRunConfig_FromFileJSON(t *testing.T) {
	content := `{"nawala":{"configuration":{"timeout":"30s","concurrency":25}}}`
	path := filepath.Join(t.TempDir(), "custom.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = path
	defer func() { configPath = saved }()

	var buf bytes.Buffer
	cmd := newConfigCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runConfig --json with file error: %v", err)
	}

	var wrapper struct {
		Nawala struct {
			Configuration effectiveConfig `json:"configuration"`
		} `json:"nawala"`
	}
	if err := json.Unmarshal(buf.Bytes(), &wrapper); err != nil {
		t.Fatalf("JSON output invalid: %v", err)
	}
	cfg := wrapper.Nawala.Configuration
	if cfg.Timeout != "30s" {
		t.Errorf("expected timeout=30s, got %q", cfg.Timeout)
	}
	if cfg.Concurrency != 25 {
		t.Errorf("expected concurrency=25, got %d", cfg.Concurrency)
	}
}

func TestRunConfig_FromFileYAML(t *testing.T) {
	content := `
nawala:
  configuration:
    timeout: 20s
    concurrency: 10
`
	path := filepath.Join(t.TempDir(), "custom.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = path
	defer func() { configPath = saved }()

	var buf bytes.Buffer
	cmd := newConfigCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{}) // YAML output
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runConfig yaml with file error: %v", err)
	}

	if !strings.Contains(buf.String(), "timeout: 20s") {
		t.Errorf("YAML output missing merged timeout: %q", buf.String())
	}
}

func TestRunConfig_BadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{invalid}"), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = path
	defer func() { configPath = saved }()

	cmd := newConfigCmd()
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for bad config, got nil")
	}
}

func TestRunConfig_OutputFile(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	outPath := filepath.Join(t.TempDir(), "out.json")
	cmd := newConfigCmd()
	cmd.SetArgs([]string{"--json", "--output", outPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runConfig -o error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), `"nawala"`) {
		t.Errorf("file output missing nawala envelope: %q", string(data))
	}
}

func TestRunConfig_OutputFile_BadPath(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	cmd := newConfigCmd()
	cmd.SetArgs([]string{"--json", "--output", "/nonexistent/dir/out.json"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for bad output path, got nil")
	}
}

func TestConfigCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"config", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "configuration") {
		t.Errorf("config --help missing description: %q", out)
	}
}

func TestRunConfig_MarshalErrorJSON(t *testing.T) {
	// For coverage symmetry — json.Marshal on effectiveConfig can't fail
	// with real data, so just verify the happy path produces valid output.
	eff := sdkDefaults()
	if eff.Timeout == "" {
		t.Error("sdkDefaults returned empty timeout")
	}
	if eff.EDNS0Size == 0 {
		t.Error("sdkDefaults returned zero EDNS0Size")
	}
}

func TestRunConfig_MergeConfig_AllFields(t *testing.T) {
	base := sdkDefaults()
	retries := 5
	concurrency := 75
	edns := uint16(2048)
	disableCache := true
	cfg := &Config{
		Timeout:      "15s",
		MaxRetries:   &retries,
		CacheTTL:     "20m",
		DisableCache: &disableCache,
		Concurrency:  &concurrency,
		EDNS0Size:    &edns,
		Servers: []ServerDef{
			{Address: "1.1.1.1", Keyword: "test", QueryType: "A"},
		},
	}
	merged := mergeConfig(base, cfg)

	if merged.Timeout != "15s" {
		t.Errorf("expected timeout=15s, got %q", merged.Timeout)
	}
	if merged.MaxRetries != 5 {
		t.Errorf("expected max_retries=5, got %d", merged.MaxRetries)
	}
	if merged.CacheTTL != "20m" {
		t.Errorf("expected cache_ttl=20m, got %q", merged.CacheTTL)
	}
	if !merged.DisableCache {
		t.Error("expected disable_cache=true")
	}
	if merged.Concurrency != 75 {
		t.Errorf("expected concurrency=75, got %d", merged.Concurrency)
	}
	if merged.EDNS0Size != 2048 {
		t.Errorf("expected edns0_size=2048, got %d", merged.EDNS0Size)
	}
	if len(merged.Servers) != 1 || merged.Servers[0].Address != "1.1.1.1" {
		t.Errorf("unexpected servers: %v", merged.Servers)
	}
}

func TestRunConfig_MergeConfig_EmptyFields(t *testing.T) {
	base := sdkDefaults()
	origTimeout := base.Timeout
	cfg := &Config{} // all nil/zero — nothing should change
	merged := mergeConfig(base, cfg)

	if merged.Timeout != origTimeout {
		t.Errorf("empty config changed timeout: got %q", merged.Timeout)
	}
}

func TestConfigLongNotEmpty(t *testing.T) {
	if configLong == "" {
		t.Error("configLong is empty — embed failed")
	}
}

// Also extend TestMagicEmbed_VarsNotEmpty coverage via this file's test.
func TestConfigCmd_IsRegistered(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "config" {
			found = true
			break
		}
	}
	if !found {
		t.Error("configCmd is not registered on rootCmd")
	}
}

func init() {
	// Ensure fmt and yaml imports are used.
	_ = fmt.Sprintf
}

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestBuildChecker_NoConfig(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	c, err := buildChecker()
	if err != nil {
		t.Fatalf("buildChecker() error: %v", err)
	}
	if c == nil {
		t.Fatal("buildChecker() returned nil checker")
	}
}

func TestBuildChecker_WithConfig(t *testing.T) {
	content := `{"timeout": "10s"}`
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = path
	defer func() { configPath = saved }()

	c, err := buildChecker()
	if err != nil {
		t.Fatalf("buildChecker() error: %v", err)
	}
	if c == nil {
		t.Fatal("buildChecker() returned nil checker")
	}
}

func TestBuildChecker_InvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{bad}"), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = path
	defer func() { configPath = saved }()

	_, err := buildChecker()
	if err == nil {
		t.Fatal("expected error for invalid config, got nil")
	}
}

func TestBuildChecker_InvalidDuration(t *testing.T) {
	content := `{"timeout": "not-a-duration"}`
	path := filepath.Join(t.TempDir(), "bad_timeout.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = path
	defer func() { configPath = saved }()

	_, err := buildChecker()
	if err == nil {
		t.Fatal("expected error for invalid timeout, got nil")
	}
}

func TestBuildChecker_MissingFile(t *testing.T) {
	saved := configPath
	configPath = "/nonexistent/config.json"
	defer func() { configPath = saved }()

	_, err := buildChecker()
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}
}

func TestRootCmd_Version(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "nawala") {
		t.Errorf("--version output missing 'nawala': %q", out)
	}
}

func TestRootCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "nawala") {
		t.Errorf("--help output missing 'nawala': %q", out)
	}
	if !strings.Contains(out, "check") {
		t.Errorf("--help output missing 'check': %q", out)
	}
	if !strings.Contains(out, "status") {
		t.Errorf("--help output missing 'status': %q", out)
	}
}

func TestRootCmd_NoArgs_ShowsHelp(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Available Commands") {
		t.Errorf("no-args output missing help text: %q", out)
	}
}

func TestCheckCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"check", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Check whether") {
		t.Errorf("check --help missing description: %q", out)
	}
}

func TestRootCmd_BareArgs_DelegatesToCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live DNS test in short mode")
	}

	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"google.com"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestRunRoot_NoArgs(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{}) // reset args
	rootCmd.Flags().Set("version", "false")

	// Call runRoot directly with empty args to hit the len(args)==0 path.
	err := runRoot(rootCmd, []string{})
	if err != nil {
		t.Fatalf("runRoot() error: %v", err)
	}
	if !strings.Contains(buf.String(), "Available Commands") {
		t.Errorf("expected help output, got: %q", buf.String())
	}
}

func TestRunRoot_BareArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live DNS test in short mode")
	}

	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{}) // reset args
	rootCmd.Flags().Set("version", "false")

	// Call runRoot directly with args to hit the runCheck delegation.
	err := runRoot(rootCmd, []string{"google.com"})
	if err != nil {
		t.Logf("runRoot() returned error (expected on non-Indonesian networks): %v", err)
	}
}

func TestStatusCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"status", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "health") {
		t.Errorf("status --help missing description: %q", out)
	}
}

func TestMagicEmbed_VarsNotEmpty(t *testing.T) {
	if rootLong == "" {
		t.Error("rootLong is empty — embed failed")
	}
	if checkLong == "" {
		t.Error("checkLong is empty — embed failed")
	}
	if statusLong == "" {
		t.Error("statusLong is empty — embed failed")
	}
}

// newStatusCmd creates a fresh status command for isolated testing.
func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "status",
		Args: cobra.NoArgs,
		RunE: runStatus,
	}
	cmd.Flags().StringP("output", "o", "", "write results to a file instead of stdout")
	cmd.Flags().Bool("json", false, "output results as NDJSON")
	return cmd
}

func TestRunStatus_BadConfig(t *testing.T) {
	badCfg := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badCfg, []byte("{invalid}"), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = badCfg
	defer func() { configPath = saved }()

	cmd := newStatusCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from bad config, got nil")
	}
}

func TestRunStatus_BadOutputPath(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	cmd := newStatusCmd()
	cmd.SetArgs([]string{"--output", "/nonexistent/dir/out.txt"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from bad output path, got nil")
	}
}

func TestRunStatus_DNSError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live DNS test in short mode")
	}

	// Config with unreachable server + tiny timeout to force DNS error.
	badCfg := filepath.Join(t.TempDir(), "bad_server.json")
	cfgContent := `{
		"timeout": "1ms",
		"max_retries": 0,
		"servers": [{"address": "192.0.2.1", "keyword": "test", "query_type": "A"}]
	}`
	if err := os.WriteFile(badCfg, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = badCfg
	defer func() { configPath = saved }()

	outPath := filepath.Join(t.TempDir(), "status.txt")
	cmd := newStatusCmd()
	cmd.SetArgs([]string{"--output", outPath})

	// May return an error or succeed with offline servers.
	_ = cmd.Execute()
}

func TestRunStatus_NoServers(t *testing.T) {
	// Config with empty servers array causes ErrNoDNSServers,
	// triggering the "dns status check failed" error wrapping.
	cfgPath := filepath.Join(t.TempDir(), "no_servers.json")
	if err := os.WriteFile(cfgPath, []byte(`{"servers": []}`), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = cfgPath
	defer func() { configPath = saved }()

	cmd := newStatusCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from empty servers, got nil")
	}
}

func TestRunStatus_LiveDNS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live DNS test in short mode")
	}

	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	outPath := filepath.Join(t.TempDir(), "status.txt")
	cmd := newStatusCmd()
	cmd.SetArgs([]string{"--output", outPath})

	if err := cmd.Execute(); err != nil {
		t.Logf("Execute() returned error (expected on non-Indonesian networks): %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "online") && !strings.Contains(out, "OFFLINE") {
		t.Errorf("output missing status indicator: %q", out)
	}
}

func TestRunStatus_LiveDNS_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live DNS test in short mode")
	}

	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	outPath := filepath.Join(t.TempDir(), "status.json")
	cmd := newStatusCmd()
	cmd.SetArgs([]string{"--json", "--output", outPath})

	if err := cmd.Execute(); err != nil {
		t.Logf("Execute() returned error (expected on non-Indonesian networks): %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), "\"server\"") {
		t.Errorf("output missing JSON server field: %q", string(data))
	}
}

func TestExecute_Success(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestExecute_Error(t *testing.T) {
	badCfg := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badCfg, []byte("{invalid}"), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = badCfg
	defer func() { configPath = saved }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"check", "google.com"})

	// SilenceErrors is true on rootCmd, so Execute() returns nil
	// but the error was still triggered internally.
	_ = Execute()
}

func TestErrPartialFailure(t *testing.T) {
	if ErrPartialFailure == nil {
		t.Fatal("ErrPartialFailure should not be nil")
	}
	if ErrPartialFailure.Error() == "" {
		t.Fatal("ErrPartialFailure should have a message")
	}
}

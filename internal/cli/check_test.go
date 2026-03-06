// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCollectDomains_ArgsOnly(t *testing.T) {
	domains, err := collectDomains([]string{"a.com", "b.com"}, "")
	if err != nil {
		t.Fatalf("collectDomains error: %v", err)
	}
	if len(domains) != 2 {
		t.Errorf("len(domains) = %d, want 2", len(domains))
	}
}

func TestCollectDomains_Dedup(t *testing.T) {
	domains, err := collectDomains([]string{"a.com", "b.com", "a.com"}, "")
	if err != nil {
		t.Fatalf("collectDomains error: %v", err)
	}
	if len(domains) != 2 {
		t.Errorf("len(domains) = %d, want 2 (deduplication)", len(domains))
	}
}

// TestCollectDomains_CaseInsensitiveDedup verifies that mixed-case variants of
// the same domain (e.g. "Google.com" and "google.com") are treated as one entry.
// This matches the SDK's internal normalizeDomain behaviour (strings.ToLower).
func TestCollectDomains_CaseInsensitiveDedup(t *testing.T) {
	domains, err := collectDomains([]string{"Google.com", "google.com", "GOOGLE.COM"}, "")
	if err != nil {
		t.Fatalf("collectDomains error: %v", err)
	}
	if len(domains) != 1 {
		t.Errorf("len(domains) = %d, want 1 (case-insensitive dedup)", len(domains))
	}
	if domains[0] != "google.com" {
		t.Errorf("domains[0] = %q, want %q", domains[0], "google.com")
	}
}

func TestCollectDomains_EmptyArgs(t *testing.T) {
	domains, err := collectDomains([]string{}, "")
	if err != nil {
		t.Fatalf("collectDomains error: %v", err)
	}
	if len(domains) != 0 {
		t.Errorf("len(domains) = %d, want 0", len(domains))
	}
}

func TestCollectDomains_BlankAndWhitespace(t *testing.T) {
	domains, err := collectDomains([]string{"  ", "", " a.com "}, "")
	if err != nil {
		t.Fatalf("collectDomains error: %v", err)
	}
	if len(domains) != 1 {
		t.Errorf("len(domains) = %d, want 1", len(domains))
	}
	if domains[0] != "a.com" {
		t.Errorf("domains[0] = %q, want %q", domains[0], "a.com")
	}
}

func TestCollectDomains_File(t *testing.T) {
	content := `# Header comment
google.com
reddit.com

# Another comment
github.com
`
	path := filepath.Join(t.TempDir(), "domains.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	domains, err := collectDomains(nil, path)
	if err != nil {
		t.Fatalf("collectDomains error: %v", err)
	}
	if len(domains) != 3 {
		t.Errorf("len(domains) = %d, want 3", len(domains))
	}
}

func TestCollectDomains_FileAndArgs_Dedup(t *testing.T) {
	content := "google.com\nreddit.com\n"
	path := filepath.Join(t.TempDir(), "domains.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	domains, err := collectDomains([]string{"google.com", "github.com"}, path)
	if err != nil {
		t.Fatalf("collectDomains error: %v", err)
	}
	if len(domains) != 3 {
		t.Errorf("len(domains) = %d, want 3", len(domains))
	}
}

func TestCollectDomains_FileNotFound(t *testing.T) {
	_, err := collectDomains(nil, "/nonexistent/domains.txt")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestCollectDomains_FileEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	domains, err := collectDomains(nil, path)
	if err != nil {
		t.Fatalf("collectDomains error: %v", err)
	}
	if len(domains) != 0 {
		t.Errorf("len(domains) = %d, want 0", len(domains))
	}
}

func TestCollectDomains_FileCommentsOnly(t *testing.T) {
	content := "# comment 1\n# comment 2\n   \n"
	path := filepath.Join(t.TempDir(), "comments.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	domains, err := collectDomains(nil, path)
	if err != nil {
		t.Fatalf("collectDomains error: %v", err)
	}
	if len(domains) != 0 {
		t.Errorf("len(domains) = %d, want 0", len(domains))
	}
}

func TestCollectDomains_ScannerError(t *testing.T) {
	// Create a file with a line longer than bufio.MaxScanTokenSize (64KB)
	// to trigger a scanner.Err() return (bufio.ErrTooLong).
	path := filepath.Join(t.TempDir(), "long_line.txt")
	line := strings.Repeat("a", 65*1024) // 65KB single line, no newline
	if err := os.WriteFile(path, []byte(line), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := collectDomains(nil, path)
	if err == nil {
		t.Fatal("expected error from oversized line, got nil")
	}
	if !strings.Contains(err.Error(), "reading domain file") {
		t.Errorf("error should wrap 'reading domain file': %v", err)
	}
}

// TestToASCIIDomain covers all branches of the toASCIIDomain helper.
func TestToASCIIDomain(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "pure ASCII - returned unchanged (idna no-op)",
			input: "google.com",
			want:  "google.com",
		},
		{
			name:  "unicode IDN - converts to Punycode",
			input: "例え.jp",
			want:  "xn--r8jz45g.jp",
		},
		{
			name: "invalid label - ToASCII fails, fallback to original",
			// Leading-hyphen labels are rejected by idna.Lookup ("invalid label").
			// The original string is returned so the SDK can reject it downstream.
			input: "-bad.com",
			want:  "-bad.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toASCIIDomain(tt.input)
			if got != tt.want {
				t.Errorf("toASCIIDomain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestCollectDomains_IDNAConversion verifies that Unicode IDN domains are
// automatically converted to Punycode and that the Unicode and Punycode
// forms of the same domain are deduplicated to a single entry.
func TestCollectDomains_IDNAConversion(t *testing.T) {
	t.Run("unicode converts to punycode", func(t *testing.T) {
		domains, err := collectDomains([]string{"例え.jp"}, "")
		if err != nil {
			t.Fatalf("collectDomains error: %v", err)
		}
		if len(domains) != 1 {
			t.Fatalf("len(domains) = %d, want 1", len(domains))
		}
		if domains[0] != "xn--r8jz45g.jp" {
			t.Errorf("domains[0] = %q, want %q", domains[0], "xn--r8jz45g.jp")
		}
	})

	t.Run("unicode and punycode forms deduplicated", func(t *testing.T) {
		domains, err := collectDomains([]string{"例え.jp", "xn--r8jz45g.jp"}, "")
		if err != nil {
			t.Fatalf("collectDomains error: %v", err)
		}
		if len(domains) != 1 {
			t.Errorf("len(domains) = %d, want 1 (unicode+punycode should be same domain)", len(domains))
		}
	})
}

// newCheckCmd creates a fresh check command for isolated testing.
func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "check [domains...]",
		Args: cobra.ArbitraryArgs,
		RunE: runCheck,
	}
	cmd.Flags().StringP("file", "f", "", "path to a .txt file with one domain per line")
	cmd.Flags().StringP("output", "o", "", "write results to a file instead of stdout")
	cmd.Flags().StringSlice("format", []string{"text"}, "output format (text, json, html, xlsx)")
	return cmd
}

func TestRunCheck_NoDomains(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no domains provided, got nil")
	}
}

func TestRunCheck_BadConfig(t *testing.T) {
	badCfg := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badCfg, []byte("{invalid}"), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = badCfg
	defer func() { configPath = saved }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"google.com"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from bad config, got nil")
	}
}

func TestRunCheck_BadOutputPath(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"--output", "/nonexistent/dir/out.txt", "google.com"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from bad output path, got nil")
	}
}

func TestRunCheck_BadDomainFile(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"--file", "/nonexistent/domains.txt"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from missing domain file, got nil")
	}
}

func TestRunCheck_PartialFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live DNS test in short mode")
	}

	// Use a config with a non-routable DNS server and tiny timeout
	// to force result-level errors (ErrPartialFailure).
	badServerCfg := filepath.Join(t.TempDir(), "bad_server.json")
	cfgContent := `{"nawala":{"configuration":{"timeout":"1ms","max_retries":0,"servers":[{"address":"192.0.2.1","keyword":"test","query_type":"A"}]}}}`
	if err := os.WriteFile(badServerCfg, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = badServerCfg
	defer func() { configPath = saved }()

	outPath := filepath.Join(t.TempDir(), "results.txt")
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"--output", outPath, "google.com"})

	// This should return ErrPartialFailure or a check-failed error.
	// This should return ErrPartialFailure or a check-failed error.
	_ = cmd.Execute()
}

func TestRunCheck_NoServers(t *testing.T) {
	// Config with empty servers array causes ErrNoDNSServers,
	// triggering the "check failed" error wrapping path.
	cfgPath := filepath.Join(t.TempDir(), "no_servers.json")
	if err := os.WriteFile(cfgPath, []byte(`{"nawala":{"configuration":{"servers":[]}}}`), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = cfgPath
	defer func() { configPath = saved }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"google.com"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from empty servers, got nil")
	}
}

func createMockDNSServer(t *testing.T) (string, func()) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 512)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				close(done)
				return
			}
			if n >= 12 {
				// Create a valid DNS response (Standard query response, No error)
				resp := append([]byte(nil), buf[:n]...) // copy request
				resp[2] |= 0x80                         // Set QR bit (Response)
				_, _ = conn.WriteToUDP(resp, addr)
			}
		}
	}()
	return conn.LocalAddr().String(), func() {
		_ = conn.Close()
		<-done
	}
}

func TestRunCheck_Success(t *testing.T) {
	mockAddr, cleanup := createMockDNSServer(t)
	defer cleanup()

	cfgContent := fmt.Sprintf(`{"nawala":{"configuration":{"timeout":"1s","max_retries":0,"servers":[{"address":"%s","keyword":"test","query_type":"A"}]}}}`, mockAddr)

	cfgPath := filepath.Join(t.TempDir(), "success.json")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	saved := configPath
	configPath = cfgPath
	defer func() { configPath = saved }()

	outPath := filepath.Join(t.TempDir(), "success_results.txt")
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"--output", outPath, "google.com"})

	flag := cmd.Flags().Lookup("format")
	fmt.Printf("DEBUG IN TEST: flag type is %s\n", flag.Value.Type())

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error (success), got: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), "google.com") {
		t.Errorf("output missing google.com: %q", string(data))
	}
}

func TestRunCheck_MultipleFormatFlags(t *testing.T) {
	saved := configPath
	configPath = ""
	defer func() { configPath = saved }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"--format", "json", "--format", "html", "google.com"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from multiple format flags, got nil")
	}
	if !strings.Contains(err.Error(), "only one output format flag") {
		t.Errorf("unexpected error: %v", err)
	}
}

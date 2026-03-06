// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/net/idna"
)

// checkCmd is the "check" subcommand.
var checkCmd = &cobra.Command{
	Use:     "check [domains...]",
	Short:   "Check domains for DNS blocking",
	Long:    checkLong,
	Example: checkExample,
	Args:    cobra.ArbitraryArgs,
	RunE:    runCheck,
}

func init() {
	checkCmd.Flags().StringP("file", "f", "", "path to a .txt file with one domain per line")
	checkCmd.Flags().StringP("output", "o", "", "write results to a file instead of stdout")
	checkCmd.Flags().StringSlice("format", []string{"text"}, "output format (text, json, html, xlsx)")
}

// runCheck is the shared implementation for both the root default action
// and the explicit "check" subcommand.
func runCheck(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")
	outputPath, _ := cmd.Flags().GetString("output")

	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}

	// Collect domains from args and/or file.
	domains, err := collectDomains(args, filePath)
	if err != nil {
		return err
	}
	if len(domains) == 0 {
		return fmt.Errorf("no domains provided (use positional args or --file)")
	}

	// Build checker from config.
	checker, cmdTimeout, err := buildChecker(cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	// Past this point every error is a runtime error (not a flag/input
	// error), so suppress Cobra's automatic usage and error output.
	// Per-domain errors are already visible in the results; the non-zero
	// exit code is still preserved for scripts.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	// Create output writer.
	w, err := NewWriter(outputPath, format)
	if err != nil {
		return err
	}
	defer func() {
		_ = w.Close()
	}()

	// Run checks with the configured command timeout (default 30s).
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	results, err := checker.Check(ctx, domains...)
	if err != nil {
		return fmt.Errorf("check failed: %w", err)
	}

	// Write results. All checks have completed at this point;
	// text and JSON are flushed per-result, HTML and XLSX buffer
	// until Close().
	hasErrors := false
	for _, r := range results {
		w.WriteResult(r)
		if r.Error != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		return ErrPartialFailure
	}
	return nil
}

// collectDomains gathers domains from positional args and an optional file,
// deduplicates them, and returns the unique list.
func collectDomains(args []string, filePath string) ([]string, error) {
	seen := make(map[string]struct{})
	var domains []string

	addDomain := func(d string) {
		d = toASCIIDomain(strings.ToLower(strings.TrimSpace(d)))
		if d == "" {
			return
		}
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			domains = append(domains, d)
		}
	}

	// Positional args.
	for _, d := range args {
		addDomain(d)
	}

	// File — one domain per line, # comments, blank lines ignored.
	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("opening domain file: %w", err)
		}
		defer func() {
			_ = f.Close()
		}()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			addDomain(line)
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading domain file: %w", err)
		}
	}

	return domains, nil
}

// toASCIIDomain converts a Unicode domain label to its ACE/Punycode form
// using the IDNA Lookup profile (UTS #46, as used by browsers and resolvers).
// Pure-ASCII input is returned unchanged by idna.Lookup as a no-op.
// If conversion fails for any reason, the original string is returned so that
// invalid domains can be rejected downstream by the SDK.
func toASCIIDomain(d string) string {
	if d == "" {
		return d
	}
	if converted, err := idna.Lookup.ToASCII(d); err == nil {
		return converted
	}
	return d
}

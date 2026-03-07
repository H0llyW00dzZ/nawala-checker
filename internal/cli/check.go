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

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

// checkCmd is the "check" subcommand.
var checkCmd = &cobra.Command{
	Use:          "check [domains...]",
	Short:        "Check domains for DNS blocking",
	Long:         checkLong,
	Example:      checkExample,
	Args:         cobra.ArbitraryArgs,
	RunE:         runCheck,
	SilenceUsage: true,
}

func init() {
	checkCmd.Flags().StringP("file", "f", "", "path to a .txt file with one domain per line")
	checkCmd.Flags().StringP("output", "o", "", "write results to a file instead of stdout")
	checkCmd.Flags().StringSlice("format", []string{"text"}, "output format (text, json, html, xlsx)")
}

// runCheck is the shared implementation for both the root default action
// and the explicit "check" subcommand.
func runCheck(cmd *cobra.Command, args []string) error {
	// Suppress Cobra's automatic usage output for any error returned from RunE.
	// Usage is only helpful for errors caught before RunE (e.g. unknown flags).
	cmd.SilenceUsage = true

	filePath, _ := cmd.Flags().GetString("file")
	outputPath, _ := cmd.Flags().GetString("output")

	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}

	// Build checker from config.
	checker, cmdTimeout, err := buildChecker(cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	defer func() { _ = checker.Close() }()

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

	in := make(chan string)
	out := make(chan nawala.Result, checker.Concurrency())

	// Start streaming domains
	streamErrCh := make(chan error, 1)
	go func() {
		streamErrCh <- streamDomains(ctx, args, filePath, in)
	}()

	// Start checking
	checkErrCh := make(chan error, 1)
	go func() {
		checkErrCh <- checker.CheckStream(ctx, nawala.Stream{In: in, Out: out})
		close(out)
	}()

	// Write results as they arrive
	hasErrors := false
	for r := range out {
		w.WriteResult(r)
		if r.Error != nil {
			hasErrors = true
		}
	}

	// Past this point every error is a runtime error.
	// We print runtime errors to stderr ourselves so they are always visible.
	// The non-zero exit code is still preserved for scripts.
	cmd.SilenceErrors = true

	// Check for streaming errors (e.g., file not found, no domains)
	if err := <-streamErrCh; err != nil {
		w.Cancel()
		fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
		return err
	}

	// Check for checker errors
	if err := <-checkErrCh; err != nil {
		w.Cancel()
		err = fmt.Errorf("check failed: %w", err)
		fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
		return err
	}

	if hasErrors {
		return ErrPartialFailure
	}
	return nil
}

// streamDomains gathers domains from positional args and an optional file,
// deduplicates them, and streams the unique list to the 'in' channel.
func streamDomains(ctx context.Context, args []string, filePath string, in chan<- string) error {
	defer close(in)
	seen := make(map[string]struct{})
	var count int

	addDomain := func(d string) {
		d = toASCIIDomain(strings.ToLower(strings.TrimSpace(d)))
		if d == "" {
			return
		}
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			select {
			case <-ctx.Done():
			case in <- d:
				count++
			}
		}
	}

	// Positional args.
	for _, d := range args {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		addDomain(d)
	}

	// File — one domain per line, # comments, blank lines ignored.
	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("opening domain file: %w", err)
		}
		defer func() {
			_ = f.Close()
		}()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			addDomain(line)
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading domain file: %w", err)
		}
	}

	if count == 0 {
		return fmt.Errorf("no domains provided (use positional args or --file)")
	}

	return nil
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

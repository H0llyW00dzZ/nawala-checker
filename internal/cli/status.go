// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// statusCmd is the "status" subcommand.
var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Show DNS server health status",
	Long:         statusLong,
	Example:      statusExample,
	Args:         cobra.NoArgs,
	RunE:         runStatus,
	SilenceUsage: true,
}

func init() {
	statusCmd.Flags().StringP("output", "o", "", "write results to a file instead of stdout")
	statusCmd.Flags().StringSlice("format", []string{"text"}, "output format (text, json, html, xlsx)")
}

// runStatus queries all configured DNS servers for health and latency,
// then writes the results to the configured output.
func runStatus(cmd *cobra.Command, _ []string) error {
	// Suppress Cobra's automatic usage output for any error returned from RunE.
	// Usage is only helpful for errors caught before RunE (e.g. unknown flags).
	cmd.SilenceUsage = true

	outputPath, _ := cmd.Flags().GetString("output")

	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}

	checker, cmdTimeout, err := buildChecker(cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	defer func() { _ = checker.Close() }()

	w, err := NewWriter(outputPath, format)
	if err != nil {
		return err
	}
	defer func() {
		_ = w.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	statuses, err := checker.DNSStatus(ctx)

	// Past this point every error is a runtime error.
	// We print runtime errors to stderr ourselves so they are always visible.
	// The non-zero exit code is still preserved for scripts.
	cmd.SilenceErrors = true

	if err != nil {
		err = fmt.Errorf("dns status check failed: %w", err)
		fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
		return err
	}
	for _, s := range statuses {
		w.WriteStatus(s)
	}

	return nil
}

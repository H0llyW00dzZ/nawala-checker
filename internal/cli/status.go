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
	Use:   "status",
	Short: "Show DNS server health status",
	Long:  statusLong,
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().StringP("output", "o", "", "write results to a file instead of stdout")
	statusCmd.Flags().StringSlice("format", []string{"text"}, "output format (text, json, html, xlsx)")
}

// runStatus queries all configured DNS servers for health and latency,
// then writes the results to the configured output.
func runStatus(cmd *cobra.Command, _ []string) error {
	outputPath, _ := cmd.Flags().GetString("output")

	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}

	checker, cmdTimeout, err := buildChecker()
	if err != nil {
		return err
	}

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
	if err != nil {
		return fmt.Errorf("dns status check failed: %w", err)
	}
	for _, s := range statuses {
		w.WriteStatus(s)
	}

	return nil
}

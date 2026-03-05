// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"context"
	"fmt"
	"time"

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
	statusCmd.Flags().Bool("json", false, "output results as JSON")
	statusCmd.Flags().Bool("html", false, "output results as an HTML report")
	statusCmd.Flags().Bool("xlsx", false, "output results as an Excel spreadsheet")
}

// runStatus queries all configured DNS servers for health and latency,
// then writes the results to the configured output.
func runStatus(cmd *cobra.Command, _ []string) error {
	outputPath, _ := cmd.Flags().GetString("output")

	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}

	checker, err := buildChecker()
	if err != nil {
		return err
	}

	w, err := NewWriter(outputPath, format)
	if err != nil {
		return err
	}
	defer w.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

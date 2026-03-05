// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"fmt"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
	"github.com/spf13/cobra"
)

// configPath holds the --config flag value.
var configPath string

// rootCmd is the base command for the nawala CLI.
var rootCmd = &cobra.Command{
	Use:   "nawala [domains...]",
	Short: "Check domains against Indonesian ISP DNS filters",
	Long:  rootLong,
	// When bare args are provided (no subcommand), delegate to check.
	Args: cobra.ArbitraryArgs,
	RunE: runRoot,
}

// runRoot is the root command handler. It prints the version when
// --version is passed, shows help when no args are given, and
// delegates to runCheck for bare domain arguments.
func runRoot(cmd *cobra.Command, args []string) error {
	// If --version was requested, print and exit.
	v, _ := cmd.Flags().GetBool("version")
	if v {
		fmt.Fprintf(cmd.OutOrStdout(), "nawala %s\n", nawala.Version)
		return nil
	}

	// No args and no subcommand — show help.
	if len(args) == 0 {
		return cmd.Help()
	}

	// Delegate to check with the same args.
	return runCheck(cmd, args)
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to JSON or YAML config file")
	rootCmd.Flags().BoolP("version", "v", false, "print version and exit")

	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
}

// Execute runs the root command and returns any error.
func Execute() error { return rootCmd.Execute() }

// buildChecker creates a nawala.Checker from the config file (if provided).
func buildChecker() (*nawala.Checker, error) {
	if configPath == "" {
		return nawala.New(), nil
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	opts, err := cfg.toOptions()
	if err != nil {
		return nil, err
	}

	return nawala.New(opts...), nil
}

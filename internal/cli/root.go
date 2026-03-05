// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"fmt"
	"time"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
	"github.com/spf13/cobra"
)

// defaultCommandTimeout is the overall timeout for CLI commands
// (check, status) when no command_timeout is specified in the config.
const defaultCommandTimeout = 30 * time.Second

// configPath holds the --config flag value.
var configPath string

// rootCmd is the base command for the nawala CLI.
var rootCmd = &cobra.Command{
	Use:     "nawala [domains...]",
	Short:   "Check domains against Indonesian ISP DNS filters",
	Long:    rootLong,
	Example: rootExample,
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
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "nawala %s\n", nawala.Version)
		return nil
	}

	// No args and no subcommand — show help.
	if len(args) == 0 {
		return cmd.Help()
	}

	// Delegate to check with the same args.
	return runCheck(checkCmd, args)
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

// buildChecker creates a nawala.Checker from the config file (if provided)
// and resolves the command timeout. The returned duration is the overall
// timeout for the CLI command; it defaults to defaultCommandTimeout.
func buildChecker() (*nawala.Checker, time.Duration, error) {
	if configPath == "" {
		return nawala.New(), defaultCommandTimeout, nil
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, 0, err
	}

	opts, err := cfg.toOptions()
	if err != nil {
		return nil, 0, err
	}

	cmdTimeout, err := cfg.parseCommandTimeout()
	if err != nil {
		return nil, 0, err
	}
	if cmdTimeout == 0 {
		cmdTimeout = defaultCommandTimeout
	}

	return nawala.New(opts...), cmdTimeout, nil
}

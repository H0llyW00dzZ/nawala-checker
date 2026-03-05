// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// resolveFormat inspects the string format flag (--format)
// on cmd and returns the corresponding format constant.
func resolveFormat(cmd *cobra.Command) (string, error) {
	formats, err := cmd.Flags().GetStringSlice("format")
	if err != nil {
		return "", err
	}

	if len(formats) > 1 {
		return "", fmt.Errorf("only one output format flag may be used")
	}

	var format string
	if len(formats) == 1 {
		format = formats[0]
	} else {
		format = FormatText // default if empty
	}

	// Check for comma-separated multiple formats, e.g. --format=json,html
	if strings.Contains(format, ",") {
		return "", fmt.Errorf("only one output format flag may be used")
	}

	switch format {
	case FormatText, FormatJSON, FormatHTML, FormatXLSX:
		return format, nil
	default:
		return "", fmt.Errorf("invalid output format %q (expected text, json, html, or xlsx)", format)
	}
}

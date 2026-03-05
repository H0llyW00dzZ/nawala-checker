// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// resolveFormat inspects the boolean format flags (--json, --html, --xlsx)
// on cmd and returns the corresponding format constant. If no flag is set,
// it defaults to FormatText. If more than one flag is set, it returns an error.
func resolveFormat(cmd *cobra.Command) (string, error) {
	jsonMode, _ := cmd.Flags().GetBool("json")
	htmlMode, _ := cmd.Flags().GetBool("html")
	xlsxMode, _ := cmd.Flags().GetBool("xlsx")

	count := 0
	if jsonMode {
		count++
	}
	if htmlMode {
		count++
	}
	if xlsxMode {
		count++
	}

	if count > 1 {
		return "", fmt.Errorf("only one output format flag may be used (--json, --html, --xlsx)")
	}

	switch {
	case jsonMode:
		return FormatJSON, nil
	case htmlMode:
		return FormatHTML, nil
	case xlsxMode:
		return FormatXLSX, nil
	default:
		return FormatText, nil
	}
}

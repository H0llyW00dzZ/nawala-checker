// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	_ "embed" // required for //go:embed directives below
	"strings"
)

// Embedded usage text for Cobra command Long descriptions and Examples.
// The text files are compiled into the binary and never read from disk.

//go:embed usage/root_long.txt
var rootLong string

//go:embed usage/root_example.txt
var rootExample string

//go:embed usage/check_long.txt
var checkLong string

//go:embed usage/check_example.txt
var checkExample string

//go:embed usage/status_long.txt
var statusLong string

//go:embed usage/status_example.txt
var statusExample string

//go:embed usage/config_long.txt
var configLong string

func init() {
	// Trim trailing newlines from embedded example strings so Cobra's
	// help output doesn't produce double blank lines after Examples.
	// We must set the fields directly on the command structs because
	// package-level var initialisation copies the string value before
	// any init() runs. The cutset includes \r for Windows CRLF endings.
	rootCmd.Example = strings.TrimRight(rootExample, "\r\n")
	checkCmd.Example = strings.TrimRight(checkExample, "\r\n")
	statusCmd.Example = strings.TrimRight(statusExample, "\r\n")
}

// Embedded HTML templates for report output.
// Like the usage text above, they are compiled into the binary and never read from disk.

//go:embed templates/result.html
var resultHTMLTemplate string

//go:embed templates/status.html
var statusHTMLTemplate string

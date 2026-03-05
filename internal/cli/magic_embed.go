// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import _ "embed" // required for //go:embed directives below

// Embedded usage text for Cobra command Long descriptions.
// The text files are compiled into the binary and never read from disk.

//go:embed usage/root_long.txt
var rootLong string

//go:embed usage/check_long.txt
var checkLong string

//go:embed usage/status_long.txt
var statusLong string

//go:embed usage/config_long.txt
var configLong string

// Embedded HTML templates for report output.

//go:embed templates/result.html
var resultHTMLTemplate string

//go:embed templates/status.html
var statusHTMLTemplate string

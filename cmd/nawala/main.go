// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package main

import (
	"os"

	"github.com/H0llyW00dzZ/nawala-checker/internal/cli"
)

// this best parctice to make main function small and simple
// and keep all the logic in the internal/cli package
func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

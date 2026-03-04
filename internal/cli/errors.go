// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import "errors"

// ErrPartialFailure is returned by runCheck when one or more domain
// checks encountered errors. The caller should exit with code 1.
var ErrPartialFailure = errors.New("nawala: one or more checks failed")

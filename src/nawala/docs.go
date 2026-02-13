// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package nawala provides a DNS-based domain blocking checker for
// Indonesian ISP DNS filters (Nawala/Kominfo (now Komdigi)).
//
// It works by querying configurable DNS servers and scanning the
// responses for blocking keywords. The checker comes pre-configured
// with known Nawala DNS servers and their associated keywords.
//
// # Quick Start
//
//	c := nawala.New()
//	results, err := c.Check(ctx, "example.com", "another.com")
//	for _, r := range results {
//	    fmt.Printf("%s: blocked=%v\n", r.Domain, r.Blocked)
//	}
//
// # Configuration
//
// Use functional options to customize the checker:
//
//	c := nawala.New(
//	    nawala.WithTimeout(10 * time.Second),
//	    nawala.WithMaxRetries(3),
//	    nawala.WithCacheTTL(10 * time.Minute),
//	)
package nawala

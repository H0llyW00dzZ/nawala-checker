// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import "strings"

// IsValidDomain reports whether domain is a syntactically valid domain name.
//
// A valid domain must have at least two labels separated by dots,
// each label must be 2-63 characters long, contain only ASCII
// letters, digits, or hyphens, and must not start or end with a hyphen.
// The TLD (last label) must be at least 2 characters and contain only letters.
func IsValidDomain(domain string) bool {
	if domain == "" {
		return false
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}

	for i, label := range labels {
		if len(label) < 2 || len(label) > 63 {
			return false
		}

		// Labels must not start or end with a hyphen.
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}

		isTLD := i == len(labels)-1

		for _, c := range label {
			switch {
			case c >= 'a' && c <= 'z':
				// ok
			case c >= 'A' && c <= 'Z':
				// ok
			case c >= '0' && c <= '9':
				if isTLD {
					return false // TLD must be letters only.
				}
			case c == '-':
				if isTLD {
					return false // TLD must be letters only.
				}
			default:
				return false
			}
		}
	}

	return true
}

// normalizeDomain lowercases and trims whitespace from a domain name.
func normalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

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
	// Remove optional trailing dot for FQDN validation
	domain = strings.TrimSuffix(domain, ".")

	if domain == "" || len(domain) > 255 {
		return false
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}

	for i, label := range labels {
		// Labels must be 1-63 characters (RFC 1035)
		if len(label) == 0 || len(label) > 63 {
			return false
		}

		// Labels must not start or end with a hyphen
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}

		isTLD := i == len(labels)-1

		// Special handling for TLDs
		if isTLD {
			// TLD must be at least 2 characters
			if len(label) < 2 {
				return false
			}

			// Check for Punycode TLD (starts with xn--)
			if strings.HasPrefix(strings.ToLower(label), "xn--") {
				// Punycode TLDs can contain digits and hyphens (standard hostname rules apply)
				// We fall through to the general check below
			} else {
				// Standard TLDs must be letters only
				for _, c := range label {
					if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
						return false
					}
				}
				continue // Skip general check since we already validated strict alpha
			}
		}

		for _, c := range label {
			switch {
			case c >= 'a' && c <= 'z':
			case c >= 'A' && c <= 'Z':
			case c >= '0' && c <= '9':
			case c == '-':
			default:
				return false // Invalid character
			}
		}
	}

	return true
}

// normalizeDomain lowercases and trims whitespace from a domain name.
func normalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

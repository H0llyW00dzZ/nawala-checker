// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import "strings"

// IsValidDomain reports whether domain is a syntactically valid domain name.
//
// A valid domain must have at least two labels separated by dots,
// each label must be 1-63 characters long, contain only ASCII
// letters, digits, hyphens, or underscores, and must not start or
// end with a hyphen.
//
// Underscores are technically non-standard per [RFC 1035] for hostname
// labels but are accepted here to accommodate real-world DNS names such
// as Google AMP cache domains and cloud-provider service records.
//
// The TLD (last label) must be at least 2 characters. Standard TLDs must
// contain only letters, while Punycode TLDs (starting with "xn--") allow
// digits and hyphens (conforming to standard hostname rules).
//
// [RFC 1035]: https://datatracker.ietf.org/doc/html/rfc1035
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
		if !isValidLabel(label) {
			return false
		}

		if i == len(labels)-1 && !isValidTLD(label) {
			return false
		}
	}

	return true
}

// isValidLabel checks if a label is valid.
//
// Labels follow [RFC 1035] hostname rules with the addition of underscores,
// which are technically non-standard but widely used in practice
// (e.g., Google AMP cache domains, cloud-provider service endpoints).
//
// [RFC 1035]: https://datatracker.ietf.org/doc/html/rfc1035
func isValidLabel(label string) bool {
	// Labels must be 1-63 characters (RFC 1035)
	if len(label) == 0 || len(label) > 63 {
		return false
	}

	// Labels must not start or end with a hyphen
	if label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}

	for _, c := range label {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '-':
		case c == '_':
		default:
			return false // Invalid character
		}
	}
	return true
}

// isValidTLD checks if the Top-Level Domain label is valid.
// It handles both standard alphabetic TLDs and Punycode (IDN) TLDs.
func isValidTLD(label string) bool {
	// TLD must be at least 2 characters
	if len(label) < 2 {
		return false
	}

	// Check for Punycode TLD (starts with xn--)
	if len(label) > 4 && strings.EqualFold(label[:4], "xn--") {
		// Punycode TLDs follow standard hostname rules (already validated by isValidLabel).
		return true
	}

	// Standard TLDs must be letters only
	for _, c := range label {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}

	return true
}

// normalizeDomain lowercases and trims whitespace from a domain name.
func normalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

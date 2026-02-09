// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"regexp"
	"strings"
)

// domainRegex validates domain names.
var domainRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9](?:\.[a-zA-Z]{2,})+$`)

// IsValidDomain reports whether domain is a syntactically valid domain name.
func IsValidDomain(domain string) bool {
	return domainRegex.MatchString(domain)
}

// normalizeDomain lowercases and trims whitespace from a domain name.
func normalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

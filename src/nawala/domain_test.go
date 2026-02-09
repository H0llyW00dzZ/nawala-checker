// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"testing"
)

func TestIsValidDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   bool
	}{
		// Valid domains
		{"standard domain", "example.com", true},
		{"subdomain", "www.example.com", true},
		{"single char label", "x.com", true},
		{"single char subdomain", "a.b.co", true},
		{"punycode domain", "xn--p1ai.ru", true},
		{"punycode TLD", "example.xn--p1ai", true},
		{"numeric label", "123.com", true},
		{"hyphenated label", "ex-ample.com", true},
		{"case insensitive", "EXAMPLE.COM", true},
		{"FQDN with trailing dot", "example.com.", true},

		// Invalid domains
		{"empty string", "", false},
		{"no TLD", "localhost", false},
		{"start with hyphen", "-example.com", false},
		{"end with hyphen", "example-.com", false},
		{"consecutive dots", "example..com", false},
		{"start with dot", ".example.com", false},
		{"spaces", "example .com", false},
		{"invalid char", "exa_mple.com", false}, // underscores not allowed in hostnames usually, though technically valid in DNS, we stick to hostname rules here as per original intent
		{"too long label", "thislabeliswaytoolongandshoulddefinitelyfailbecausethelimitissixtythreecharacters.com", false},
		{"TLD numeric", "example.123", false},
		{"mixed case punycode TLD", "example.Xn--P1ai", true},
		{"unicode characters", "example.рф", false}, // we only support ASCII/Punycode
		{"underscore in label", "exa_mple.com", false},
		{"underscore in TLD", "example.c_m", false},
		{"punycode prefix only", "example.xn--", false},
		{"punycode prefix only case insensitive", "example.XN--", false},
		{"trailing dot with space", "example.com. ", false},
		{"double trailing dot", "example.com..", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidDomain(tt.domain); got != tt.want {
				t.Errorf("IsValidDomain(%q) = %v, want %v", tt.domain, got, tt.want)
			}
		})
	}
}

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Example.Com", "example.com"},
		{"  example.com  ", "example.com"},
		{"EXAMPLE.COM", "example.com"},
	}

	for _, tt := range tests {
		if got := normalizeDomain(tt.input); got != tt.want {
			t.Errorf("normalizeDomain(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package nawala

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/idna"
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
		{"invalid char", "exa!mple.com", false}, // special chars like ! are not allowed
		{"too long label", "thislabeliswaytoolongandshoulddefinitelyfailbecausethelimitissixtythreecharacters.com", false},
		{"TLD numeric", "example.123", false},
		{"mixed case punycode TLD", "example.Xn--P1ai", true},
		{"unicode characters", "example.рф", false},   // we only support ASCII/Punycode
		{"underscore in label", "exa_mple.com", true}, // underscores allowed for real-world domains (e.g. Google AMP cache)
		{"underscore in TLD", "example.c_m", false},
		{"punycode prefix only", "example.xn--", false},
		{"punycode prefix only case insensitive", "example.XN--", false},
		{"trailing dot with space", "example.com. ", false},
		{"double trailing dot", "example.com..", false},
		{"punycode TLD with underscore", "example.xn--p1ai_", false},

		// IDN / Punycode — Indonesian script
		// contoh.id is plain ASCII — .id is the Indonesian ccTLD
		{"IDN: plain ASCII with .id ccTLD", "contoh.id", true},
		// Indonesian Bahasa labels are encoded as Punycode before use.
		// xn--mlh5bm9hra.id represents a hypothetical Punycode-encoded Indonesian SLD
		// under the Indonesian ccTLD (.id).
		{"IDN: punycode SLD with .id ccTLD", "xn--mlh5bm9hra.id", true},
		// xn--contoh-p18d.id — a Punycode SLD containing hyphens and alphanumerics
		{"IDN: punycode SLD hyphenated with .id ccTLD", "xn--contoh-p18d.id", true},
		// Raw Unicode Indonesian/Malay — must be rejected (not Punycode)
		{"IDN: raw Unicode Indonesian (non-ASCII)", "contoh.id\xc2\xa0", false}, // NBSP — non-ASCII

		// IDN / Punycode — Thai script (xn--o3cw4h = ไทย)
		// ทดสอบ.ไทย encodes to xn--12c1fe0br.xn--o3cw4h
		{"IDN: Thai punycode SLD + Thai ccTLD", "xn--12c1fe0br.xn--o3cw4h", true},
		// Thai SLD under a standard ASCII TLD
		{"IDN: Thai punycode SLD + .th TLD", "xn--12c1fe0br.th", true},
		// Mixed-case Punycode — must lower-normalize correctly
		{"IDN: Thai punycode uppercase mixed case", "XN--12C1FE0BR.XN--O3CW4H", true},
		// Raw Thai Unicode — must be rejected
		{"IDN: raw Thai Unicode (non-ASCII)", "ทดสอบ.ไทย", false},

		// IDN / Punycode — Arabic script
		// مثال.مصر encodes to xn--mgbh0fb.xn--wgbh1c (example.egypt)
		{"IDN: Arabic punycode SLD + Egyptian ccTLD", "xn--mgbh0fb.xn--wgbh1c", true},
		// موقع.امارات encodes to xn--4gbrim.xn--mgbaam7a8h (site.uae)
		{"IDN: Arabic punycode SLD + UAE ccTLD", "xn--4gbrim.xn--mgbaam7a8h", true},
		// Arabic SLD under a standard ASCII TLD (.com)
		{"IDN: Arabic punycode SLD + .com TLD", "xn--mgbh0fb.com", true},
		// Subdomain + Arabic SLD + Arabic TLD (3 labels)
		{"IDN: subdomain + Arabic punycode domain", "www.xn--mgbh0fb.xn--wgbh1c", true},
		// Raw Arabic Unicode — must be rejected
		{"IDN: raw Arabic Unicode (non-ASCII)", "مثال.مصر", false},
		// Raw Arabic with Arabic TLD — both non-ASCII, must be rejected
		{"IDN: raw Arabic SLD and TLD (non-ASCII)", "كوم.مثال", false},
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

		// IDN / Punycode normalization — uppercase Punycode labels are lowercased.
		// normalizeDomain does NOT convert Unicode → Punycode; callers must
		// supply already-encoded Punycode labels.
		{"XN--12C1FE0BR.XN--O3CW4H", "xn--12c1fe0br.xn--o3cw4h"},   // Thai: ทดสอบ.ไทย
		{"  xn--wgbl6a.xn--p1ai  ", "xn--wgbl6a.xn--p1ai"},         // Arabic SLD + Cyrillic TLD
		{"XN--MGBH0FB.XN--WGBH1C", "xn--mgbh0fb.xn--wgbh1c"},       // Arabic: مثال.مصر
		{"XN--4GBRIM.XN--MGBAAM7A8H", "xn--4gbrim.xn--mgbaam7a8h"}, // Arabic: موقع.امارات
		{"  XN--MLH5BM9HRA.ID  ", "xn--mlh5bm9hra.id"},             // Indonesian Punycode SLD
	}

	for _, tt := range tests {
		if got := normalizeDomain(tt.input); got != tt.want {
			t.Errorf("normalizeDomain(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsValidTLD(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  bool
	}{
		// Too short
		{"empty", "", false},
		{"single char", "c", false},

		// Punycode TLDs (xn-- prefix, length > 4)
		{"punycode TLD xn--p1ai", "xn--p1ai", true},
		{"punycode TLD xn--80akhbyknj4f", "xn--80akhbyknj4f", true},
		{"punycode mixed case XN--", "XN--p1ai", true}, // EqualFold match

		// Punycode prefix only (xn-- exactly, length == 4) — not a valid punycode TLD
		{"punycode prefix only xn--", "xn--", false}, // len == 4, falls through to letter check → fails on '-'

		// Standard letters-only TLDs
		{"com", "com", true},
		{"io", "io", true},
		{"org", "org", true},
		{"id", "id", true},
		{"top", "top", true},
		{"museum", "museum", true},
		{"COM uppercase", "COM", true},
		{"Co mixed case", "Co", true},

		// Invalid: digits in TLD
		{"digit in TLD", "c0m", false},
		{"all digits", "123", false},

		// Invalid: underscore in TLD
		{"underscore in TLD", "c_m", false},
		{"punycode TLD with underscore", "xn--p1ai_", false},

		// Invalid: hyphen in TLD
		{"hyphen in TLD", "co-m", false},
		{"starts with hyphen", "-om", false},

		// Invalid: space
		{"space in TLD", "co m", false},

		// IDN / Punycode ccTLDs — Thai
		// xn--o3cw4h = ไทย (Thailand)
		{"IDN: Thai ccTLD xn--o3cw4h", "xn--o3cw4h", true},
		// Mixed-case Thai ccTLD
		{"IDN: Thai ccTLD uppercase XN--O3CW4H", "XN--O3CW4H", true},

		// IDN / Punycode ccTLDs — Arabic
		// xn--wgbh1c = مصر (Egypt)
		{"IDN: Egyptian ccTLD xn--wgbh1c", "xn--wgbh1c", true},
		// xn--mgbaam7a8h = امارات (UAE)
		{"IDN: UAE ccTLD xn--mgbaam7a8h", "xn--mgbaam7a8h", true},
		// xn--mgberp4a5d4ar = السعودية (Saudi Arabia)
		{"IDN: Saudi ccTLD xn--mgberp4a5d4ar", "xn--mgberp4a5d4ar", true},
		// xn--p1acf = рус (Russian generic)
		{"IDN: Russian generic IDN TLD xn--p1acf", "xn--p1acf", true},
		// Mixed-case Arabic ccTLD
		{"IDN: Egyptian ccTLD uppercase XN--WGBH1C", "XN--WGBH1C", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidTLD(tt.label); got != tt.want {
				t.Errorf("isValidTLD(%q) = %v, want %v", tt.label, got, tt.want)
			}
		})
	}
}

// TestIsValidDomainWithIDNA verifies the recommended consumer workflow for
// Internationalized Domain Names: Unicode → [idna.Lookup.ToASCII] → [IsValidDomain].
//
// Consumers of this SDK must convert Unicode domain names to Punycode
// (ASCII-Compatible Encoding) before passing them to [IsValidDomain],
// [Checker.Check], or [Checker.CheckOne]. This test exercises that contract by:
//
//  1. Converting raw Unicode domains to Punycode via [idna.Lookup.ToASCII].
//  2. Asserting that the Punycode output matches the expected wire-format string.
//  3. Asserting that [IsValidDomain] accepts the converted Punycode domain.
//  4. Asserting that [IsValidDomain] rejects the raw Unicode input (pre-conversion).
//
// The test covers multiple scripts: Indonesian, Thai, Arabic, Cyrillic, Chinese,
// and multi-label (subdomain + IDN) domains.
//
// [idna.Lookup.ToASCII]: https://pkg.go.dev/golang.org/x/net/idna#Profile.ToASCII
func TestIsValidDomainWithIDNA(t *testing.T) {
	// roundTrip contains Unicode domains that, after IDNA conversion,
	// must produce valid Punycode accepted by IsValidDomain.
	roundTrip := []struct {
		name     string
		unicode  string // raw Unicode input
		expected string // expected Punycode output from idna.Lookup.ToASCII
	}{
		// Indonesian — plain ASCII domains are a no-op through IDNA.
		{"Indonesian plain ASCII .id", "contoh.id", "contoh.id"},
		{"Indonesian plain ASCII tes.id", "tes.id", "tes.id"},

		// Thai — ทดสอบ.ไทย → xn--l3cfk7dp.xn--o3cw4h
		// Note: idna.Lookup applies UTS #46 mapping which may produce different
		// Punycode than raw IDNA2003 encoding (e.g., xn--l3cfk7dp vs xn--12c1fe0br).
		{"Thai SLD + Thai ccTLD", "ทดสอบ.ไทย", "xn--l3cfk7dp.xn--o3cw4h"},
		// Thai SLD under standard ASCII TLD
		{"Thai SLD + .th", "ทดสอบ.th", "xn--l3cfk7dp.th"},

		// Arabic — مثال.مصر → xn--mgbh0fb.xn--wgbh1c
		{"Arabic SLD + Egyptian ccTLD", "مثال.مصر", "xn--mgbh0fb.xn--wgbh1c"},
		// Arabic — موقع.امارات → xn--4gbrim.xn--mgbaam7a8h
		{"Arabic SLD + UAE ccTLD", "موقع.امارات", "xn--4gbrim.xn--mgbaam7a8h"},
		// Arabic SLD under .com
		{"Arabic SLD + .com", "مثال.com", "xn--mgbh0fb.com"},

		// Cyrillic — пример.рф → xn--e1afmkfd.xn--p1ai
		{"Cyrillic SLD + Russian ccTLD", "пример.рф", "xn--e1afmkfd.xn--p1ai"},

		// Chinese — 例え.中国
		{"Chinese/Japanese SLD + Chinese ccTLD", "例え.中国", "xn--r8jz45g.xn--fiqs8s"},

		// Mixed: ASCII subdomain + Unicode SLD + Unicode TLD
		{"subdomain + Arabic IDN", "www.مثال.مصر", "www.xn--mgbh0fb.xn--wgbh1c"},
		// Uppercase ASCII subdomain + Unicode (tests case normalization)
		{"uppercase subdomain + Thai IDN", "WWW.ทดสอบ.ไทย", "www.xn--l3cfk7dp.xn--o3cw4h"},

		// German (UTS #46 mapping: ü → u + combining → Punycode)
		{"German umlaut", "bücher.de", "xn--bcher-kva.de"},
	}

	for _, tt := range roundTrip {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Convert Unicode → Punycode via idna.Lookup.ToASCII.
			ascii, err := idna.Lookup.ToASCII(tt.unicode)
			require.NoError(t, err, "idna.Lookup.ToASCII(%q) returned error", tt.unicode)
			assert.Equal(t, tt.expected, ascii,
				"idna.Lookup.ToASCII(%q) produced unexpected Punycode", tt.unicode)

			// 2. The Punycode output must pass IsValidDomain.
			assert.True(t, IsValidDomain(ascii),
				"IsValidDomain(%q) should return true after IDNA conversion", ascii)

			// 3. The raw Unicode input must be rejected by IsValidDomain.
			// (Unless it's pure ASCII, in which case it's valid as-is.)
			if tt.unicode != tt.expected {
				assert.False(t, IsValidDomain(tt.unicode),
					"IsValidDomain(%q) should return false for raw Unicode", tt.unicode)
			}
		})
	}

	// defenseInDepth contains inputs where idna.Lookup.ToASCII succeeds
	// (returns a result without error) but the output is still structurally
	// invalid for DNS use. IsValidDomain provides a second layer of defense
	// by rejecting these domains.
	defenseInDepth := []struct {
		name    string
		unicode string
	}{
		// Single-label Unicode (no TLD) — IDNA converts it, but
		// IsValidDomain rejects single-label domains (no dot separator).
		{"single label Unicode", "ทดสอบ"},
		// Empty string — IDNA returns "" without error,
		// but IsValidDomain rejects empty input.
		{"empty string", ""},
	}

	for _, tt := range defenseInDepth {
		t.Run("defense in depth/"+tt.name, func(t *testing.T) {
			ascii, _ := idna.Lookup.ToASCII(tt.unicode)
			assert.False(t, IsValidDomain(ascii),
				"IsValidDomain(%q) should reject structurally invalid IDNA output", ascii)
		})
	}
}

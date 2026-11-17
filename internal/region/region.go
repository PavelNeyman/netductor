// Package region validates and formats fleet placement labels (CF-14 / fleet plan).
package region

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Preferred form: iso3166-1-alpha2 + city slug (ru-msk, fi-hel).
// Legacy short labels (ru, eu) remain valid during transition.

var (
	reCCCity = regexp.MustCompile(`(?i)^[a-z]{2}-[a-z0-9]{2,24}$`)
	reShort  = regexp.MustCompile(`(?i)^[a-z]{2,8}$`)
	// nd-<role>-<region> with region = cc-city (role tokens fixed set)
	reHost = regexp.MustCompile(`(?i)^nd-(control|egress|build|edge|primary|secondary)-([a-z]{2}-[a-z0-9]{2,24})(?:-|$|\.)`)
)

// Normalize lowercases and trims; empty stays empty.
func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Valid reports whether s is an accepted region label (or empty).
func Valid(s string) bool {
	s = Normalize(s)
	if s == "" {
		return true
	}
	if reCCCity.MatchString(s) {
		return true
	}
	// legacy short: letters only, 2–8
	if reShort.MatchString(s) {
		for _, r := range s {
			if !unicode.IsLetter(r) {
				return false
			}
		}
		return true
	}
	return false
}

// Parse validates and returns normalized region or error if non-empty invalid.
func Parse(s string) (string, error) {
	s = Normalize(s)
	if s == "" {
		return "", nil
	}
	if !Valid(s) {
		return "", fmt.Errorf("invalid region %q (want cc-city e.g. ru-msk or short legacy e.g. ru)", s)
	}
	return s, nil
}

// CountryCode extracts iso alpha-2 from cc-city or short label.
func CountryCode(s string) string {
	s = Normalize(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '-'); i == 2 {
		return s[:2]
	}
	if len(s) >= 2 && reShort.MatchString(s) {
		return s[:2]
	}
	return ""
}

// FlagEmoji returns regional-indicator flag for the country code, or empty.
func FlagEmoji(s string) string {
	cc := CountryCode(s)
	if len(cc) != 2 {
		return ""
	}
	a := rune(cc[0])
	b := rune(cc[1])
	if a < 'a' || a > 'z' || b < 'a' || b > 'z' {
		return ""
	}
	// Regional Indicator Symbol Letter A is U+1F1E6
	return string([]rune{0x1F1E6 + (a - 'a'), 0x1F1E6 + (b - 'a')})
}

// ProfileFragment builds a URI fragment label for subscription/VLESS.
// Prefer region (+ flag); fall back to ip-based legacy label.
func ProfileFragment(region, publicIP string, index int) string {
	region = Normalize(region)
	if region != "" && Valid(region) {
		if fl := FlagEmoji(region); fl != "" {
			return fl + "-" + region
		}
		return "nd-" + region
	}
	label := strings.NewReplacer(".", "-", ":", "-", "[", "", "]", "", "%", "-").
		Replace(strings.TrimSpace(publicIP))
	label = strings.Trim(label, "-")
	if label == "" {
		label = fmt.Sprintf("%d", index+1)
	}
	return "nd-secondary-" + label
}

// FromHostname extracts region from nd-<role>-<cc-city>… hostnames.
func FromHostname(hostname string) string {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "" {
		return ""
	}
	// strip domain
	if i := strings.IndexByte(hostname, '.'); i > 0 {
		hostname = hostname[:i]
	}
	m := reHost.FindStringSubmatch(hostname)
	if len(m) >= 3 {
		return Normalize(m[2])
	}
	return ""
}

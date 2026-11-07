// Package role normalizes node role names (CF-14).
// Canonical: control | egress | build | edge
// Legacy aliases: primary≡control, secondary≡egress (dual-read forever).
package role

import (
	"os"
	"strings"
)

const (
	Control  = "control"
	Egress   = "egress"
	Build    = "build"
	Edge     = "edge"
	Primary  = "primary"   // legacy alias of control
	Secondary = "secondary" // legacy alias of egress
)

// Normalize maps legacy and canonical names to canonical role.
// Unknown non-empty values are returned lowercased as-is (caller may reject).
func Normalize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case Primary, Control, "ctrl":
		return Control
	case Secondary, Egress, "relay":
		return Egress
	case Build, Edge:
		return s
	default:
		return s
	}
}

// IsControl reports control-plane role (primary or control).
func IsControl(s string) bool { return Normalize(s) == Control }

// IsEgress reports egress/VPN-entry role (secondary or egress).
func IsEgress(s string) bool { return Normalize(s) == Egress }

// AcceptsRoleFile returns true for values allowed in /etc/netductor/role today
// (legacy primary|secondary plus canonical control|egress).
func AcceptsRoleFile(s string) bool {
	n := Normalize(s)
	return n == Control || n == Egress
}

// LegacyWrite returns the on-disk value for dual-write period (still primary|secondary
// so older binaries keep working). Canonical control→primary, egress→secondary.
func LegacyWrite(s string) string {
	switch Normalize(s) {
	case Control:
		return Primary
	case Egress:
		return Secondary
	default:
		return strings.TrimSpace(strings.ToLower(s))
	}
}

// ReadFile reads /etc/netductor/role (or override path) and returns normalized role.
func ReadFile(path string) (normalized string, raw string, err error) {
	if path == "" {
		path = "/etc/netductor/role"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	raw = strings.TrimSpace(string(b))
	return Normalize(raw), raw, nil
}

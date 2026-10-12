package vpn

import (
	"strings"
	"testing"
)

func TestOperatorSRIncludesWorkDomains(t *testing.T) {
	s := BuildShadowrocketRoutingConfProfile(SRProfileOperatorMobile)
	for _, want := range []string{
		"bypass-system = true",
		"dns-direct-fallback-proxy = false",
		"udp-policy-not-supported-behaviour = DIRECT",
		"DOMAIN-SUFFIX,tealpos.ru,DIRECT",
		"192.168.70.250/32",
		"*.tealpos.ru",
		"tealpos.ru",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s[:min(800, len(s))])
		}
	}
	// skip-proxy line should list domain
	idx := strings.Index(s, "skip-proxy =")
	if idx < 0 {
		t.Fatal("no skip-proxy")
	}
	line := s[idx:]
	if end := strings.Index(line, "\n"); end > 0 {
		line = line[:end]
	}
	if !strings.Contains(line, "tealpos.ru") {
		t.Fatalf("skip-proxy missing tealpos: %s", line)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

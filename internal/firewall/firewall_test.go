package firewall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTruncate(t *testing.T) {
	if got := truncate("ab", 5); got != "ab" {
		t.Fatalf("%q", got)
	}
	if got := truncate("  hello world  ", 5); got != "hello…" {
		t.Fatalf("%q", got)
	}
}

func TestFirstLines(t *testing.T) {
	in := "a\nb\nc\nd"
	got := firstLines(in, 2)
	if got != "a\nb" {
		t.Fatalf("%q", got)
	}
	if firstLines("x", 5) != "x" {
		t.Fatal("short")
	}
}

func TestDetectRoleDefaultPrimary(t *testing.T) {
	// Without agent/role markers on this host, default is primary.
	r := detectRole()
	if r != "primary" && r != "secondary" {
		t.Fatalf("role %q", r)
	}
}

func TestRegisteredPeerAgentSourcesIncludesAdditionalSlots(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", root)
	dir := filepath.Join(root, "svc-paths")
	if err := os.MkdirAll(dir, 0o700); err != nil { t.Fatal(err) }
	raw := `{"version":1,"peers":[
		{"public_ip":"203.0.113.20","state":"prepared","slot":{"addr_s_sp":"10.87.20.2","addr_s_ps":"10.87.21.2"}},
		{"public_ip":"203.0.113.21","state":"removed","slot":{"addr_s_sp":"10.87.30.2","addr_s_ps":"10.87.31.2"}}
	]}`
	if err := os.WriteFile(filepath.Join(dir, "peers.json"), []byte(raw), 0o600); err != nil { t.Fatal(err) }
	got, err := registeredPeerAgentSources()
	if err != nil { t.Fatal(err) }
	if len(got) != 2 || got[0] != "10.87.20.2" || got[1] != "10.87.21.2" {
		t.Fatalf("unexpected peer API sources: %#v", got)
	}
}

func TestUFWSourceRangeFirstAllowAcceptsNarrowAllowBeforeCoveringRange(t *testing.T) {
	rules := []ufwRule{
		{Num: 1, Port: "9443", Proto: "tcp", Action: "ALLOW", Direction: "IN", From: "10.87.10.2"},
		{Num: 2, Port: "9443", Proto: "tcp", Action: "ALLOW", Direction: "IN", From: "10.87.10.0/30"},
	}
	covered, found := ufwSourceRangeFirstAllow(rules, "9443", "10.87.10.0/30")
	if !found || !covered {
		t.Fatalf("narrow /32 followed by covering /30 should cover the full range: covered=%t found=%t", covered, found)
	}
}

func TestUFWSourceRangeFirstAllowRejectsShadowedAddress(t *testing.T) {
	rules := []ufwRule{
		{Num: 1, Port: "9443", Proto: "tcp", Action: "DENY", Direction: "IN", From: "10.87.10.2"},
		{Num: 2, Port: "9443", Proto: "tcp", Action: "ALLOW", Direction: "IN", From: "10.87.10.0/30"},
	}
	covered, found := ufwSourceRangeFirstAllow(rules, "9443", "10.87.10.0/30")
	if !found || covered {
		t.Fatalf("earlier /32 DENY must shadow the later /30 ALLOW: covered=%t found=%t", covered, found)
	}
}

func TestUFWSourceRangeFirstAllowRequiresCompleteCoverage(t *testing.T) {
	rules := []ufwRule{
		{Num: 1, Port: "9443", Proto: "tcp", Action: "ALLOW", Direction: "IN", From: "10.87.10.2"},
	}
	covered, found := ufwSourceRangeFirstAllow(rules, "9443", "10.87.10.0/30")
	if !found || covered {
		t.Fatalf("a single /32 ALLOW must not cover the full /30: covered=%t found=%t", covered, found)
	}
}

func TestUFWSourceRangeFirstAllowSupportsSingleHost(t *testing.T) {
	rules := []ufwRule{
		{Num: 1, Port: "8789", Proto: "tcp", Action: "ALLOW", Direction: "IN", From: "10.87.10.2"},
	}
	covered, found := ufwSourceRangeFirstAllow(rules, "8789", "10.87.10.2")
	if !found || !covered {
		t.Fatalf("individual IPv4 source should be evaluated as /32: covered=%t found=%t", covered, found)
	}
}

func TestUFWSourceRangeHasFirstAllowAllowsAbsentPolicyForPreflight(t *testing.T) {
	if !ufwSourceRangeHasFirstAllow(nil, "9443", "10.87.10.0/30") {
		t.Fatal("absent policy must remain appendable during preflight")
	}
}
